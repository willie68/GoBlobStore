package dao

import (
	"io"

	"github.com/willie68/GoBlobStore/pkg/model"
)

type mainStorageDao struct {
	retMng     RetentionManager
	storageDao BlobStorageDao
	tenant     string
}

// Init initialise this dao
func (m *mainStorageDao) Init() error {
	return m.storageDao.Init()
}

// GetBlobs getting a list of blob from the filesystem using offset and limit
func (m *mainStorageDao) GetBlobs(offset int, limit int) ([]string, error) {
	return m.storageDao.GetBlobs(offset, limit)
}

// StoreBlob storing a blob to the storage system
func (m *mainStorageDao) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	id, err := m.storageDao.StoreBlob(b, f)
	b.BlobID = id
	if err == nil && m.retMng != nil {
		err = m.retMng.AddRetention(m.tenant, b)
	}
	return id, err
}

// GetBlobDescription getting the description of the file
func (m *mainStorageDao) GetBlobDescription(id string) (*model.BlobDescription, error) {
	return m.storageDao.GetBlobDescription(id)
}

// RetrieveBlob retrieving the binary data from the storage system
func (m *mainStorageDao) RetrieveBlob(id string, w io.Writer) error {
	return m.storageDao.RetrieveBlob(id, w)
}

// DeleteBlob removing a blob from the storage system
func (m *mainStorageDao) DeleteBlob(id string) error {
	return m.storageDao.DeleteBlob(id)
}

//GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (m *mainStorageDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return m.storageDao.GetAllRetentions(callback)
}
func (m *mainStorageDao) AddRetention(r *model.RetentionEntry) error {
	return m.storageDao.AddRetention(r)
}
func (m *mainStorageDao) DeleteRetention(r *model.RetentionEntry) error {
	return m.storageDao.DeleteRetention(r)
}
func (m *mainStorageDao) ResetRetention(r *model.RetentionEntry) error {
	return m.storageDao.ResetRetention(r)
}

// Close closing the blob storage
func (m *mainStorageDao) Close() error {
	return m.storageDao.Close()
}
