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
	rootpath = "R:/blbcch/"
)

func initTest(_ *testing.T) {
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
		MaxRAMSize: 1 * 1024 * 1024 * 1024,
	}
	err := dao.Init()
	if err != nil {
		t.Fatal(err)
	}
	return &dao
}

func TestAutoCreatPath(t *testing.T) {
	ast := assert.New(t)

	if _, err := os.Stat(rootpath); err == nil {
		err := os.RemoveAll(rootpath)
		ast.Nil(err)
	}
	getStoreageDao(t)
	_, err := os.Stat(rootpath)

	ast.Nil(err)
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

func TestCRD(t *testing.T) {
	dao := getStoreageDao(t)
	uuid := utils.GenerateID()

	b := model.BlobDescription{
		BlobID:        uuid,
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      "test.txt",
		LastAccess:    time.Now().UnixMilli(),
		Retention:     180000,
		Properties:    make(map[string]any),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = "MCS"

	ast := assert.New(t)

	r := strings.NewReader("this is a blob content")
	id, err := dao.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	ast.Equal(id, b.BlobID)

	info, err := dao.GetBlobDescription(id)
	ast.Nil(err)
	ast.Equal(id, info.BlobID)

	var buf bytes.Buffer

	err = dao.RetrieveBlob(id, &buf)
	ast.Nil(err)

	ast.Equal("this is a blob content", buf.String())

	// update
	b.Properties["X-tenant"] = "MCS_2"
	err = dao.UpdateBlobDescription(id, &b)
	ast.Nil(err)

	info, err = dao.GetBlobDescription(id)
	ast.Nil(err)
	ast.Equal(id, info.BlobID)
	ast.Equal("MCS_2", info.Properties["X-tenant"])

	err = dao.DeleteBlob(id)
	ast.Nil(err)

	dao.Close()
}

func TestMaxCount(t *testing.T) {
	initTest(t)
	clear(t)
	ast := assert.New(t)

	dao := getStoreageDao(t)
	ids := make([]string, 0)

	for i := 0; i < 50; i++ {
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

			//			err = dao.DeleteBlob(id)
			//			ast.Nil(err)
		}
	}

	dao.Close()
	//clear(t)
}

func getBlobDescription(id string) *model.BlobDescription {
	uuid := utils.GenerateID()

	b := model.BlobDescription{
		BlobID:        uuid,
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      id,
		LastAccess:    time.Now().UnixMilli(),
		Retention:     180000,
		Properties:    make(map[string]any),
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

func TestSameUUID(t *testing.T) {
	initTest(t)
	clear(t)
	ast := assert.New(t)

	dao := getStoreageDao(t)
	uuid := utils.GenerateID()

	b := model.BlobDescription{
		BlobID:        uuid,
		StoreID:       "MCS",
		TenantID:      "MCS",
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      "test.txt",
		LastAccess:    time.Now().UnixMilli(),
		Retention:     180000,
		Properties:    make(map[string]any),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = "MCS"

	r := strings.NewReader("this is a blob content")
	id, err := dao.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	ast.Equal(id, b.BlobID)

	info, err := dao.GetBlobDescription(id)
	ast.Nil(err)
	ast.Equal(id, info.BlobID)

	var buf bytes.Buffer

	err = dao.RetrieveBlob(id, &buf)
	ast.Nil(err)

	ast.Equal("this is a blob content", buf.String())

	r = strings.NewReader("this is a blob content")
	id, err = dao.StoreBlob(&b, r)
	ast.NotNil(err)
	ast.Equal(os.ErrExist, err)
	ast.Equal(id, b.BlobID)

	err = dao.DeleteBlob(id)
	ast.Nil(err)
	dao.Close()
}
