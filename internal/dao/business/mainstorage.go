// Package business the package contains the structs for the business rules of the storage system
package business

/*
This type doing all the business logic of storing blobs of the service.
Managing backup and cache requests, managing the Retentions
*/
import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// testing interface compatibility
var _ interfaces.BlobStorage = &MainStorage{}

// MainStorage the main dao for the business rules
type MainStorage struct {
	RtnMng      interfaces.RetentionManager
	StgDao      interfaces.BlobStorage
	BckDao      interfaces.BlobStorage
	CchDao      interfaces.BlobStorage
	IdxDao      interfaces.Index
	TntBckDao   interfaces.BlobStorage
	Bcksyncmode bool
	Tenant      string
	hasIdx      bool
	TntError    error
}

// Init initialize this dao
func (m *MainStorage) Init() error {
	// all storages should be initialized before adding to this business class
	// there for only specific initialization for this class is required
	m.hasIdx = m.IdxDao != nil
	return nil
}

// GetTenant return the id of the tenant
func (m *MainStorage) GetTenant() string {
	return m.Tenant
}

// GetBlobs getting a list of blob from the filesystem using offset and limit
func (m *MainStorage) GetBlobs(callback func(id string) bool) error {
	return m.StgDao.GetBlobs(callback)
}

// StoreBlob storing a blob to the storage system
func (m *MainStorage) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	hasBlob, err := m.StgDao.HasBlob(b.BlobID)
	if err != nil {
		return "", fmt.Errorf("main: store blob: check blob: %s, %v", b.BlobID, err)
	}
	if hasBlob {
		return "", fmt.Errorf(`blob with id "%s" already exists`, b.BlobID)
	}

	id, err := m.StgDao.StoreBlob(b, f)
	if err != nil {
		return "", err
	}
	b.BlobID = id
	if m.hasIdx {
		err = m.IdxDao.Index(id, *b)
		if err != nil {
			return "", err
		}
	}
	if err == nil && m.RtnMng != nil {
		r := model.RetentionEntryFromBlobDescription(*b)
		err = m.RtnMng.AddRetention(m.Tenant, &r)
	}
	// main backup
	if m.BckDao != nil {
		if m.Bcksyncmode {
			m.backupFile(b, id)
		} else {
			go m.backupFile(b, id)
		}
	}
	// tenant backup
	if m.TntBckDao != nil {
		if m.Bcksyncmode {
			m.tntBackupFile(b, id)
		} else {
			go m.tntBackupFile(b, id)
		}
	}
	go m.cacheFile(b)
	gor := (runtime.NumGoroutine() / 1000)
	time.Sleep(time.Duration(gor) * time.Millisecond)
	return id, err
}

// UpdateBlobDescription updating the blob description
func (m *MainStorage) UpdateBlobDescription(id string, b *model.BlobDescription) error {
	err := m.StgDao.UpdateBlobDescription(id, b)
	if err != nil {
		return err
	}
	if m.hasIdx {
		err = m.IdxDao.Index(id, *b)
		if err != nil {
			return err
		}
	}
	if m.BckDao != nil {
		if m.Bcksyncmode {
			err = m.BckDao.UpdateBlobDescription(id, b)
			if err != nil {
				return err
			}
		} else {
			go m.BckDao.UpdateBlobDescription(id, b)
		}
	}
	if m.CchDao != nil {
		m.CchDao.UpdateBlobDescription(id, b)
	}
	return nil
}

func (m *MainStorage) cacheFileByID(id string) {
	if m.CchDao != nil {
		ok, err := m.CchDao.HasBlob(id)
		if err != nil {
			log.Logger.Errorf("main: cacheFileByID: check blob: %s, %v", id, err)
			return
		}
		if !ok {
			b, err := m.GetBlobDescription(id)
			if err != nil {
				log.Logger.Errorf("main: cacheFileByID: getDescription: %s, %v", id, err)
				return
			}
			m.cacheFile(b)
		}
	}
}

func (m *MainStorage) cacheFile(b *model.BlobDescription) {
	if m.CchDao != nil {
		ok, err := m.CchDao.HasBlob(b.BlobID)
		if err != nil {
			log.Logger.Errorf("main: cacheFile: check blob: %s, %v", b.BlobID, err)
			return
		}
		if !ok {
			rd, wr := io.Pipe()
			go func() {
				defer wr.Close()
				if err := m.StgDao.RetrieveBlob(b.BlobID, wr); err != nil {
					log.Logger.Errorf("main: cacheFile: retrieve, error getting blob: %s, %v", b.BlobID, err)
				}
				// close the writer, so the reader knows there's no more data
			}()
			defer rd.Close()
			if _, err := m.CchDao.StoreBlob(b, rd); err != nil {
				log.Logger.Errorf("main: cacheFile: store, error getting blob: %s, %v", b.BlobID, err)
			}
		}
	}
}

