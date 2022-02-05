package interfaces

import (
	"io"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// this is the interface for the factory which will create tenant specifig storage implementations
type StorageFactory interface {
	Init(storage config.Engine, rtnm RetentionManager) error
	GetStorageDao(tenant string) (BlobStorageDao, error)
	Close() error
}

// BlobStoreDao this is the interface which all implementation of a blob storage engine has to fulfill
type BlobStorageDao interface {
	Init() error       // initialise this dao
	GetTenant() string // get the tenant id

	GetBlobs(callback func(id string) bool) error // getting a list of blob from the storage

	// CRUD operation on the blob files
	StoreBlob(b *model.BlobDescription, r io.Reader) (string, error) // storing a blob to the storage system
	HasBlob(id string) (bool, error)                                 // checking, if a blob is present
	GetBlobDescription(id string) (*model.BlobDescription, error)    // getting the description of the file
	UpdateBlobDescription(id string, b *model.BlobDescription) error // updating the blob description
	RetrieveBlob(id string, w io.Writer) error                       // retrieving the binary data from the storage system
	DeleteBlob(id string) error                                      // removing a blob from the storage system

	//Retentionrelated methods
	GetAllRetentions(callback func(r model.RetentionEntry) bool) error // for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
	AddRetention(r *model.RetentionEntry) error
	GetRetention(id string) (model.RetentionEntry, error)
	DeleteRetention(id string) error
	ResetRetention(id string) error

	Close() error // closing the storage
}
