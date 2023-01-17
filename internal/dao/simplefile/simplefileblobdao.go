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

// SimpleFileBlobStorageDao service for storing blob files into a file system
type SimpleFileBlobStorageDao struct {
	RootPath string                           // this is the root path for the file system storage
	Tenant   string                           // this is the tenant, on which this dao will work
	filepath string                           // direct path to the tenant specifig sub path
	bdCch    map[string]model.BlobDescription // short time cache of blobdescriptions
	cm       sync.RWMutex
}

var _ interfaces.BlobStorageDao = &SimpleFileBlobStorageDao{}

const retentionBaseKey = "retentionBase"

// ---- SimpleFileBlobStorageDao

// Init initialise this dao
func (s *SimpleFileBlobStorageDao) Init() error {
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
func (s *SimpleFileBlobStorageDao) GetTenant() string {
	return s.Tenant
}

// GetBlobs getting a list of blob from the filesystem
func (s *SimpleFileBlobStorageDao) GetBlobs(callback func(id string) bool) error {
	return s.getBlobsV2(callback)
}

// StoreBlob storing a blob to the storage system
func (s *SimpleFileBlobStorageDao) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	return s.storeBlobV2(b, f)
}

// UpdateBlobDescription updating the blob description
func (s *SimpleFileBlobStorageDao) UpdateBlobDescription(id string, b *model.BlobDescription) error {
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
func (s *SimpleFileBlobStorageDao) HasBlob(id string) (bool, error) {
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
func (s *SimpleFileBlobStorageDao) GetBlobDescription(id string) (*model.BlobDescription, error) {
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
func (s *SimpleFileBlobStorageDao) RetrieveBlob(id string, writer io.Writer) error {
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
func (s *SimpleFileBlobStorageDao) DeleteBlob(id string) error {
	s.deleteFilesV1(id)
	s.deleteFilesV2(id)
	return nil
}

// CheckBlob checking a single blob from the storage system
func (s *SimpleFileBlobStorageDao) CheckBlob(id string) (*model.CheckInfo, error) {
	return utils.CheckBlob(id, s)
}

// SearchBlobs quering a single blob, niy
func (s *SimpleFileBlobStorageDao) SearchBlobs(_ string, _ func(id string) bool) error {
	return errors.New("not implemented yet")
}

// GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (s *SimpleFileBlobStorageDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
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
	filepath.Walk(retPath, retCbk)

	return nil
}

// GetRetention getting a single retention entry
func (s *SimpleFileBlobStorageDao) GetRetention(id string) (model.RetentionEntry, error) {
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
func (s *SimpleFileBlobStorageDao) AddRetention(r *model.RetentionEntry) error {
	b, err := s.GetBlobDescription(r.BlobID)
	if err != nil {
		return err
	}
	b.Retention = r.Retention
	b.Properties[retentionBaseKey] = r.RetentionBase
	return s.writeRetentionFile(b)
}

// DeleteRetention deletes the retention entry from the storage
func (s *SimpleFileBlobStorageDao) DeleteRetention(id string) error {
	/*
		_, err := s.GetBlobDescription(id)
		if err != nil {
			return err
		}
	*/
	return s.deleteRetentionFile(id)
}

// ResetRetention resets the retention for a blob
func (s *SimpleFileBlobStorageDao) ResetRetention(id string) error {
	r, err := s.getRetention(id)
	if err != nil {
		return err
	}
	r.RetentionBase = time.Now().UnixMilli()
	return s.AddRetention(r)
}

// GetLastError returning the last error (niy)
func (s *SimpleFileBlobStorageDao) GetLastError() error {
	return nil
}

// Close closing the storage
func (s *SimpleFileBlobStorageDao) Close() error {
	return nil
}
