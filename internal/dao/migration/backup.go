package backup

import (
	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	clog "github.com/willie68/GoBlobStore/internal/logging"
)

type BackupCheck struct {
	Tenant string
}

func MigrateBackup(tenantDao interfaces.TenantDao, stgf interfaces.StorageFactory) error {

	err := tenantDao.GetTenants(func(t string) bool {
		clog.Logger.Debugf("BckMgr: found tenant: %s", t)
		stg, err := stgf.GetStorageDao(t)
		if err != nil {
			return true
		}
		mainstg, ok := stg.(*business.MainStorageDao)
		if ok {
			if mainstg.BckDao == nil {
				clog.Logger.Debugf("no backstorage found for tenant %s", t)
			}
		}
		return true
	})
	if err != nil {
		return err
	}
	return nil
}

func migrateBckTnt(stg interfaces.BlobStorageDao, bck interfaces.BlobStorageDao) error {
	return nil
}
