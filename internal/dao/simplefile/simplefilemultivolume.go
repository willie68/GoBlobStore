package simplefile

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/volume"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// SimpleFileMultiVolumeDao this dao takes multi volumes and treats them as a single file storage
type SimpleFileMultiVolumeDao struct {
	RootPath string // this is the root path for the file system storage
	Tenant   string // this is the tenant, on which this dao will work
	volMan   volume.VolumeManager
	daos     []SimpleFileBlobStorageDao
	daoIdx   map[string]*SimpleFileBlobStorageDao
	cm       sync.Mutex
}

// checking interface compatibility
var _ interfaces.BlobStorageDao = &SimpleFileMultiVolumeDao{}

// defining some error
var (
	ErrNotImplemented = errors.New("not implemented")
	ErrDaoNotFound    = errors.New("dao not found")
)

// ---- SimpleFileMultiVolumeDao

// Init initialize this dao
func (s *SimpleFileMultiVolumeDao) Init() error {
	if s.Tenant == "" {
		return errors.New("tenant should not be null or empty")
	}
	s.cm = sync.Mutex{}
	volMan, err := volume.NewVolumeManager(s.RootPath)
	if err != nil {
		return err
	}
	s.cm.Lock()
	s.daos = make([]SimpleFileBlobStorageDao, 0)
	s.daoIdx = make(map[string]*SimpleFileBlobStorageDao)
	s.cm.Unlock()
	s.volMan = volMan
	s.volMan.AddCallback(func(name string) bool {
		return s.addVolume(name)
	})
	s.volMan.Init()
	return nil
}

// GetTenant return the id of the tenant
func (s *SimpleFileMultiVolumeDao) GetTenant() string {
	return s.Tenant
}

// GetBlobs walking thru all blobs of this tenant
func (s *SimpleFileMultiVolumeDao) GetBlobs(callback func(id string) bool) error {
	for _, dao := range s.daos {
		err := dao.GetBlobs(callback)
		if err != io.EOF {
			return err
		}
	}
	return io.EOF
}

// StoreBlob storing a blob to the storage system
func (s *SimpleFileMultiVolumeDao) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	dao, err := s.selectDao()
	if err != nil {
		return "", err
	}
	return dao.StoreBlob(b, f)
}

// UpdateBlobDescription updating the blob description
// TODO implement this
func (s *SimpleFileMultiVolumeDao) UpdateBlobDescription(_ string, _ *model.BlobDescription) error {
	return ErrNotImplemented
}

// HasBlob checking if one dao has this blob
func (s *SimpleFileMultiVolumeDao) HasBlob(id string) (bool, error) {
	_, err := s.dao4id(id)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetBlobDescription getting the lob description from the dao holding the blob
func (s *SimpleFileMultiVolumeDao) GetBlobDescription(id string) (*model.BlobDescription, error) {
	dao, err := s.dao4id(id)
	if err != nil {
		return nil, err
	}
	return dao.GetBlobDescription(id)
}

// RetrieveBlob retrieving the blob from the first dao holding the blob file
func (s *SimpleFileMultiVolumeDao) RetrieveBlob(id string, writer io.Writer) error {
	dao, err := s.dao4id(id)
	if err != nil {
		return err
	}
	return dao.RetrieveBlob(id, writer)
}

// DeleteBlob removing a blob from the storage system
func (s *SimpleFileMultiVolumeDao) DeleteBlob(id string) error {
	dao, err := s.dao4id(id)
	if err != nil {
		return err
	}
	return dao.DeleteBlob(id)
}

// CheckBlob checking a single blob from the storage system
func (s *SimpleFileMultiVolumeDao) CheckBlob(id string) (*model.CheckInfo, error) {
	dao, err := s.dao4id(id)
	if err != nil {
		return nil, err
	}
	return dao.CheckBlob(id)
}

// SearchBlobs is not implemented for this storage
func (s *SimpleFileMultiVolumeDao) SearchBlobs(_ string, _ func(id string) bool) error {
	return ErrNotImplemented
}

// GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returning a false
func (s *SimpleFileMultiVolumeDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	for _, dao := range s.daos {
		err := dao.GetAllRetentions(callback)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetRetention getting a single retention entry
func (s *SimpleFileMultiVolumeDao) GetRetention(id string) (model.RetentionEntry, error) {
	dao, err := s.dao4id(id)
	if err != nil {
		return model.RetentionEntry{}, err
	}
	return dao.GetRetention(id)
}

// AddRetention adding a retention entry to the storage
func (s *SimpleFileMultiVolumeDao) AddRetention(r *model.RetentionEntry) error {
	dao, err := s.dao4id(r.BlobID)
	if err != nil {
		return err
	}
	return dao.AddRetention(r)
}

// DeleteRetention deletes the retention entry from the storage
func (s *SimpleFileMultiVolumeDao) DeleteRetention(id string) error {
	dao, err := s.dao4id(id)
	if err != nil {
		return err
	}
	return dao.DeleteRetention(id)
}

// ResetRetention resets the retention for a blob
func (s *SimpleFileMultiVolumeDao) ResetRetention(id string) error {
	r, err := s.GetRetention(id)
	if err != nil {
		return err
	}
	r.RetentionBase = time.Now().UnixMilli()
	return s.AddRetention(&r)
}

// GetLastError returning the last error (niy)
func (s *SimpleFileMultiVolumeDao) GetLastError() error {
	return ErrNotImplemented
}

// Close closing the storage
func (s *SimpleFileMultiVolumeDao) Close() error {
	for _, dao := range s.daos {
		dao.Close()
	}
	s.daos = make([]SimpleFileBlobStorageDao, 0)
	return nil
}

func (s *SimpleFileMultiVolumeDao) selectDao() (*SimpleFileBlobStorageDao, error) {
	rnd := s.volMan.Rnd()
	name := s.volMan.SelectFree(rnd)
	s.cm.Lock()
	defer s.cm.Unlock()
	dao := s.daoIdx[name]
	if dao == nil {
		return nil, errors.New("dao not found")
	}
	return dao, nil
}

func (s *SimpleFileMultiVolumeDao) dao4id(id string) (*SimpleFileBlobStorageDao, error) {
	for _, dao := range s.daos {
		ok, _ := dao.HasBlob(id)
		if ok {
			return &dao, nil
		}
	}
	return nil, ErrDaoNotFound
}

func (s *SimpleFileMultiVolumeDao) addVolume(name string) bool {
	if !s.volMan.HasVolume(name) {
		return false
	}
	vi := s.volMan.Info(name)
	if vi == nil {
		return false
	}
	sfbd := &SimpleFileBlobStorageDao{
		RootPath: vi.Path,
		Tenant:   s.Tenant,
	}
	err := sfbd.Init()
	if err != nil {
		return false
	}
	s.daos = append(s.daos, *sfbd)
	s.cm.Lock()
	defer s.cm.Unlock()
	s.daoIdx[name] = sfbd
	return true
}
