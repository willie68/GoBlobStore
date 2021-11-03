package simplefile

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	clog "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type SimpleFileTenantManager struct {
	RootPath string // this is the root path for the file system storage
}

var _ interfaces.TenantDao = &SimpleFileTenantManager{}

type SimpleFileBlobStorageDao struct {
	RootPath string // this is the root path for the file system storage
	Tenant   string // this is the tenant, on which this dao will work
	filepath string // direct path to the tenant specifig sub path
}

var _ interfaces.BlobStorageDao = &SimpleFileBlobStorageDao{}

const retentionBaseKey = "retentionBase"

func (s *SimpleFileTenantManager) Init() error {
	return nil
}

func (s *SimpleFileTenantManager) GetTenants(callback func(tenant string) bool) error {
	infos, err := ioutil.ReadDir(s.RootPath)
	if err != nil {
		return err
	}
	for _, i := range infos {
		if !strings.HasPrefix(i.Name(), "_") {
			ok := callback(i.Name())
			if !ok {
				return nil
			}
		}
	}
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

func (s *SimpleFileTenantManager) Close() error {
	return nil
}

//---- SimpleFileBlobStorageDao
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

func (s *SimpleFileBlobStorageDao) HasBlob(id string) (bool, error) {
	found := s.hasBlobV1(id)
	if found {
		return true, nil
	}
	found = s.hasBlobV2(id)
	return found, nil
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
	retCbk := func(path string, file os.FileInfo, err error) error {
		if file != nil {
			if !file.IsDir() {
				dat, err := os.ReadFile(path)
				if err != nil {
					clog.Logger.Errorf("GetAllRetention: error getting file data for: %s\r\n%v", file.Name(), err)
					return nil
				}
				ety := model.RetentionEntry{}
				err = json.Unmarshal(dat, &ety)
				if err != nil {
					clog.Logger.Errorf("GetAllRetention: error deserialising: %s\r\n%v", file.Name(), err)
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

func (s *SimpleFileBlobStorageDao) AddRetention(r *model.RetentionEntry) error {
	b, err := s.GetBlobDescription(r.BlobID)
	if err != nil {
		return err
	}
	b.Retention = r.Retention
	b.Properties[retentionBaseKey] = r.RetentionBase
	return s.writeRetentionFile(b)
}

func (s *SimpleFileBlobStorageDao) DeleteRetention(id string) error {
	/*
		_, err := s.GetBlobDescription(id)
		if err != nil {
			return err
		}
	*/
	return s.deleteRetentionFile(id)
}

func (s *SimpleFileBlobStorageDao) ResetRetention(id string) error {
	r, err := s.getRetention(id)
	if err != nil {
		return err
	}
	r.RetentionBase = int(time.Now().UnixNano() / 1000000)
	return s.AddRetention(r)
}

func (s *SimpleFileBlobStorageDao) Close() error {
	return nil
}
