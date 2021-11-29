package fastcache

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	rootpath = "../../../testdata/blobcache"
)

func initTest(t *testing.T) {
	// getting the zip file and extracting it into the file system
	os.MkdirAll(rootpath, os.ModePerm)
}

func clear(t *testing.T) {
	// getting the zip file and extracting it into the file system
	err := removeContents(rootpath)
	assert.Nil(t, err)
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func getStoreageDao(t *testing.T) *FastCache {
	dao := FastCache{
		RootPath:   rootpath,
		MaxCount:   20,
		MaxRamSize: 1 * 1024 * 1024 * 1024,
	}
	err := dao.Init()
	if err != nil {
		t.Fatal(err)
	}
	return &dao
}

func TestNotFound(t *testing.T) {
	initTest(t)
	clear(t)
	dao := getStoreageDao(t)
	ast := assert.New(t)

	ok, err := dao.HasBlob("wrongid")
	ast.Nil(err)
	ast.False(ok)

	_, err = dao.GetBlobDescription("wrongid")
	ast.NotNil(err)

	var b bytes.Buffer
	err = dao.RetrieveBlob("wrongid", &b)
	ast.NotNil(err)
}

func TestList(t *testing.T) {
	initTest(t)
	clear(t)
	dao := getStoreageDao(t)

	blobs := make([]string, 0)
	err := dao.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(blobs))

	dao.Close()
}

func TestInfo(t *testing.T) {
	initTest(t)
	dao := getStoreageDao(t)
	ast := assert.New(t)

	ok, err := dao.HasBlob("004b4987-42fb-43e4-8e13-d6994ce0e6f1")
	ast.Nil(err)
	ast.True(ok)

	ok, err = dao.HasBlob("0000fc02-050a-418a-a701-efd814aa6b36")
	ast.Nil(err)
	ast.True(ok)

	info, err := dao.GetBlobDescription("004b4987-42fb-43e4-8e13-d6994ce0e6f1")
	if err != nil {
		t.Fatal(err)
	}
	ast.Equal("004b4987-42fb-43e4-8e13-d6994ce0e6f1", info.BlobID)

	info, err = dao.GetBlobDescription("0000fc02-050a-418a-a701-efd814aa6b36")
	if err != nil {
		t.Fatal(err)
	}
	ast.Equal("0000fc02-050a-418a-a701-efd814aa6b36", info.BlobID)

	dao.Close()
}

func TestCRD(t *testing.T) {
	dao := getStoreageDao(t)
	uuid := utils.GenerateID()

	b := model.BlobDescription{
		BlobID:        uuid,
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		Filename:      "test.txt",
		LastAccess:    int(time.Now().UnixNano() / 1000000),
		Retention:     180000,
		Properties:    make(map[string]interface{}),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = "MCS"

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

	var buf bytes.Buffer

	err = dao.RetrieveBlob(id, &buf)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "this is a blob content", buf.String())

	err = dao.DeleteBlob(id)
	if err != nil {
		t.Fatal(err)
	}

	dao.Close()
}

func TestMaxCount(t *testing.T) {
	initTest(t)
	clear(t)
	ast := assert.New(t)

	dao := getStoreageDao(t)
	ids := make([]string, 0)

	for i := 0; i < 20; i++ {
		b := getBlobDescription(strconv.Itoa(i))
		ids = append(ids, b.BlobID)

		r := strings.NewReader("this is a blob content")
		id, err := dao.StoreBlob(b, r)
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
	}

	files := getFiles(t, dao)
	ast.Equal(20, len(files))

	b := *getBlobDescription(strconv.Itoa(20))
	ids = append(ids, b.BlobID)

	r := strings.NewReader("this is a blob content")
	id, err := dao.StoreBlob(&b, r)
	ast.Nil(err)
	assert.NotNil(t, id)
	assert.Equal(t, id, b.BlobID)

	info, err := dao.GetBlobDescription(id)
	ast.Nil(err)
	assert.Equal(t, id, info.BlobID)

	files = getFiles(t, dao)
	ast.Equal(20, len(files))

	for _, id := range files {
		var buf bytes.Buffer
		err := dao.RetrieveBlob(id, &buf)
		ast.Nil(err)
		if err == nil {

			assert.Equal(t, "this is a blob content", buf.String())

			err = dao.DeleteBlob(id)
			ast.Nil(err)
		}
	}

	dao.Close()
	clear(t)
}

func getBlobDescription(id string) *model.BlobDescription {
	uuid := utils.GenerateID()

	b := model.BlobDescription{
		BlobID:        uuid,
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		Filename:      "id",
		LastAccess:    int(time.Now().UnixNano() / 1000000),
		Retention:     180000,
		Properties:    make(map[string]interface{}),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = "MCS"
	return &b
}

func getFiles(t *testing.T, dao *FastCache) []string {
	blobs := make([]string, 0)
	err := dao.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	return blobs
}
