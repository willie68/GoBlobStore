package interfaces

import "github.com/willie68/GoBlobStore/pkg/model"

type Index interface {
	Search(query string, callback func(id string) bool) error // getting a list of blob from the storage
	Index(id string, b model.BlobDescription) error
}
