package interfaces

import "github.com/willie68/GoBlobStore/pkg/model"

// Index interface for indexer
type Index interface {
	Init() error                                              // initialise the indexer
	Search(query string, callback func(id string) bool) error // getting a list of blob from the storage
	Index(id string, b model.BlobDescription) error           // index a single blob description
	NewBatch() IndexBatch                                     // returning a index batch processor
}

// IndexBatch interface batch index
type IndexBatch interface {
	Add(id string, b model.BlobDescription) error // add a single blob description to this batch
	Index() error                                 // index all added description in one batch and empty this batch
}
