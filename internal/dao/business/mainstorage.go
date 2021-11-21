package business

import (
	"io"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	clog "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type MainStorageDao struct {
	RtnMng      interfaces.RetentionManager
	StgDao      interfaces.BlobStorageDao
	BckDao      interfaces.BlobStorageDao
	Bcksyncmode bool
	Tenant      string
}

// Init initialise this dao
func (m *MainStorageDao) Init() error {
	// all storages should be initialised before adding to this business class
	// there for only specifig initialisation for this class is required
	return nil
}

// GetBlobs getting a list of blob from the filesystem using offset and limit
func (m *MainStorageDao) GetBlobs(offset int, limit int) ([]string, error) {
	return m.StgDao.GetBlobs(offset, limit)
}

// StoreBlob storing a blob to the storage system
func (m *MainStorageDao) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	id, err := m.StgDao.StoreBlob(b, f)
	b.BlobID = id
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
	return id, err
}

func (m *MainStorageDao) backupFile(b *model.BlobDescription, id string) {
	rd, wr := io.Pipe()

	go func() {
		// close the writer, so the reader knows there's no more data
		defer wr.Close()

		err := m.StgDao.RetrieveBlob(id, wr)
		if err != nil {
			clog.Logger.Errorf("error getting blob: %s,%v", id, err)
		}
	}()
	_, err := m.BckDao.StoreBlob(b, rd)
	if err != nil {
		clog.Logger.Errorf("error getting blob: %s,%v", id, err)
	}
	defer rd.Close()
}

// HasBlob getting the description of the file
func (m *MainStorageDao) HasBlob(id string) (bool, error) {
	return m.StgDao.HasBlob(id)
}

// GetBlobDescription getting the description of the file
func (m *MainStorageDao) GetBlobDescription(id string) (*model.BlobDescription, error) {
	return m.StgDao.GetBlobDescription(id)
}

// RetrieveBlob retrieving the binary data from the storage system
func (m *MainStorageDao) RetrieveBlob(id string, w io.Writer) error {
	return m.StgDao.RetrieveBlob(id, w)
}

// DeleteBlob removing a blob from the storage system
func (m *MainStorageDao) DeleteBlob(id string) error {
	err := m.StgDao.DeleteBlob(id)
	if err != nil {
		return err
	}
	if m.BckDao != nil {
		err = m.BckDao.DeleteBlob(id)
		if err != nil {
			clog.Logger.Errorf("error deleting blob on backup: %v", err)
		}
	}
	m.RtnMng.DeleteRetention(m.Tenant, id)
	return nil
}

//GetAllRetentions for every retention entry for this Tenant we call this this function, you can stop the listing by returnong a false
func (m *MainStorageDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return m.StgDao.GetAllRetentions(callback)
}
func (m *MainStorageDao) AddRetention(r *model.RetentionEntry) error {
	err := m.StgDao.AddRetention(r)
	if m.BckDao != nil {
		err1 := m.BckDao.AddRetention(r)
		if err1 != nil {
			clog.Logger.Errorf("error adding retention on backup: %v", err1)
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
		err1 := m.BckDao.DeleteRetention(id)
		if err1 != nil {
			clog.Logger.Errorf("error deleting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}
func (m *MainStorageDao) ResetRetention(id string) error {
	err := m.StgDao.ResetRetention(id)
	if m.BckDao != nil {
		err1 := m.BckDao.ResetRetention(id)
		if err1 != nil {
			clog.Logger.Errorf("error reseting retention on backup:%s, %v", id, err1)
		}
	}
	return err
}

// Close closing the blob storage
func (m *MainStorageDao) Close() error {
	err := m.StgDao.Close()
	if m.BckDao != nil {
		err1 := m.BckDao.Close()
		if err1 != nil {
			clog.Logger.Errorf("error closing backup storage: %v", err1)
		}
	}
	return err
}
