package retentionmanager

import (
	"sort"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// SingleRetentionManagerName name of this retentionmananger
const SingleRetentionManagerName = "SingleRetention"

// SingleRetentionManager is a single node retention manager
// It will periodically browse thru all tenants and there to all retentions files, to get a list of all retention entries for the next hour.
// Than it will sort this list and process the retention entries
type SingleRetentionManager struct {
	TntDao        interfaces.TenantManager
	stgf          interfaces.StorageFactory
	retentionList []model.RetentionEntry
	MaxSize       int
	background    *time.Ticker
	quit          chan bool
}

// check interface compatibility
var _ interfaces.RetentionManager = &SingleRetentionManager{}

// Init initialise the retention manager, creating the list of retention entries
func (s *SingleRetentionManager) Init(stgf interfaces.StorageFactory) error {
	s.stgf = stgf
	s.retentionList = make([]model.RetentionEntry, 0)
	err := s.refereshRetention()
	if err != nil {
		log.Logger.Errorf("RetMgr: error on refresh: %v", err)
		return err
	}
	s.background = time.NewTicker(60 * time.Second)
	s.quit = make(chan bool)
	go func() {
		for {
			select {
			case <-s.background.C:
				err := s.processRetention()
				if err != nil {
					log.Logger.Errorf("RetMgr: error on process: %v", err)
				}
				err = s.refereshRetention()
				if err != nil {
					log.Logger.Errorf("RetMgr: error on refresh: %v", err)
				}
			case <-s.quit:
				s.background.Stop()
				return
			}
		}
	}()
	return nil
}

func (s *SingleRetentionManager) processRetention() error {
	actualTime := time.Now().Unix() * 1000
	rmvList := make([]string, 0)
	for _, v := range s.retentionList {
		if v.GetRetentionTimestampMS() < actualTime {
			//TODO maybe the retention entry has been changed (from another node), so please refresh the entry and check again
			rmvList = append(rmvList, v.BlobID)
			stg, err := s.stgf.GetStorage(v.TenantID)
			if err != nil {
				log.Logger.Errorf("RetMgr: error getting tenant store: %s", v.TenantID)
				continue
			}
			err = stg.DeleteBlob(v.BlobID)
			if err != nil {
				log.Logger.Errorf("RetMgr: error removing blob, t:%s, name: %s, id:%s", v.TenantID, v.Filename, v.BlobID)
				continue
			}
		}
	}
	for _, v := range rmvList {
		s.removeEntry(v)
	}
	return nil
}

func (s *SingleRetentionManager) removeEntry(id string) {
	var i int
	for x, v := range s.retentionList {
		if id == v.BlobID {
			i = x
			break
		}
	}
	if len(s.retentionList) > i {
		// Remove the element at index i from a.
		if i < len(s.retentionList)-1 {
			copy(s.retentionList[i:], s.retentionList[i+1:]) // Shift a[i+1:] left one index.
		}
		s.retentionList = s.retentionList[:len(s.retentionList)-1] // Truncate slice.
	}
}

func (s *SingleRetentionManager) refereshRetention() error {
	err := s.TntDao.GetTenants(func(t string) bool {
		log.Logger.Debugf("RetMgr: found tenant: %s", t)
		stg, err := s.stgf.GetStorage(t)
		if err != nil {
			return true
		}
		stg.GetAllRetentions(func(r model.RetentionEntry) bool {
			s.pushToList(r)
			return true
		})
		return true
	})
	if err != nil {
		return err
	}
	return nil
}

// pushToList adding a new retention to the retention list, if fits
func (s *SingleRetentionManager) pushToList(r model.RetentionEntry) {
	//s.retentionList = append(s.retentionList, r)
	for _, v := range s.retentionList {
		if r.BlobID == v.BlobID {
			return
		}
	}
	i := sort.Search(len(s.retentionList), func(i int) bool {
		return s.retentionList[i].GetRetentionTimestampMS() > r.GetRetentionTimestampMS()
	})
	if i < s.MaxSize {
		s.retentionList = insertAt(s.retentionList, i, r)
		if len(s.retentionList) > s.MaxSize {
			s.retentionList = s.retentionList[:s.MaxSize-1]
		}
	}
}

// insertAt inserts v into s at index i and returns the new slice.
func insertAt(data []model.RetentionEntry, i int, v model.RetentionEntry) []model.RetentionEntry {
	if i == len(data) {
		return append(data, v)
	}
	data = append(data[:i+1], data[i:]...)
	data[i] = v
	return data
}

// GetAllRetentions walk thru all blobs with retentions
func (s *SingleRetentionManager) GetAllRetentions(tenant string, callback func(r model.RetentionEntry) bool) error {
	stg, err := s.stgf.GetStorage(tenant)
	if err != nil {
		return err
	}
	stg.GetAllRetentions(func(r model.RetentionEntry) bool {
		callback(r)
		return true
	})
	return nil
}

// AddRetention adding a new retention to the retention manager
func (s *SingleRetentionManager) AddRetention(tenant string, r *model.RetentionEntry) error {
	if r.Retention > 0 {
		stg, err := s.stgf.GetStorage(tenant)
		if err != nil {
			return err
		}
		err = stg.AddRetention(r)
		if err != nil {
			return err
		}
		s.pushToList(*r)
	}
	return nil
}

// DeleteRetention deleting a retention
func (s *SingleRetentionManager) DeleteRetention(tenant string, id string) error {
	stg, err := s.stgf.GetStorage(tenant)
	if err != nil {
		return err
	}
	err = stg.DeleteRetention(id)
	if err != nil {
		return err
	}
	return nil
}

// ResetRetention resets the retention for a single blob
func (s *SingleRetentionManager) ResetRetention(tenant string, id string) error {
	stg, err := s.stgf.GetStorage(tenant)
	if err != nil {
		return err
	}
	err = stg.ResetRetention(id)
	if err != nil {
		return err
	}
	return nil
}

// Close closing this manager
func (s *SingleRetentionManager) Close() error {
	s.quit <- true
	return nil
}