func (m *MainStorage) tntBackupFile(b *model.BlobDescription, id string) {
	if m.TntBckDao != nil {
		m.bckFile(m.TntBckDao, b, id)
	}
}

func (m *MainStorage) backupFile(b *model.BlobDescription, id string) {
	if m.BckDao != nil {
		m.bckFile(m.BckDao, b, id)
	}
}

func (m *MainStorage) bckFile(dao interfaces.BlobStorage, b *model.BlobDescription, id string) {
	if dao != nil {
		ok, err := dao.HasBlob(b.BlobID)
		if err != nil {
			log.Logger.Errorf("main: backupFile: check blob: %s, %v", b.BlobID, err)
			return
		}
		if !ok {
			rd, wr := io.Pipe()
			go func() {
				// close the writer, so the reader knows there's no more data
				defer wr.Close()
				if err := m.StgDao.RetrieveBlob(id, wr); err != nil {
					log.Logger.Errorf("main: backupFile: retrieve, error getting blob: %s, %v", id, err)
				}
			}()
			defer rd.Close()
			if _, err := dao.StoreBlob(b, rd); err != nil {
				log.Logger.Errorf("main: backupFile: store, error getting blob: %s, %v", id, err)
			}
		}
	}
}

func (m *MainStorage) restoreFile(b *model.BlobDescription) {
	if m.BckDao != nil {
		id := b.BlobID
		ok, err := m.BckDao.HasBlob(id)
		if err != nil {
			log.Logger.Errorf("main: restoreFile: check blob: %s, %v", id, err)
			return
		}
		if ok {
			rd, wr := io.Pipe()
			go func() {
				// close the writer, so the reader knows there's no more data
				defer wr.Close()
				if err := m.BckDao.RetrieveBlob(id, wr); err != nil {
					log.Logger.Errorf("main: restoreFile: retrieve, error getting blob: %s, %v", id, err)
				}
			}()
			defer rd.Close()
			if _, err := m.StgDao.StoreBlob(b, rd); err != nil {
				log.Logger.Errorf("main: restoreFile: store, error getting blob: %s, %v", id, err)
			}
		}
	}
}

// HasBlob getting the description of the file
func (m *MainStorage) HasBlob(id string) (bool, error) {
	if m.CchDao != nil {
		ok, err := m.CchDao.HasBlob(id)
		if err == nil && ok {
			return true, nil
		}
	}
	ok, err := m.StgDao.HasBlob(id)
	if err != nil || !ok {
		if m.BckDao != nil {
			bok, berr := m.BckDao.HasBlob(id)
			if berr == nil && bok {
				bb, berr := m.BckDao.GetBlobDescription(id)
				if berr == nil {
					go m.restoreFile(bb)
				}
				return true, nil
			}
		}
	}
	return ok, err
}

// GetBlobDescription getting the description of the file
func (m *MainStorage) GetBlobDescription(id string) (*model.BlobDescription, error) {
	if m.CchDao != nil {
		b, err := m.CchDao.GetBlobDescription(id)
		if err == nil {
			if b.TenantID == m.Tenant {
				return b, nil
			}
		}
	}
	b, err := m.StgDao.GetBlobDescription(id)
	if err != nil {
		if m.BckDao != nil {
			bb, berr := m.BckDao.GetBlobDescription(id)
			if berr == nil {
				go m.restoreFile(bb)
				return bb, nil
			}
		}
	}
	return b, err
}

// RetrieveBlob retrieving the binary data from the storage system
func (m *MainStorage) RetrieveBlob(id string, w io.Writer) error {
	if m.CchDao != nil {
		ok, _ := m.CchDao.HasBlob(id)
		if ok {
			b, err := m.CchDao.GetBlobDescription(id)
			if err == nil {
				if b.TenantID == m.Tenant {
					err := m.CchDao.RetrieveBlob(id, w)
					if err == nil {
						return nil
					}
				}
			}
		}
	}
	err := m.StgDao.RetrieveBlob(id, w)
	if err != nil {
		if m.BckDao != nil {
			berr := m.BckDao.RetrieveBlob(id, w)
			if berr == nil {
				bb, berr := m.BckDao.GetBlobDescription(id)
				if berr == nil {
					go m.restoreFile(bb)
				}
				return nil
			}
		}
		return err
	}

	go m.cacheFileByID(id)
	return nil
}

