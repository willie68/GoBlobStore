package factory

import (
	"io"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	clog "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type mainStorageDao struct {
	rtnMng      interfaces.RetentionManager
	stgDao      interfaces.BlobStorageDao
	bckDao      interfaces.BlobStorageDao
	bcksyncmode bool
	tenant      string
}

// Init initialise this dao
func (m *mainStorageDao) Init() error {
	return m.stgDao.Init()
}

// GetBlobs getting a list of blob from the filesystem using offset and limit
func (m *mainStorageDao) GetBlobs(offset int, limit int) ([]string, error) {
	return m.stgDao.GetBlobs(offset, limit)
}

// StoreBlob storing a blob to the storage system
func (m *mainStorageDao) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	id, err := m.stgDao.StoreBlob(b, f)
	b.BlobID = id
	if err == nil && m.rtnMng != nil {
		r := model.RetentionEntryFromBlobDescription(*b)
		err = m.rtnMng.AddRetention(m.tenant, &r)
	}
	if m.bckDao != nil {
		if m.bcksyncmode {
			m.backupFile(b, id)
		} else {
			go m.backupFile(b, id)
		}
	}
	return id, err
}

func (m *mainStorageDao) backupFile(b *model.BlobDescription, id string) {
	rd, wr := io.Pipe()

	go func() {
		// close the writer, so the reader knows there's no more data
		defer wr.Close()

		err := m.stgDao.RetrieveBlob(id, wr)
		if err != nil {
			clog.Logger.Errorf("error getting blob: %s,%v", id, err)
		}
	}()
	_, err := m.bckDao.StoreBlob(b, rd)
	if err != nil {
		clog.Logger.Errorf("error getting blob: %s,%v", id, err)
	}
	defer rd.Close()
}

// HasBlob getting the description of the file
func (m *mainStorageDao) HasBlob(id string) (bool, error) {
	return m.stgDao.HasBlob(id)
}

// GetBlobDescription getting the description of the file
func (m *mainStorageDao) GetBlobDescription(id string) (*model.BlobDescription, error) {
	return m.stgDao.GetBlobDescription(id)
}

// RetrieveBlob retrieving the binary data from the storage system
func (m *mainStorageDao) RetrieveBlob(id string, w io.Writer) error {
	return m.stgDao.RetrieveBlob(id, w)
}

// DeleteBlob removing a blob from the storage system
func (m *mainStorageDao) DeleteBlob(id string) error {
	err := m.stgDao.DeleteBlob(id)
	if err != nil {
		return err
	}
	if m.bckDao != nil {
		err = m.bckDao.DeleteBlob(id)
		if err != nil {
			clog.Logger.Errorf("error deleting blob on backup: %v", err)
		}
	}
	m.rtnMng.DeleteRetention(m.tenant, id)
	return nil
}

//GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (m *mainStorageDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return m.stgDao.GetAllRetentions(callback)
}
func (m *mainStorageDao) AddRetention(r *model.RetentionEntry) error {
	err := m.stgDao.AddRetention(r)
	if m.bckDao != nil {
		err1 := m.bckDao.AddRetention(r)
		if err1 != nil {
			clog.Logger.Errorf("error adding retention on backup: %v", err1)
		}
	}
	return err
}
func (m *mainStorageDao) DeleteRetention(id string) error {
	err := m.stgDao.DeleteRetention(id)
	if m.bckDao != nil {
		err1 := m.bckDao.DeleteRetention(id)
		if err1 != nil {
			clog.Logger.Errorf("error deleting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}
func (m *mainStorageDao) ResetRetention(id string) error {
	err := m.stgDao.ResetRetention(id)
	if m.bckDao != nil {
		err1 := m.bckDao.ResetRetention(id)
		if err1 != nil {
			clog.Logger.Errorf("error reseting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}

// Close closing the blob storage
func (m *mainStorageDao) Close() error {
	err := m.stgDao.Close()
	if m.bckDao != nil {
		err1 := m.bckDao.Close()
		if err1 != nil {
			clog.Logger.Errorf("error closing backup storage: %v", err1)
		}
	}
	return err
}
