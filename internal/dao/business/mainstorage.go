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

var _ interfaces.BlobStorageDao = &MainStorageDao{}

type MainStorageDao struct {
	RtnMng      interfaces.RetentionManager
	StgDao      interfaces.BlobStorageDao
	BckDao      interfaces.BlobStorageDao
	CchDao      interfaces.BlobStorageDao
	IdxDao      interfaces.Index
	Bcksyncmode bool
	Tenant      string
	hasIdx      bool
}

// Init initialise this dao
func (m *MainStorageDao) Init() error {
	// all storages should be initialised before adding to this business class
	// there for only specifig initialisation for this class is required
	m.hasIdx = m.IdxDao != nil
	return nil
}

// GetTenant return the id of the tenant
func (m *MainStorageDao) GetTenant() string {
	return m.Tenant
}

// GetBlobs getting a list of blob from the filesystem using offset and limit
func (m *MainStorageDao) GetBlobs(callback func(id string) bool) error {
	return m.StgDao.GetBlobs(callback)
}

// StoreBlob storing a blob to the storage system
func (m *MainStorageDao) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
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
	if m.BckDao != nil {
		if m.Bcksyncmode {
			m.backupFile(b, id)
		} else {
			go m.backupFile(b, id)
		}
	}
	go m.cacheFile(b)
	gor := (runtime.NumGoroutine() / 1000)
	time.Sleep(time.Duration(gor) * time.Millisecond)
	return id, err
}

// updating the blob description
func (m *MainStorageDao) UpdateBlobDescription(id string, b *model.BlobDescription) error {
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

func (m *MainStorageDao) cacheFileByID(id string) {
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

func (m *MainStorageDao) cacheFile(b *model.BlobDescription) {
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

func (m *MainStorageDao) backupFile(b *model.BlobDescription, id string) {
	if m.BckDao != nil {
		ok, err := m.BckDao.HasBlob(b.BlobID)
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
			if _, err := m.BckDao.StoreBlob(b, rd); err != nil {
				log.Logger.Errorf("main: backupFile: store, error getting blob: %s, %v", id, err)
			}
		}
	}
}

func (m *MainStorageDao) restoreFile(b *model.BlobDescription) {
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
func (m *MainStorageDao) HasBlob(id string) (bool, error) {
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
func (m *MainStorageDao) GetBlobDescription(id string) (*model.BlobDescription, error) {
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
func (m *MainStorageDao) RetrieveBlob(id string, w io.Writer) error {
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
func (m *MainStorageDao) DeleteBlob(id string) error {
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

func (m *MainStorageDao) SearchBlobs(q string, callback func(id string) bool) error {
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
func (m *MainStorageDao) CheckBlob(id string) (*model.CheckInfo, error) {
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
		Storage: stgCI,
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

//GetAllRetentions for every retention entry for this Tenant we call this this function, you can stop the listing by returnong a false
func (m *MainStorageDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return m.StgDao.GetAllRetentions(callback)
}

func (m *MainStorageDao) AddRetention(r *model.RetentionEntry) error {
	err := m.StgDao.AddRetention(r)
	if m.BckDao != nil {
		if err1 := m.BckDao.AddRetention(r); err1 != nil {
			log.Logger.Errorf("error adding retention on backup: %v", err1)
		}
	}
	return err
}

func (m *MainStorageDao) GetRetention(id string) (model.RetentionEntry, error) {
	return m.StgDao.GetRetention(id)
}

func (m *MainStorageDao) DeleteRetention(id string) error {
	err := m.StgDao.DeleteRetention(id)
	if m.BckDao != nil {
		if err1 := m.BckDao.DeleteRetention(id); err1 != nil {
			log.Logger.Errorf("error deleting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}

func (m *MainStorageDao) ResetRetention(id string) error {
	err := m.StgDao.ResetRetention(id)
	if m.BckDao != nil {
		if err1 := m.BckDao.ResetRetention(id); err1 != nil {
			log.Logger.Errorf("error reseting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}

// Close closing the blob storage
func (m *MainStorageDao) Close() error {
	err := m.StgDao.Close()
	if m.BckDao != nil {
		if err1 := m.BckDao.Close(); err1 != nil {
			log.Logger.Errorf("error closing backup storage: %v", err1)
		}
	}
	return err
}
