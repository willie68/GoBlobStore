package migration

import (
	"fmt"
	"io"
	"sync"

	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
)

type BackupCheck struct {
	Tenant string
}

var wg sync.WaitGroup

// MigrateBackup migrates all blobs in the main storage for all tenants into the backup storage, if not already present
func MigrateBackup(tenantDao interfaces.TenantDao, stgf interfaces.StorageFactory) error {
	err := tenantDao.GetTenants(func(t string) bool {
		log.Logger.Debugf("BckMgr: found tenant: %s", t)
		stg, err := stgf.GetStorageDao(t)
		if err != nil {
			return true
		}
		mainstg, ok := stg.(*business.MainStorageDao)
		if ok {
			if mainstg.BckDao == nil {
				log.Logger.Debugf("no backstorage found for tenant %s", t)
				return true
			}
			go migrateBckTnt(mainstg.StgDao, mainstg.BckDao)
		}
		return true
	})
	if err != nil {
		return err
	}
	return nil
}

// migrateBckTnt migrates all files from the main storage of the teant to the backup storage
func migrateBckTnt(stg interfaces.BlobStorageDao, bck interfaces.BlobStorageDao) error {
	log.Logger.Infof("starting backup migration for tenant: %s", stg.GetTenant())
	stg.GetBlobs(func(id string) bool {
		found, err := bck.HasBlob(id)
		if err != nil {
			log.Logger.Errorf("error checking blob from backup storage %s: %s\r\n%v ", bck.GetTenant(), id, err)
		}
		if !found {
			log.Logger.Infof("migrating file for tenant: %s, %s", stg.GetTenant(), id)
			wg.Add(1)
			go func() {
				defer wg.Done()
				backup(id, stg, bck)
			}()
		}
		return true
	})
	return nil
}

// backup migrates a file from the main storage of the tenant to the backup storage
func backup(id string, stg interfaces.BlobStorageDao, bck interfaces.BlobStorageDao) error {
	found, err := stg.HasBlob(id)
	if err != nil {
		log.Logger.Errorf("error checking blob: %s\n%v", id, err)
		return err
	}
	if found {
		b, err := stg.GetBlobDescription(id)
		if err != nil {
			log.Logger.Errorf("error checking blob: %s\n%v", id, err)
			return err
		}

		rd, wr := io.Pipe()

		go func() {
			// close the writer, so the reader knows there's no more data
			defer wr.Close()

			err := stg.RetrieveBlob(id, wr)
			if err != nil {
				log.Logger.Errorf("error getting blob: %s,%v", id, err)
			}
		}()
		_, err = bck.StoreBlob(b, rd)
		if err != nil {
			log.Logger.Errorf("error getting blob: %s,%v", id, err)
		}
		defer rd.Close()
		if b.Retention > 0 {
			rt, err := stg.GetRetention(id)
			if err != nil {
				log.Logger.Errorf("error getting retention: %s,%v", id, err)
			} else {
				bck.AddRetention(&rt)
			}
		}
		return nil
	}
	return fmt.Errorf("blob not found: %s", id)
}
