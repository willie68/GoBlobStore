// Package business the package contains the structs for the business rules of the storage system
package business

// This type doing all the business logic of storing blobs of the service.
// Managing backup and cache requests, managing the Retentions
import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// testing interface compatibility
var _ interfaces.BlobStorage = &MainStorage{}

// MainStorage the main service for the business rules
type MainStorage struct {
	RtnMng      interfaces.RetentionManager
	StgSrv      interfaces.BlobStorage
	BckSrv      interfaces.BlobStorage
	CchSrv      interfaces.BlobStorage
	IdxSrv      interfaces.Index
	TntBckSrv   interfaces.BlobStorage
	TntMgr      interfaces.TenantManager
	Bcksyncmode bool
	Tenant      string
	hasIdx      bool
	TntError    error
}

// Init initialize this service
func (m *MainStorage) Init() error {
	// all storages should be initialized before adding to this business class
	// there for only specific initialization for this class is required
	m.hasIdx = m.IdxSrv != nil
	return nil
}

// GetTenant return the id of the tenant
func (m *MainStorage) GetTenant() string {
	return m.Tenant
}

// GetBlobs getting a list of blob from the filesystem using offset and limit
func (m *MainStorage) GetBlobs(callback func(id string) bool) error {
	return m.StgSrv.GetBlobs(callback)
}

// StoreBlob storing a blob to the storage system
func (m *MainStorage) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	hasBlob, err := m.StgSrv.HasBlob(b.BlobID)
	if err != nil {
		return "", fmt.Errorf("main: store blob: check blob: %s, %v", b.BlobID, err)
	}
	if hasBlob {
		return "", fmt.Errorf(`blob with id "%s" already exists`, b.BlobID)
	}

	id, err := m.StgSrv.StoreBlob(b, f)
	if err != nil {
		return "", err
	}
	b.BlobID = id
	if m.hasIdx {
		err = m.IdxSrv.Index(id, *b)
		if err != nil {
			return "", err
		}
	}
	if err == nil && m.RtnMng != nil {
		r := model.RetentionEntryFromBlobDescription(*b)
		err = m.RtnMng.AddRetention(m.Tenant, &r)
	}
	// main backup
	if m.BckSrv != nil {
		if m.Bcksyncmode {
			m.backupFile(b, id)
		} else {
			go m.backupFile(b, id)
		}
	}
	// tenant backup
	if m.TntBckSrv != nil {
		if m.Bcksyncmode {
			m.tntBackupFile(b, id)
		} else {
			go m.tntBackupFile(b, id)
		}
	}
	go m.cacheFile(b)
	go m.addStorageSize(id)
	gor := (runtime.NumGoroutine() / 1000)
	time.Sleep(time.Duration(gor) * time.Millisecond)
	return id, err
}

// UpdateBlobDescription updating the blob description
func (m *MainStorage) UpdateBlobDescription(id string, b *model.BlobDescription) error {
	err := m.StgSrv.UpdateBlobDescription(id, b)
	if err != nil {
		return err
	}
	if m.hasIdx {
		err = m.IdxSrv.Index(id, *b)
		if err != nil {
			return err
		}
	}
	if m.BckSrv != nil {
		if m.Bcksyncmode {
			err = m.BckSrv.UpdateBlobDescription(id, b)
			if err != nil {
				return err
			}
		} else {
			go m.BckSrv.UpdateBlobDescription(id, b)
		}
	}
	if m.CchSrv != nil {
		m.CchSrv.UpdateBlobDescription(id, b)
	}
	return nil
}

