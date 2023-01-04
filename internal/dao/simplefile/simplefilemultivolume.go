package simplefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/volume"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type SimpleFileMultiVolumeDao struct {
	RootPath string                           // this is the root path for the file system storage
	Tenant   string                           // this is the tenant, on which this dao will work
	bdCch    map[string]model.BlobDescription // short time cache of blobdescriptions
	volMan   volume.VolumeManager
	cm       sync.RWMutex
	daos     []SimpleFileBlobStorageDao
}

var _ interfaces.BlobStorageDao = &SimpleFileMultiVolumeDao{}

// ---- SimpleFileMultiVolumeDao
func (s *SimpleFileMultiVolumeDao) Init() error {
	if s.Tenant == "" {
		return errors.New("tenant should not be null or empty")
	}
	s.bdCch = make(map[string]model.BlobDescription)
	volMan, err := volume.NewVolumeManager(s.RootPath)
	if err != nil {
		return err
	}
	s.daos = make([]SimpleFileBlobStorageDao, 0)
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

func (s *SimpleFileMultiVolumeDao) GetBlobs(callback func(id string) bool) error {
	return s.getBlobsV2(callback)
}

func (s *SimpleFileMultiVolumeDao) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	return s.storeBlobV2(b, f)
}

// updating the blob description
func (s *SimpleFileMultiVolumeDao) UpdateBlobDescription(id string, b *model.BlobDescription) error {
	err := s.updateBlobDescriptionV2(id, b)
	if err == os.ErrNotExist {
		err = s.updateBlobDescriptionV1(id, b)
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *SimpleFileMultiVolumeDao) HasBlob(id string) (bool, error) {
	if id == "" {
		return false, nil
	}
	found := s.hasBlobV1(id)
	if found {
		return true, nil
	}
	found = s.hasBlobV2(id)
	return found, nil
}

func (s *SimpleFileMultiVolumeDao) GetBlobDescription(id string) (*model.BlobDescription, error) {
	info, err := s.getBlobDescriptionV2(id)
	if err == os.ErrNotExist {
		info, err = s.getBlobDescriptionV1(id)
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SimpleFileMultiVolumeDao) RetrieveBlob(id string, writer io.Writer) error {
	err := s.getBlobV2(id, writer)
	if err == os.ErrNotExist {
		err = s.getBlobV1(id, writer)
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *SimpleFileMultiVolumeDao) DeleteBlob(id string) error {
	s.deleteFilesV1(id)
	s.deleteFilesV2(id)
	return nil
}

// CheckBlob checking a single blob from the storage system
func (s *SimpleFileMultiVolumeDao) CheckBlob(id string) (*model.CheckInfo, error) {
	return utils.CheckBlob(id, s)
}

func (s *SimpleFileMultiVolumeDao) SearchBlobs(q string, callback func(id string) bool) error {
	return errors.New("not implemented yet")
}

// GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (s *SimpleFileMultiVolumeDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	retCbk := func(path string, file os.FileInfo, err error) error {
		if file != nil {
			if !file.IsDir() {
				dat, err := os.ReadFile(path)
				if err != nil {
					log.Logger.Errorf("GetAllRetention: error getting file data for: %s\r\n%v", file.Name(), err)
					return nil
				}
				ety := model.RetentionEntry{}
				err = json.Unmarshal(dat, &ety)
				if err != nil {
					log.Logger.Errorf("GetAllRetention: error deserialising: %s\r\n%v", file.Name(), err)
					return nil
				}
				ok := callback(ety)
				if !ok {
					return filepath.SkipDir
				}
				return nil
			}
		}
		return nil
	}
	retPath := filepath.Join(s.filepath, RETENTION_PATH)
	filepath.Walk(retPath, retCbk)

	return nil
}

func (s *SimpleFileMultiVolumeDao) GetRetention(id string) (model.RetentionEntry, error) {
	r, err := s.getRetention(id)
	if err != nil {
		return model.RetentionEntry{}, err
	}
	if r == nil {
		return model.RetentionEntry{}, fmt.Errorf("no retention file found for id %s", id)
	}
	return *r, err
}

func (s *SimpleFileMultiVolumeDao) AddRetention(r *model.RetentionEntry) error {
	b, err := s.GetBlobDescription(r.BlobID)
	if err != nil {
		return err
	}
	b.Retention = r.Retention
	b.Properties[retentionBaseKey] = r.RetentionBase
	return s.writeRetentionFile(b)
}

func (s *SimpleFileMultiVolumeDao) DeleteRetention(id string) error {
	idx := s.dao4id(id)
	dao := s.daos[idx]
	return dao.deleteRetentionFile(id)
}

func (s *SimpleFileMultiVolumeDao) ResetRetention(id string) error {
	r, err := s.getRetention(id)
	if err != nil {
		return err
	}
	r.RetentionBase = int(time.Now().UnixNano() / 1000000)
	return s.AddRetention(r)
}

func (s *SimpleFileMultiVolumeDao) GetLastError() error {
	return nil
}

func (s *SimpleFileMultiVolumeDao) Close() error {
	for _, dao := range s.daos {
		dao.Close()
	}
	s.daos = make([]SimpleFileBlobStorageDao, 0)
	return nil
}

func (s *SimpleFileMultiVolumeDao) dao4id(id string) int {
	for x, dao := range s.daos {
		ok, _ := dao.HasBlob(id)
		if ok {
			return x
		}
	}
	return -1
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
	return true
}
