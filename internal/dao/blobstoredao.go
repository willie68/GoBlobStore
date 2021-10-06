package dao

import (
	"io"

	"github.com/willie68/GoBlobStore/pkg/model"
)

// BlobStoreDao this is the interface which all implementation of a blob storage engine has to fulfill
type BlobStorageDao interface {
	Init() error // initialise this dao

	GetBlobs(offset int, limit int) ([]string, error) // getting a list of blob from the filesystem using offset and limit

	StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) // storing a blob to the storage system
	GetBlobDescription(id string) (*model.BlobDescription, error)    // getting the description of the file
	RetrieveBlob(idStr string, writer io.Writer) error               // retrieving the binary data from the storage system

	DeleteBlob(id string) error // removing a blob from the storage system

	Close() error // closing the blob storage
}

// TenantDao is the part of the daos which will adminitrate the tenant part of a storage system
type TenantDao interface {
	AddTenant(tenant string) error
	RemoveTenant(tenant string) error
	HasTenant(tenant string) bool
}
