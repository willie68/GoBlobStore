package dao

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/utils/slicesutils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

func TestList(t *testing.T) {
	dao := SimpleFileBlobStorageDao{
		RootPath: "../../testdata/blobstorage",
		Tenant:   "EASY",
	}
	err := dao.Init()
	if err != nil {
		t.Fatal(err)
	}

	srcPath, _ := filepath.Abs("../../testdata/blobstorage/EASY")
	assert.Equal(t, srcPath, dao.filepath)

	blobs, err := dao.GetBlobs(0, 10)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 7, len(blobs))
	assert.True(t, slicesutils.Contains(blobs, "004b4987-42fb-43e4-8e13-d6994ce0e6f1"))
	assert.True(t, slicesutils.Contains(blobs, "0000fc02-050a-418a-a701-efd814aa6b36"))

	for _, blobid := range blobs {
		fmt.Println(blobid)
	}
}

func TestInfo(t *testing.T) {
	dao := SimpleFileBlobStorageDao{
		RootPath: "../../testdata/blobstorage",
		Tenant:   "EASY",
	}
	err := dao.Init()
	if err != nil {
		t.Fatal(err)
	}

	info, err := dao.GetBlobDescription("004b4987-42fb-43e4-8e13-d6994ce0e6f1")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "004b4987-42fb-43e4-8e13-d6994ce0e6f1", info.BlobID)

	info, err = dao.GetBlobDescription("0000fc02-050a-418a-a701-efd814aa6b36")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "0000fc02-050a-418a-a701-efd814aa6b36", info.BlobID)

}

func TestCRD(t *testing.T) {
	dao := SimpleFileBlobStorageDao{
		RootPath: "../../testdata/blobstorage",
		Tenant:   "EASY",
	}
	err := dao.Init()
	if err != nil {
		t.Fatal(err)
	}
	b := model.BlobDescription{
		StoreID:       "EASY",
		TenantID:      "EASY",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		Filename:      "test.txt",
		LastAccess:    int(time.Now().UnixNano() / 1000000),
		Retention:     180000,
		Properties:    make(map[string]interface{}),
	}
	b.Properties["X-es-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-es-retention"] = []int{123456}
	b.Properties["X-es-tenant"] = "EASY"

	r := strings.NewReader("this is a blob content")
	id, err := dao.StoreBlob(&b, r)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, id)
	assert.Equal(t, id, b.BlobID)

	info, err := dao.GetBlobDescription(id)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, id, info.BlobID)

	info, err = dao.GetBlobDescription("0000fc02-050a-418a-a701-efd814aa6b36")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "0000fc02-050a-418a-a701-efd814aa6b36", info.BlobID)

}
