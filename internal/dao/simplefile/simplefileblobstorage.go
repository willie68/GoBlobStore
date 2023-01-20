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
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// BlobStorage service for storing blob files into a file system
type BlobStorage struct {
	RootPath string                           // this is the root path for the file system storage
	Tenant   string                           // this is the tenant, on which this dao will work
	filepath string                           // direct path to the tenant specific sub path
	bdCch    map[string]model.BlobDescription // short time cache of blob descriptions
	cm       sync.RWMutex
}

var _ interfaces.BlobStorage = &BlobStorage{}

const retentionBaseKey = "retentionBase"

// ---- SimpleFileBlobStorageDao

// Init initialize this dao
func (s *BlobStorage) Init() error {
	if s.Tenant == "" {
		return errors.New("tenant should not be null or empty")
	}
	fileppath, err := filepath.Abs(filepath.Join(s.RootPath, s.Tenant))
	if err != nil {
		return err
	}
	s.filepath = fileppath
	log.Logger.Debugf("building file path for tenant: %s", s.filepath)
	if _, err := os.Stat(s.filepath); os.IsNotExist(err) {
		log.Logger.Debugf("tenant not exists: %s", s.Tenant)
	}
	s.bdCch = make(map[string]model.BlobDescription)
	return nil
}

// GetTenant return the id of the tenant
func (s *BlobStorage) GetTenant() string {
	return s.Tenant
}

// GetBlobs getting a list of blob from the filesystem
func (s *BlobStorage) GetBlobs(callback func(id string) bool) error {
	return s.getBlobsV2(callback)
}

// StoreBlob storing a blob to the storage system
func (s *BlobStorage) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	return s.storeBlobV2(b, f)
}

// UpdateBlobDescription updating the blob description
func (s *BlobStorage) UpdateBlobDescription(id string, b *model.BlobDescription) error {
	err := s.updateBlobDescriptionV2(id, b)
	if err == os.ErrNotExist {
		err = s.updateBlobDescriptionV1(id, b)
	}
	if err != nil {
		return err
	}
	return nil
}

// HasBlob checking, if a blob is present
func (s *BlobStorage) HasBlob(id string) (bool, error) {
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

// GetBlobDescription getting the description of the file
func (s *BlobStorage) GetBlobDescription(id string) (*model.BlobDescription, error) {
	info, err := s.getBlobDescriptionV2(id)
	if err == os.ErrNotExist {
		info, err = s.getBlobDescriptionV1(id)
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}

// RetrieveBlob retrieving the binary data from the storage system
func (s *BlobStorage) RetrieveBlob(id string, writer io.Writer) error {
	err := s.getBlobV2(id, writer)
	if err == os.ErrNotExist {
		err = s.getBlobV1(id, writer)
	}
	if err != nil {
		return err
	}
	return nil
}

// DeleteBlob removing a blob from the storage system
func (s *BlobStorage) DeleteBlob(id string) error {
	err := s.deleteFilesV1(id)
	if errors.Is(err, os.ErrNotExist) {
		err = s.deleteFilesV2(id)
	}
	return err
}

// CheckBlob checking a single blob from the storage system
func (s *BlobStorage) CheckBlob(id string) (*model.CheckInfo, error) {
	return utils.CheckBlob(id, s)
}

// SearchBlobs querying a single blob, niy
func (s *BlobStorage) SearchBlobs(_ string, _ func(id string) bool) error {
	return errors.New("not implemented yet")
}

// GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (s *BlobStorage) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
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
	retPath := filepath.Join(s.filepath, RetentionPath)
	err := filepath.Walk(retPath, retCbk)
	return err
}

// GetRetention getting a single retention entry
func (s *BlobStorage) GetRetention(id string) (model.RetentionEntry, error) {
	r, err := s.getRetention(id)
	if err != nil {
		return model.RetentionEntry{}, err
	}
	if r == nil {
		return model.RetentionEntry{}, fmt.Errorf("no retention file found for id %s", id)
	}
	return *r, err
}

// AddRetention adding a retention entry to the storage
func (s *BlobStorage) AddRetention(r *model.RetentionEntry) error {
	b, err := s.GetBlobDescription(r.BlobID)
	if err != nil {
		return err
	}
	b.Retention = r.Retention
	b.Properties[retentionBaseKey] = r.RetentionBase
	return s.writeRetentionFile(b)
}

// DeleteRetention deletes the retention entry from the storage
func (s *BlobStorage) DeleteRetention(id string) error {
	/*
		_, err := s.GetBlobDescription(id)
		if err != nil {
			return err
		}
	*/
	return s.deleteRetentionFile(id)
}

// ResetRetention resets the retention for a blob
func (s *BlobStorage) ResetRetention(id string) error {
	r, err := s.getRetention(id)
	if err != nil {
		return err
	}
	r.RetentionBase = time.Now().UnixMilli()
	return s.AddRetention(r)
}

// GetLastError returning the last error (niy)
func (s *BlobStorage) GetLastError() error {
	return nil
}

// Close closing the storage
func (s *BlobStorage) Close() error {
	return nil
}
