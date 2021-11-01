package dao

import (
	"io"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type mainStorageDao struct {
	rtnMng interfaces.RetentionManager
	stgDao interfaces.BlobStorageDao
	bckDao interfaces.BlobStorageDao
	tenant string
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
	return id, err
}

// GetBlobDescription getting the description of the file
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
	m.rtnMng.DeleteRetention(m.tenant, id)
	return nil
}

//GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (m *mainStorageDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return m.stgDao.GetAllRetentions(callback)
}
func (m *mainStorageDao) AddRetention(r *model.RetentionEntry) error {
	return m.stgDao.AddRetention(r)
}
func (m *mainStorageDao) DeleteRetention(id string) error {
	return m.stgDao.DeleteRetention(id)
}
func (m *mainStorageDao) ResetRetention(id string) error {
	return m.stgDao.ResetRetention(id)
}

// Close closing the blob storage
func (m *mainStorageDao) Close() error {
	return m.stgDao.Close()
}
