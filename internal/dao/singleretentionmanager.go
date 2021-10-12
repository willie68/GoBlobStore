package dao

import (
	"sort"

	clog "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// SingleRetentionManager is a single node retention manager
// It will periodically browse thru all tenants and there to all retentions files, to get a list of all retention entries for the next hour.
// Than it will sort this list and process the retention entries
type SingleRetentionManager struct {
	tntDao        TenantDao
	retentionList []model.RetentionEntry
	maxSize       int
}

// Init initialise the retention manager, creating the list of retention entries
func (s *SingleRetentionManager) Init() error {
	s.retentionList = make([]model.RetentionEntry, 0)
	tenants, err := s.tntDao.GetTenants()
	if err != nil {
		return err
	}
	for _, t := range tenants {
		clog.Logger.Debugf("RetMgr: found tenant: %s", t)
		dao, err := GetStorageDao(t)
		if err != nil {
			return err
		}
		err = dao.GetAllRetentions(func(r model.RetentionEntry) bool {
			s.pushToList(r)
			return true
		})
		if err != nil {
			return err
		}
	}
	return nil
}

//pushToList adding a new retention to the retention list, if fits
func (s *SingleRetentionManager) pushToList(r model.RetentionEntry) {
	s.retentionList = append(s.retentionList, r)
	i := sort.Search(len(s.retentionList), func(i int) bool {
		return s.retentionList[i].GetRetentionTimestamp() > r.GetRetentionTimestamp()
	})
	s.retentionList = insertAt(s.retentionList, i, r)
	if len(s.retentionList) > s.maxSize {
		s.retentionList = s.retentionList[:s.maxSize-1]
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

//AddRetention adding a new retention to the retention manager
func (s *SingleRetentionManager) AddRetention(tenant string, b *model.BlobDescription) error {
	if b.Retention > 0 {
		stgDao, err := GetStorageDao(tenant)
		if err != nil {
			return err
		}
		re := model.RetentionEntry{
			BlobID:        b.BlobID,
			CreationDate:  b.CreationDate,
			Filename:      b.Filename,
			Retention:     b.Retention,
			RetentionBase: 0,
			TenantID:      tenant,
		}
		err = stgDao.AddRetention(&re)
		if err != nil {
			return err
		}
		s.pushToList(re)
	}
	return nil
}
