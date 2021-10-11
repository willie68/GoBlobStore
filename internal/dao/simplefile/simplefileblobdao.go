package simplefile

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	clog "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type SimpleFileTenantManager struct {
	RootPath string // this is the root path for the file system storage
}

type SimpleFileBlobStorageDao struct {
	RootPath string // this is the root path for the file system storage
	Tenant   string // this is the tenant, on which this dao will work
	filepath string // direct path to the tenant specifig sub path
}

func (s *SimpleFileTenantManager) Init() error {
	return nil
}

func (s *SimpleFileTenantManager) AddTenant(tenant string) error {

	tenantPath := filepath.Join(s.RootPath, tenant)

	err := os.MkdirAll(tenantPath, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (s *SimpleFileTenantManager) RemoveTenant(tenant string) error {
	if !s.HasTenant(tenant) {
		return errors.New("tenant not exists")
	}
	tenantPath := filepath.Join(s.RootPath, tenant)
	err := os.RemoveAll(tenantPath)
	if err != nil {
		return err
	}
	return nil
}

func (s *SimpleFileTenantManager) HasTenant(tenant string) bool {
	tenantPath := filepath.Join(s.RootPath, tenant)

	if _, err := os.Stat(tenantPath); os.IsNotExist(err) {
		return false
	}

	return true
}

func (s *SimpleFileTenantManager) GetSize(tenant string) int64 {
	if !s.HasTenant(tenant) {
		return -1
	}
	tenantPath := filepath.Join(s.RootPath, tenant)

	if _, err := os.Stat(tenantPath); os.IsNotExist(err) {
		return 0
	}

	var dirSize int64 = 0
	readSize := func(path string, file os.FileInfo, err error) error {
		if !file.IsDir() {
			dirSize += file.Size()
		}
		return nil
	}

	filepath.Walk(tenantPath, readSize)
	return dirSize
}

func (s *SimpleFileBlobStorageDao) Init() error {
	if s.Tenant == "" {
		return errors.New("tenant should not be null or empty")
	}
	fileppath, err := filepath.Abs(filepath.Join(s.RootPath, s.Tenant))
	if err != nil {
		return err
	}
	s.filepath = fileppath
	clog.Logger.Debugf("building file path for tenant: %s", s.filepath)
	if _, err := os.Stat(s.filepath); os.IsNotExist(err) {
		clog.Logger.Debugf("tenant not exists: %s", s.Tenant)
	}
	return nil
}

func (s *SimpleFileBlobStorageDao) GetBlobs(offset int, limit int) ([]string, error) {
	blobs, err := s.getBlobsV2(0, limit)
	if err != nil {
		return nil, err
	}
	return blobs, nil
}

func (s *SimpleFileBlobStorageDao) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	return s.storeBlobV2(b, f)
}

func (s *SimpleFileBlobStorageDao) GetBlobDescription(id string) (*model.BlobDescription, error) {
	info, err := s.getBlobDescriptionV1(id)
	if err == os.ErrNotExist {
		info, err = s.getBlobDescriptionV2(id)
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (s *SimpleFileBlobStorageDao) RetrieveBlob(id string, writer io.Writer) error {
	err := s.getBlobV1(id, writer)
	if err == os.ErrNotExist {
		err = s.getBlobV2(id, writer)
	}
	if err != nil {
		return err
	}
	return nil
}

func (s *SimpleFileBlobStorageDao) DeleteBlob(id string) error {
	s.deleteFilesV1(id)
	s.deleteFilesV2(id)
	return nil
}

//GetAllRetentions for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (s *SimpleFileBlobStorageDao) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return errors.New("not yet implemented")
}
func (s *SimpleFileBlobStorageDao) AddRetention(r *model.RetentionEntry) error {
	return errors.New("not yet implemented")
}
func (s *SimpleFileBlobStorageDao) DeleteRetention(r *model.RetentionEntry) error {
	return errors.New("not yet implemented")
}
func (s *SimpleFileBlobStorageDao) ResetRetention(r *model.RetentionEntry) error {
	return errors.New("not yet implemented")
}

func (s *SimpleFileBlobStorageDao) Close() error {
	return nil
}
