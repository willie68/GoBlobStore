package extractor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/services/interfaces/mocks"
	"github.com/willie68/GoBlobStore/pkg/model"
)

func TestNoExtractor(t *testing.T) {
	ast := assert.New(t)

	ext := &NoExtractor{}

	stg := &mocks.BlobStorage{}
	stg.EXPECT().GetBlobDescription(mock.Anything).Return(&model.BlobDescription{
		BlobID: "12345678",
	}, nil)
	stg.EXPECT().HasBlob(mock.Anything).Return(true, nil)

	err := ext.Init(config.Extractor{
		Service: "nn",
	}, stg)
	ast.Nil(err)

	meta, err := ext.Metadata("12345678")
	ast.Nil(err)
	ast.NotNil(meta)
	ast.Equal("12345678", meta["blobID"])

	txt, err := ext.Fulltext("12345678")
	ast.NotNil(txt)
	ast.Nil(err)
	ast.Equal("", txt)

	err = ext.Close()
	ast.Nil(err)
}

func TestWrongID(t *testing.T) {
	ast := assert.New(t)

	ext := &NoExtractor{}

	stg := &mocks.BlobStorage{}
	stg.EXPECT().GetBlobDescription(mock.Anything).Return(nil, errors.New("wrong id"))
	stg.EXPECT().HasBlob(mock.Anything).Return(false, nil)
	err := ext.Init(config.Extractor{
		Service: "nn",
	}, stg)
	ast.Nil(err)

	meta, err := ext.Metadata("12345678")
	ast.NotNil(err)
	ast.Nil(meta)

	txt, err := ext.Fulltext("12345678")
	ast.NotNil(err)
	ast.Equal("", txt)

	err = ext.Close()
	ast.Nil(err)
}
