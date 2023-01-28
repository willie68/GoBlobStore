package interfaces

import (
	"io"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// StorageFactory this is the interface for the factory which will create tenant specific storage implementations
type StorageFactory interface {
	Init(storage config.Engine, rtnm RetentionManager) error
	GetStorage(tenant string) (BlobStorage, error)
	RemoveStorage(tenant string) error
	Close() error
}

// BlobStorage this is the interface which all implementation of a blob storage engine has to fulfill
//
//go:generate mockery --name=BlobStorage --outpkg=mocks --with-expecter
type BlobStorage interface {
	Init() error       // initialize this service
	GetTenant() string // get the tenant id

	GetBlobs(callback func(id string) bool) error // getting a list of blob from the storage

	// CRUD operation on the blob files
	StoreBlob(b *model.BlobDescription, r io.Reader) (string, error) // storing a blob to the storage system
	HasBlob(id string) (bool, error)                                 // checking, if a blob is present
	GetBlobDescription(id string) (*model.BlobDescription, error)    // getting the description of the file
	UpdateBlobDescription(id string, b *model.BlobDescription) error // updating the blob description
	RetrieveBlob(id string, w io.Writer) error                       // retrieving the binary data from the storage system
	DeleteBlob(id string) error                                      // removing a blob from the storage system
	CheckBlob(id string) (*model.CheckInfo, error)                   // checking a single blob from the storage system

	// Searching for blobs
	SearchBlobs(query string, callback func(id string) bool) error // getting a list of blob from the storage

	// Retention related methods
	GetAllRetentions(callback func(r model.RetentionEntry) bool) error // for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
	AddRetention(r *model.RetentionEntry) error
	GetRetention(id string) (model.RetentionEntry, error)
	DeleteRetention(id string) error
	ResetRetention(id string) error

	GetLastError() error // getting the last error

	Close() error // closing the storage
}
