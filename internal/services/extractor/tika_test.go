package extractor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/services/interfaces/mocks"
	"github.com/willie68/GoBlobStore/pkg/model"
)

func TestTikaExtractorTxt(t *testing.T) {
	ast := assert.New(t)

	ext := NewExtractor("tika")
	tk, ok := ext.(*Tika)
	ast.True(ok)

	stg := &mocks.BlobStorage{}
	bd := &model.BlobDescription{
		StoreID:       "mcs",
		ContentLength: 1234,
		ContentType:   "text/plain",
		CreationDate:  time.Now().Unix(),
		Filename:      "test.txt",
		TenantID:      "mcs",
		BlobID:        "12345678",
		LastAccess:    time.Now().Unix(),
		Retention:     0,
	}
	stg.EXPECT().HasBlob(mock.Anything).Return(true, nil)
	stg.EXPECT().GetBlobDescription(mock.Anything).Return(bd, nil)
	cfg := config.Extractor{}
	err := tk.Init(cfg, stg)
	ast.Nil(err)

	bdm, err := tk.Metadata("12345678")
	ast.Nil(err)
	ast.NotNil(bdm)
	ast.Equal("test.txt", bdm["filename"])
}
