package s3

import (
	"errors"
	"io"

	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type S3TenantManager struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Password  string
}

type S3BlobStorage struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Tenant    string
	Password  string
}

func (s *S3TenantManager) Init() error {
	return errors.New("not yet implemented")
}

func (s *S3TenantManager) GetTenants(callback func(tenant string) bool) error {
	return errors.New("not yet implemented")
}

func (s *S3TenantManager) AddTenant(tenant string) error {
	return errors.New("not yet implemented")
}

func (s *S3TenantManager) RemoveTenant(tenant string) error {
	return errors.New("not yet implemented")
}

func (s *S3TenantManager) HasTenant(tenant string) bool {
	return false
}

func (s *S3TenantManager) GetSize(tenant string) int64 {
	return 0
}

func (s *S3TenantManager) getEncryption() encrypt.ServerSide {
	return encrypt.DefaultPBKDF([]byte(s.Password), []byte(s.Bucket))
}

func (s *S3TenantManager) Close() error {
	return errors.New("not yet implemented")
}

//S3 Blob Storage
// initialise this dao
func (s *S3BlobStorage) Init() error {
	return errors.New("not yet implemented")
}

// getting a list of blob from the filesystem using offset and limit
func (s *S3BlobStorage) GetBlobs(offset int, limit int) ([]string, error) {
	return nil, errors.New("not yet implemented")
}

// CRUD operation on the blob files
// storing a blob to the storage system
func (s *S3BlobStorage) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	return "", errors.New("not yet implemented")
}

// checking, if a blob is present
func (s *S3BlobStorage) HasBlob(id string) (bool, error) {
	return false, errors.New("not yet implemented")
}

// getting the description of the file
func (s *S3BlobStorage) GetBlobDescription(id string) (*model.BlobDescription, error) {
	return nil, errors.New("not yet implemented")
}

// retrieving the binary data from the storage system
func (s *S3BlobStorage) RetrieveBlob(id string, w io.Writer) error {
	return errors.New("not yet implemented")
}

// removing a blob from the storage system
func (s *S3BlobStorage) DeleteBlob(id string) error {
	return errors.New("not yet implemented")
}

//Retentionrelated methods
// for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (s *S3BlobStorage) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	return errors.New("not yet implemented")
}

func (s *S3BlobStorage) AddRetention(r *model.RetentionEntry) error {
	return errors.New("not yet implemented")
}

func (s *S3BlobStorage) DeleteRetention(id string) error {
	return errors.New("not yet implemented")
}

func (s *S3BlobStorage) ResetRetention(id string) error {
	return errors.New("not yet implemented")
}

// closing the storage
func (s *S3BlobStorage) Close() error {
	return errors.New("not yet implemented")
}

//getEncryption here you get the ServerSide encryption for the service itself
func (s *S3BlobStorage) getEncryption() encrypt.ServerSide {
	return encrypt.DefaultPBKDF([]byte(s.Password), []byte(s.Bucket+s.Tenant))
}
