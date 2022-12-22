package nop

import (
	"errors"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

var _ interfaces.Index = &NOPIndex{}
var _ interfaces.IndexBatch = &NOPIndexBatch{}

type NOPIndex struct{}

type NOPIndexBatch struct{}

func (i *NOPIndex) Init() error {
	return nil
}

func (i *NOPIndex) Search(query string, callback func(id string) bool) error {
	return errors.New("no index defined")
}

func (i *NOPIndex) Index(id string, b model.BlobDescription) error {
	return nil
}

func (i *NOPIndex) NewBatch() interfaces.IndexBatch {
	return &NOPIndexBatch{}
}

func (i *NOPIndexBatch) Add(id string, b model.BlobDescription) error {
	return nil
}

func (i *NOPIndexBatch) Index() error {
	return nil
}
