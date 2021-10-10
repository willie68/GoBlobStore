package dao

import (
	"github.com/willie68/GoBlobStore/pkg/model"
)

// SingleRetentionManager is a single node retention manager
// It will periodically browse thru all tenants and there to all retentions files, to get a list of all retention entries for the next hour.
// Than it will sort this list and process the retention entries
type SingleRetentionManager struct {
	tntDao TenantDao
}

func (s *SingleRetentionManager) AddRetention(tenant string, b *model.BlobDescription) error {
	if b.Retention > 0 {
		stgDao, err := GetStorageDao(tenant)
		if err != nil {
			return err
		}
		err = stgDao.AddRetention(&model.RetentionEntry{
			BlobID:        b.BlobID,
			CreationDate:  b.CreationDate,
			Filename:      b.Filename,
			Retention:     b.Retention,
			RetentionBase: 0,
			TenantID:      tenant,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
