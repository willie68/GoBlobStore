package dao

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	clog "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type SimpleFileBlobStorageDao struct {
	RootPath string // this is the root path for the file system storage
	Tenant   string // this is the tenant, on which this dao will work
	filepath string // direct path to the tenant specifig sub path
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
	return "", errors.New("not implemented yet")
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

func (s *SimpleFileBlobStorageDao) RetrieveBlob(idStr string, writer io.Writer) error {
	return errors.New("not implemented yet")
}

func (s *SimpleFileBlobStorageDao) DeleteBlob(id string) error {
	return errors.New("not implemented yet")
}

func (s *SimpleFileBlobStorageDao) Close() error {
	return errors.New("not implemented yet")
}