// DeleteBlob removing a blob from the storage system
func (m *MainStorage) DeleteBlob(id string) error {
	err := m.StgDao.DeleteBlob(id)
	if err != nil {
		return err
	}
	if m.BckDao != nil {
		if err = m.BckDao.DeleteBlob(id); err != nil {
			log.Logger.Errorf("error deleting blob on backup: %v", err)
		}
	}
	if m.RtnMng != nil {
		m.RtnMng.DeleteRetention(m.Tenant, id)
	}
	if m.CchDao != nil {
		if err = m.CchDao.DeleteBlob(id); err != nil {
			log.Logger.Errorf("error deleting blob on cache: %v", err)
		}
	}
	return nil
}

// SearchBlobs if an index dao is present, redirect the search to the index service
func (m *MainStorage) SearchBlobs(q string, callback func(id string) bool) error {
	if !m.hasIdx {
		return errors.New("index not configured")
	}

	err := m.IdxDao.Search(q, callback)
	if err != nil {
		return err
	}
	return nil
}

// CheckBlob checking a single blob from the storage system
func (m *MainStorage) CheckBlob(id string) (*model.CheckInfo, error) {
	// check blob on main storage
	stgCI, err := m.StgDao.CheckBlob(id)
	if err != nil {
		return nil, err
	}
	bd, err := m.StgDao.GetBlobDescription(id)
	if err != nil {
		return nil, err
	}
	ri := model.Check{
		Store:   stgCI,
		Healthy: stgCI.Healthy,
		Message: stgCI.Message,
	}
	bd.Check = &ri
	// check blob on backup storage
	if m.BckDao != nil {
		bckDI, err := m.BckDao.CheckBlob(id)
		if err != nil {
			log.Logger.Errorf("error checking blob on backup: %v", err)
		}
		bckBd, err := m.StgDao.GetBlobDescription(id)
		if err != nil {
			log.Logger.Errorf("error getting blob description on backup: %v", err)
		}
		// merge stgCI and bckCI
		ri.Backup = bckDI
		ri.Healthy = ri.Healthy && bckDI.Healthy
		msg := bckDI.Message
		if ri.Message != "" && msg != "" {
			msg = fmt.Sprintf("%s, %s", ri.Message, msg)
		}
		if msg != "" {
			ri.Message = msg
		}

		// checking if both hashes are equal
		if bd.Hash != bckBd.Hash {
			ri.Healthy = false
			msg := "hashes are not equal"
			if ri.Message != "" {
				msg = fmt.Sprintf("%s, %s", ri.Message, msg)
			}
			ri.Message = msg
		}
		bckBd.Check = &ri
		m.BckDao.UpdateBlobDescription(id, bckBd)
	}
	bd.Check = &ri
	m.StgDao.UpdateBlobDescription(id, bd)
	return stgCI, nil
}

// GetAllRetentions for every retention entry for this Tenant we call this this function, you can stop the listing by returning a false
func (m *MainStorage) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return m.StgDao.GetAllRetentions(callback)
}

// AddRetention adding a retention entry to the main and backup storage
func (m *MainStorage) AddRetention(r *model.RetentionEntry) error {
	err := m.StgDao.AddRetention(r)
	if m.BckDao != nil {
		if err1 := m.BckDao.AddRetention(r); err1 != nil {
			log.Logger.Errorf("error adding retention on backup: %v", err1)
		}
	}
	return err
}

// GetRetention getting a single retention entry from the main storage
func (m *MainStorage) GetRetention(id string) (model.RetentionEntry, error) {
	return m.StgDao.GetRetention(id)
}

// DeleteRetention deletes the retention entry from the main and backup storage
func (m *MainStorage) DeleteRetention(id string) error {
	err := m.StgDao.DeleteRetention(id)
	if m.BckDao != nil {
		if err1 := m.BckDao.DeleteRetention(id); err1 != nil {
			log.Logger.Errorf("error deleting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}

// ResetRetention resets the retention for a blob, main and backup storage
func (m *MainStorage) ResetRetention(id string) error {
	err := m.StgDao.ResetRetention(id)
	if m.BckDao != nil {
		if err1 := m.BckDao.ResetRetention(id); err1 != nil {
			log.Logger.Errorf("error resetting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}

// GetLastError Error get last error for this tenant
func (m *MainStorage) GetLastError() error {
	return m.TntError
}

// Close closing the blob storage
func (m *MainStorage) Close() error {
	err := m.StgDao.Close()
	if m.BckDao != nil {
		if err1 := m.BckDao.Close(); err1 != nil {
			log.Logger.Errorf("error closing backup storage: %v", err1)
		}
	}
	return err
}
