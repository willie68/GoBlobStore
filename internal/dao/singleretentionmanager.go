package dao

import (
	"errors"
	"sort"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	clog "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// SingleRetentionManager is a single node retention manager
// It will periodically browse thru all tenants and there to all retentions files, to get a list of all retention entries for the next hour.
// Than it will sort this list and process the retention entries
type SingleRetentionManager struct {
	tntDao        interfaces.TenantDao
	retentionList []model.RetentionEntry
	maxSize       int
	background    *time.Ticker
	quit          chan bool
}

var _ interfaces.RetentionManager = &SingleRetentionManager{}

// Init initialise the retention manager, creating the list of retention entries
func (s *SingleRetentionManager) Init() error {
	s.retentionList = make([]model.RetentionEntry, 0)
	err := s.refereshRetention()
	if err != nil {
		clog.Logger.Errorf("RetMgr: error on refresh: %v", err)
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
					clog.Logger.Errorf("RetMgr: error on process: %v", err)
				}
				err = s.refereshRetention()
				if err != nil {
					clog.Logger.Errorf("RetMgr: error on refresh: %v", err)
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
	for x, v := range s.retentionList {
		if v.GetRetentionTimestampMS() < actualTime {
			//TODO maybe the retention entry has been changed (from another node), so please refresh the entry and check again
			stg, err := GetStorageDao(v.TenantID)
			if err != nil {
				clog.Logger.Errorf("RetMgr: error getting tenant store: %s", v.TenantID)
				continue
			}
			err = stg.DeleteBlob(v.BlobID)
			if err != nil {
				clog.Logger.Errorf("RetMgr: error removing blob, t:%s, name: %s, id:%s", v.TenantID, v.Filename, v.BlobID)
				continue
			}
			err = stg.DeleteRetention(v.BlobID)
			if err != nil {
				clog.Logger.Errorf("RetMgr: error removing retention entry, t:%s, name: %s, id:%s", v.TenantID, v.Filename, v.BlobID)
				continue
			}
			s.removeEntry(x)
		}
	}
	return nil
}

func (s *SingleRetentionManager) removeEntry(i int) {
	if len(s.retentionList) > i {
		// Remove the element at index i from a.
		if i < len(s.retentionList)-1 {
			copy(s.retentionList[i:], s.retentionList[i+1:]) // Shift a[i+1:] left one index.
		}
		s.retentionList = s.retentionList[:len(s.retentionList)-1] // Truncate slice.
	}
}

func (s *SingleRetentionManager) refereshRetention() error {
	err := s.tntDao.GetTenants(func(t string) bool {
		clog.Logger.Debugf("RetMgr: found tenant: %s", t)
		stg, err := GetStorageDao(t)
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

//pushToList adding a new retention to the retention list, if fits
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
	if i < s.maxSize {
		s.retentionList = insertAt(s.retentionList, i, r)
		if len(s.retentionList) > s.maxSize {
			s.retentionList = s.retentionList[:s.maxSize-1]
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

func (s *SingleRetentionManager) GetAllRetentions(tenant string, callback func(r model.RetentionEntry) bool) error {
	return errors.New("not implemented yet")
}

//AddRetention adding a new retention to the retention manager
func (s *SingleRetentionManager) AddRetention(tenant string, r *model.RetentionEntry) error {
	if r.Retention > 0 {
		stg, err := GetStorageDao(tenant)
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

func (s *SingleRetentionManager) DeleteRetention(tenant string, id string) error {
	return errors.New("not implemented yet")
}

func (s *SingleRetentionManager) ResetRetention(tenant string, id string) error {
	return errors.New("not implemented yet")
}

func (s *SingleRetentionManager) Close() error {
	s.quit <- true
	return nil
}
