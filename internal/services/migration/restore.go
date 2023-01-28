package migration

import (
	"errors"
	"fmt"
	"io"
	"time"

	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/services/business"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

// RestoreContext struct for the running a full restore of the tenant
type RestoreContext struct {
	TenantID  string
	ID        string
	Started   time.Time
	Finnished time.Time
	Primary   interfaces.BlobStorage
	Backup    interfaces.BlobStorage
	Running   bool
	cancel    bool
}

// checking interface compatibility
var _ interfaces.Running = &RestoreContext{}

// MigrateRestore migrates all blobs in the backup storage for a tenant into the main storage, if not already present
func MigrateRestore(tenant string, stgf interfaces.StorageFactory) (*RestoreContext, error) {
	d, err := stgf.GetStorage(tenant)
	if err != nil {
		return nil, err
	}

	main, ok := d.(*business.MainStorage)
	if !ok {
		return nil, errors.New("wrong storage class for check")
	}
	r := RestoreContext{
		TenantID: tenant,
		Primary:  main.StgSrv,
		Backup:   main.BckSrv,
		Running:  false,
	}
	go r.Restore()
	return &r, nil
}

// Restore starting a full restore of a tenant
func (r *RestoreContext) Restore() {
	r.Running = true
	defer func() {
		r.Running = false
	}()
	r.cancel = false
	log.Logger.Debugf("start restoring tenant \"%s\"", r.TenantID)
	// restoring all blobs in backup storage
	if r.Backup != nil {
		log.Logger.Debug("checking backup")
		count := 0
		err := r.Backup.GetBlobs(func(id string) bool {
			// process only blobs that are not already in primary store
			if ok, _ := r.Primary.HasBlob(id); !ok {
				err := restore(id, r.Backup, r.Primary)
				if err != nil {
					log.Logger.Errorf("error restoring file from backup: %v", err)
				}
			}
			count++
			return true
		})
		if err != nil {
			log.Logger.Errorf("error getting files from backup: %v", err)
		}
	}
}

// IsRunning checking if a full restore is running
func (r *RestoreContext) IsRunning() bool {
	return r.Running
}

// restore migrates a file from the backup storage of the tenant to the primary storage
func restore(id string, src interfaces.BlobStorage, dst interfaces.BlobStorage) error {
	found, err := src.HasBlob(id)
	if err != nil {
		log.Logger.Errorf("error checking blob: %s\n%v", id, err)
		return err
	}
	if found {
		b, err := src.GetBlobDescription(id)
		if err != nil {
			log.Logger.Errorf("error checking blob: %s\n%v", id, err)
			return err
		}

		rd, wr := io.Pipe()

		go func() {
			// close the writer, so the reader knows there's no more data
			defer wr.Close()

			err := src.RetrieveBlob(id, wr)
			if err != nil {
				log.Logger.Errorf("error getting blob: %s,%v", id, err)
			}
		}()
		_, err = dst.StoreBlob(b, rd)
		if err != nil {
			log.Logger.Errorf("error getting blob: %s,%v", id, err)
		}
		defer rd.Close()
		if b.Retention > 0 {
			rt, err := src.GetRetention(id)
			if err != nil {
				log.Logger.Errorf("error getting retention: %s,%v", id, err)
			} else {
				dst.AddRetention(&rt)
			}
		}
		return nil
	}
	return fmt.Errorf("blob not found: %s", id)
}
