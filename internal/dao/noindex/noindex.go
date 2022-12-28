package noindex

import (
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

var _ interfaces.Index = &Index{}
var _ interfaces.IndexBatch = &IndexBatch{}

type Index struct {
}

type IndexBatch struct {
}

func (i *Index) Init() error {
	return nil
}
func (i *Index) Search(query string, callback func(id string) bool) error {
	return nil
}
func (i *Index) Index(id string, b model.BlobDescription) error {
	return nil
}

func (i *Index) NewBatch() interfaces.IndexBatch {
	return &IndexBatch{}
}

func (i *IndexBatch) Add(id string, b model.BlobDescription) error {
	return nil
}
func (i *IndexBatch) Index() error {
	return nil
}
