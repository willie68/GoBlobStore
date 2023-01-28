package noindex

import (
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

// NoIndexName name of the noindex index service
const NoIndexName = "noindex"

// checking interface compatibility
var _ interfaces.Index = &Index{}
var _ interfaces.IndexBatch = &IndexBatch{}

// Index NOP indexer
type Index struct {
}

// IndexBatch NOP batch indexer
type IndexBatch struct {
}

// Init init this
func (i *Index) Init() error {
	return nil
}

// Search search for nothing
func (i *Index) Search(_ string, _ func(id string) bool) error {
	return nil
}

// Index NOP Index single
func (i *Index) Index(_ string, _ model.BlobDescription) error {
	return nil
}

// NewBatch creates a NOP batch
func (i *Index) NewBatch() interfaces.IndexBatch {
	return &IndexBatch{}
}

// Add add something to NOP Batch
func (i *IndexBatch) Add(_ string, _ model.BlobDescription) error {
	return nil
}

// Index NOP batch index
func (i *IndexBatch) Index() error {
	return nil
}
