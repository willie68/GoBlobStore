// Package migration this package holds a async service for doing some tenant related migration tasks, like backup, restore, check
package migration

import (
	"fmt"
	"io"
	"sync"

	"github.com/willie68/GoBlobStore/internal/services/business"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

// BackupCheck the struct for doing a backup
type BackupCheck struct {
	Tenant string
}

var wg sync.WaitGroup

// MigrateBackup migrates all blobs in the main storage for all tenants into the backup storage, if not already present
func MigrateBackup(tntsrv interfaces.TenantManager, stgf interfaces.StorageFactory) error {
	err := tntsrv.GetTenants(func(t string) bool {
		logger.Debugf("BckMgr: found tenant: %s", t)
		stg, err := stgf.GetStorage(t)
		if err != nil {
			return true
		}
		mainstg, ok := stg.(*business.MainStorage)
		if ok {
			if mainstg.BckSrv == nil {
				logger.Debugf("no backstorage found for tenant %s", t)
				return true
			}
			go migrateBckTnt(mainstg.StgSrv, mainstg.BckSrv)
		}
		return true
	})
	if err != nil {
		return err
	}
	return nil
}

// migrateBckTnt migrates all files from the main storage of the tenant to the backup storage
func migrateBckTnt(stg interfaces.BlobStorage, bck interfaces.BlobStorage) error {
	logger.Infof("starting backup migration for tenant: %s", stg.GetTenant())
	stg.GetBlobs(func(id string) bool {
		found, err := bck.HasBlob(id)
		if err != nil {
			logger.Errorf("error checking blob from backup storage %s: %s\r\n%v ", bck.GetTenant(), id, err)
		}
		if !found {
			logger.Infof("migrating file for tenant: %s, %s", stg.GetTenant(), id)
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
func backup(id string, stg interfaces.BlobStorage, bck interfaces.BlobStorage) error {
	found, err := stg.HasBlob(id)
	if err != nil {
		logger.Errorf("error checking blob: %s\n%v", id, err)
		return err
	}
	if found {
		b, err := stg.GetBlobDescription(id)
		if err != nil {
			logger.Errorf("error checking blob: %s\n%v", id, err)
			return err
		}

		rd, wr := io.Pipe()

		go func() {
			// close the writer, so the reader knows there's no more data
			defer wr.Close()

			err := stg.RetrieveBlob(id, wr)
			if err != nil {
				logger.Errorf("error getting blob: %s,%v", id, err)
			}
		}()
		_, err = bck.StoreBlob(b, rd)
		if err != nil {
			logger.Errorf("error getting blob: %s,%v", id, err)
		}
		defer rd.Close()
		if b.Retention > 0 {
			rt, err := stg.GetRetention(id)
			if err != nil {
				logger.Errorf("error getting retention: %s,%v", id, err)
			} else {
				bck.AddRetention(&rt)
			}
		}
		return nil
	}
	return fmt.Errorf("blob not found: %s", id)
}