func (m *MainStorage) cacheFileByID(id string) {
	if m.CchSrv != nil {
		ok, err := m.CchSrv.HasBlob(id)
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
	if m.CchSrv != nil {
		ok, err := m.CchSrv.HasBlob(b.BlobID)
		if err != nil {
			log.Logger.Errorf("main: cacheFile: check blob: %s, %v", b.BlobID, err)
			return
		}
		if !ok {
			rd, wr := io.Pipe()
			go func() {
				defer wr.Close()
				if err := m.StgSrv.RetrieveBlob(b.BlobID, wr); err != nil {
					log.Logger.Errorf("main: cacheFile: retrieve, error getting blob: %s, %v", b.BlobID, err)
				}
				// close the writer, so the reader knows there's no more data
			}()
			defer rd.Close()
			if _, err := m.CchSrv.StoreBlob(b, rd); err != nil {
				log.Logger.Errorf("main: cacheFile: store, error getting blob: %s, %v", b.BlobID, err)
			}
		}
	}
}

func (m *MainStorage) tntBackupFile(b *model.BlobDescription, id string) {
	if m.TntBckSrv != nil {
		m.bckFile(m.TntBckSrv, b, id)
	}
}

func (m *MainStorage) backupFile(b *model.BlobDescription, id string) {
	if m.BckSrv != nil {
		m.bckFile(m.BckSrv, b, id)
	}
}

func (m *MainStorage) bckFile(srv interfaces.BlobStorage, b *model.BlobDescription, id string) {
	if srv != nil {
		ok, err := srv.HasBlob(b.BlobID)
		if err != nil {
			log.Logger.Errorf("main: backupFile: check blob: %s, %v", b.BlobID, err)
			return
		}
		if !ok {
			rd, wr := io.Pipe()
			go func() {
				// close the writer, so the reader knows there's no more data
				defer wr.Close()
				if err := m.StgSrv.RetrieveBlob(id, wr); err != nil {
					log.Logger.Errorf("main: backupFile: retrieve, error getting blob: %s, %v", id, err)
				}
			}()
			defer rd.Close()
			if _, err := srv.StoreBlob(b, rd); err != nil {
				log.Logger.Errorf("main: backupFile: store, error getting blob: %s, %v", id, err)
			}
		}
	}
}

func (m *MainStorage) restoreFile(b *model.BlobDescription) {
	if m.BckSrv != nil {
		id := b.BlobID
		ok, err := m.BckSrv.HasBlob(id)
		if err != nil {
			log.Logger.Errorf("main: restoreFile: check blob: %s, %v", id, err)
			return
		}
		if ok {
			rd, wr := io.Pipe()
			go func() {
				// close the writer, so the reader knows there's no more data
				defer wr.Close()
				if err := m.BckSrv.RetrieveBlob(id, wr); err != nil {
					log.Logger.Errorf("main: restoreFile: retrieve, error getting blob: %s, %v", id, err)
				}
			}()
			defer rd.Close()
			if _, err := m.StgSrv.StoreBlob(b, rd); err != nil {
				log.Logger.Errorf("main: restoreFile: store, error getting blob: %s, %v", id, err)
			}
			go m.cacheFileByID(id)
		}
	}
}

// addStorageSize adjust the storage size for the tenant
func (m *MainStorage) addStorageSize(id string) {
	bd, err := m.GetBlobDescription(id)
	if err != nil {
		log.Logger.Errorf("adjust: can't get blob description: %v", err)
		return
	}
	if m.TntMgr != nil {
		m.TntMgr.AddSize(m.Tenant, bd.ContentLength)
	}
}

// subStorageSize subtract the storage size for the tenant
func (m *MainStorage) subStorageSize(bd *model.BlobDescription) {
	if m.TntMgr != nil {
		m.TntMgr.SubSize(m.Tenant, bd.ContentLength)
	}
}

// HasBlob getting the description of the file
func (m *MainStorage) HasBlob(id string) (bool, error) {
	if m.CchSrv != nil {
		ok, err := m.CchSrv.HasBlob(id)
		if err == nil && ok {
			return true, nil
		}
	}
	ok, err := m.StgSrv.HasBlob(id)
	if (err != nil || !ok) && (m.BckSrv != nil) {
		bok, berr := m.BckSrv.HasBlob(id)
		if berr == nil && bok {
			bb, berr := m.BckSrv.GetBlobDescription(id)
			if berr == nil {
				go m.restoreFile(bb)
			}
			return true, nil
		}
	}
	return ok, err
}

// GetBlobDescription getting the description of the file
func (m *MainStorage) GetBlobDescription(id string) (*model.BlobDescription, error) {
	if m.CchSrv != nil {
		b, err := m.CchSrv.GetBlobDescription(id)
		if err == nil {
			if b.TenantID == m.Tenant {
				return b, nil
			}
		}
	}
	b, err := m.StgSrv.GetBlobDescription(id)
	if err != nil {
		if m.BckSrv != nil {
			bb, berr := m.BckSrv.GetBlobDescription(id)
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
	// check cache
	ok := m.retrieveFromCache(id, w)
	if ok {
		return nil
	}

	err := m.StgSrv.RetrieveBlob(id, w)
	if err == nil {
		go m.cacheFileByID(id)
		return nil
	}

	if m.BckSrv != nil {
		berr := m.BckSrv.RetrieveBlob(id, w)
		if berr == nil {
			if bb, berr := m.BckSrv.GetBlobDescription(id); berr == nil {
				go m.restoreFile(bb)
			}
			return nil
		}
		return err
	}
	return nil
}

func (m *MainStorage) retrieveFromCache(id string, w io.Writer) bool {
	if m.CchSrv != nil {
		if ok, _ := m.CchSrv.HasBlob(id); ok {
			b, err := m.CchSrv.GetBlobDescription(id)
			if err == nil && b.TenantID == m.Tenant {
				err := m.CchSrv.RetrieveBlob(id, w)
				if err == nil {
					return true
				}
			}
		}
	}
	return false
}

// DeleteBlob removing a blob from the storage system
func (m *MainStorage) DeleteBlob(id string) error {
	bd, err := m.StgSrv.GetBlobDescription(id)
	if err != nil {
		return err
	}
	err = m.StgSrv.DeleteBlob(id)
	if err != nil {
		return err
	}
	go m.subStorageSize(bd)
	if m.BckSrv != nil {
		if err = m.BckSrv.DeleteBlob(id); err != nil {
			log.Logger.Errorf("error deleting blob on backup: %v", err)
		}
	}
	if m.RtnMng != nil {
		m.RtnMng.DeleteRetention(m.Tenant, id)
	}
	if m.CchSrv != nil {
		if err = m.CchSrv.DeleteBlob(id); err != nil {
			log.Logger.Errorf("error deleting blob on cache: %v", err)
		}
	}
	return nil
}

// SearchBlobs if an index service is present, redirect the search to the index service
func (m *MainStorage) SearchBlobs(q string, callback func(id string) bool) error {
	if !m.hasIdx {
		return errors.New("index not configured")
	}

	err := m.IdxSrv.Search(q, callback)
	if err != nil {
		return err
	}
	return nil
}

// CheckBlob checking a single blob from the storage system
func (m *MainStorage) CheckBlob(id string) (*model.CheckInfo, error) {
	// check blob on main storage
	stgCI, err := m.StgSrv.CheckBlob(id)
	if err != nil {
		return nil, err
	}
	bd, err := m.StgSrv.GetBlobDescription(id)
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
	if m.BckSrv != nil {
		m.checkBck(id, &ri, bd)
	}
	bd.Check = &ri
	m.StgSrv.UpdateBlobDescription(id, bd)
	return stgCI, nil
}

func (m *MainStorage) checkBck(id string, ri *model.Check, bd *model.BlobDescription) {
	bckDI, err := m.BckSrv.CheckBlob(id)
	if err != nil {
		log.Logger.Errorf("error checking blob on backup: %v", err)
	}
	bckBd, err := m.BckSrv.GetBlobDescription(id)
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
	bckBd.Check = ri
	m.BckSrv.UpdateBlobDescription(id, bckBd)
}

// GetAllRetentions for every retention entry for this Tenant we call this this function, you can stop the listing by returning a false
func (m *MainStorage) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return m.StgSrv.GetAllRetentions(callback)
}

// AddRetention adding a retention entry to the main and backup storage
func (m *MainStorage) AddRetention(r *model.RetentionEntry) error {
	err := m.StgSrv.AddRetention(r)
	if m.BckSrv != nil {
		if err1 := m.BckSrv.AddRetention(r); err1 != nil {
			log.Logger.Errorf("error adding retention on backup: %v", err1)
		}
	}
	return err
}

// GetRetention getting a single retention entry from the main storage
func (m *MainStorage) GetRetention(id string) (model.RetentionEntry, error) {
	return m.StgSrv.GetRetention(id)
}

// DeleteRetention deletes the retention entry from the main and backup storage
func (m *MainStorage) DeleteRetention(id string) error {
	err := m.StgSrv.DeleteRetention(id)
	if m.BckSrv != nil {
		if err1 := m.BckSrv.DeleteRetention(id); err1 != nil {
			log.Logger.Errorf("error deleting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}

// ResetRetention resets the retention for a blob, main and backup storage
func (m *MainStorage) ResetRetention(id string) error {
	err := m.StgSrv.ResetRetention(id)
	if m.BckSrv != nil {
		if err1 := m.BckSrv.ResetRetention(id); err1 != nil {
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
	err := m.StgSrv.Close()
	if m.BckSrv != nil {
		if err1 := m.BckSrv.Close(); err1 != nil {
			log.Logger.Errorf("error closing backup storage: %v", err1)
		}
	}
	return err
}
