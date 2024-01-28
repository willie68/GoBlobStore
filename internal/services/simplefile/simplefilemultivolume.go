package simplefile

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/internal/services/volume"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// MultiVolumeStorage this service takes multi volumes and treats them as a single file storage
type MultiVolumeStorage struct {
	RootPath string // this is the root path for the file system storage
	Tenant   string // this is the tenant, on which this service will work
	volMan   volume.Manager
	srvs     []BlobStorage
	idxsrv   map[string]*BlobStorage
	cm       sync.Mutex
}

// checking interface compatibility
var _ interfaces.BlobStorage = &MultiVolumeStorage{}

// defining some error
var (
	ErrNotImplemented = errors.New("not implemented")
	ErrSrvNotFound    = errors.New("service not found")
)

// ---- SimpleFileMultiVolume

// Init initialize this service
func (s *MultiVolumeStorage) Init() error {
	if s.Tenant == "" {
		return errors.New("tenant should not be null or empty")
	}
	s.cm = sync.Mutex{}
	volMan, err := volume.NewVolumeManager(s.RootPath)
	if err != nil {
		return err
	}
	s.cm.Lock()
	s.srvs = make([]BlobStorage, 0)
	s.idxsrv = make(map[string]*BlobStorage)
	s.cm.Unlock()
	s.volMan = volMan
	s.volMan.AddCallback(func(name string) bool {
		return s.addVolume(name)
	})
	s.volMan.Init()
	return nil
}

// GetTenant return the id of the tenant
func (s *MultiVolumeStorage) GetTenant() string {
	return s.Tenant
}

// GetBlobs walking thru all blobs of this tenant
func (s *MultiVolumeStorage) GetBlobs(callback func(id string) bool) error {
	for _, srv := range s.srvs {
		err := srv.GetBlobs(callback)
		if err != io.EOF {
			return err
		}
	}
	return io.EOF
}

// StoreBlob storing a blob to the storage system
func (s *MultiVolumeStorage) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	srv, err := s.selectSrv()
	if err != nil {
		return "", err
	}
	return srv.StoreBlob(b, f)
}

// UpdateBlobDescription updating the blob description
// TODO implement this
func (s *MultiVolumeStorage) UpdateBlobDescription(_ string, _ *model.BlobDescription) error {
	return ErrNotImplemented
}

// HasBlob checking if one service has this blob
func (s *MultiVolumeStorage) HasBlob(id string) (bool, error) {
	_, err := s.srv4id(id)
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetBlobDescription getting the lob description from the service holding the blob
func (s *MultiVolumeStorage) GetBlobDescription(id string) (*model.BlobDescription, error) {
	srv, err := s.srv4id(id)
	if err != nil {
		return nil, err
	}
	return srv.GetBlobDescription(id)
}

// RetrieveBlob retrieving the blob from the first service holding the blob file
func (s *MultiVolumeStorage) RetrieveBlob(id string, writer io.Writer) error {
	srv, err := s.srv4id(id)
	if err != nil {
		return err
	}
	return srv.RetrieveBlob(id, writer)
}

// DeleteBlob removing a blob from the storage system
func (s *MultiVolumeStorage) DeleteBlob(id string) error {
	srv, err := s.srv4id(id)
	if err != nil {
		return err
	}
	return srv.DeleteBlob(id)
}

// CheckBlob checking a single blob from the storage system
func (s *MultiVolumeStorage) CheckBlob(id string) (*model.CheckInfo, error) {
	srv, err := s.srv4id(id)
	if err != nil {
		return nil, err
	}
	return srv.CheckBlob(id)
}

// SearchBlobs is not implemented for this storage
func (s *MultiVolumeStorage) SearchBlobs(_ string, _ func(id string) bool) error {
	return ErrNotImplemented
}

// GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returning a false
func (s *MultiVolumeStorage) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	for _, srv := range s.srvs {
		err := srv.GetAllRetentions(callback)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetRetention getting a single retention entry
func (s *MultiVolumeStorage) GetRetention(id string) (model.RetentionEntry, error) {
	srv, err := s.srv4id(id)
	if err != nil {
		return model.RetentionEntry{}, err
	}
	return srv.GetRetention(id)
}

// AddRetention adding a retention entry to the storage
func (s *MultiVolumeStorage) AddRetention(r *model.RetentionEntry) error {
	srv, err := s.srv4id(r.BlobID)
	if err != nil {
		return err
	}
	return srv.AddRetention(r)
}

// DeleteRetention deletes the retention entry from the storage
func (s *MultiVolumeStorage) DeleteRetention(id string) error {
	srv, err := s.srv4id(id)
	if err != nil {
		return err
	}
	return srv.DeleteRetention(id)
}

// ResetRetention resets the retention for a blob
func (s *MultiVolumeStorage) ResetRetention(id string) error {
	r, err := s.GetRetention(id)
	if err != nil {
		return err
	}
	r.RetentionBase = time.Now().UnixMilli()
	return s.AddRetention(&r)
}

// GetLastError returning the last error (niy)
func (s *MultiVolumeStorage) GetLastError() error {
	return ErrNotImplemented
}

// Close closing the storage
func (s *MultiVolumeStorage) Close() error {
	for _, srv := range s.srvs {
		err := srv.Close()
		if err != nil {
			logger.Errorf("error closing volume service: %v", err)
		}
	}
	s.srvs = make([]BlobStorage, 0)
	return nil
}

func (s *MultiVolumeStorage) selectSrv() (*BlobStorage, error) {
	rnd := s.volMan.Rnd()
	name := s.volMan.SelectFree(rnd)
	s.cm.Lock()
	defer s.cm.Unlock()
	srv := s.idxsrv[name]
	if srv == nil {
		return nil, ErrSrvNotFound
	}
	return srv, nil
}

func (s *MultiVolumeStorage) srv4id(id string) (*BlobStorage, error) {
	for _, srv := range s.srvs {
		ok, _ := srv.HasBlob(id)
		if ok {
			return &srv, nil
		}
	}
	return nil, ErrSrvNotFound
}

func (s *MultiVolumeStorage) addVolume(name string) bool {
	if !s.volMan.HasVolume(name) {
		return false
	}
	vi := s.volMan.Info(name)
	if vi == nil {
		return false
	}
	sfbd := &BlobStorage{
		RootPath: vi.Path,
		Tenant:   s.Tenant,
	}
	err := sfbd.Init()
	if err != nil {
		return false
	}
	s.srvs = append(s.srvs, *sfbd)
	s.cm.Lock()
	defer s.cm.Unlock()
	s.idxsrv[name] = sfbd
	return true
}
