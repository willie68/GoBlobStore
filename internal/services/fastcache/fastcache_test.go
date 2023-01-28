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
	rootpath = "../../../testdata/fc"
)

func initTest(t *testing.T) {
	// getting the zip file and extracting it into the file system
	err := os.MkdirAll(rootpath, os.ModePerm)
	assert.Nil(t, err)
}

func clear(t *testing.T) {
	// getting the zip file and extracting it into the file system
	err := os.RemoveAll(rootpath)
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

func getStoreageSrv(t *testing.T) *FastCache {
	srv := FastCache{
		RootPath:   rootpath,
		MaxCount:   20,
		MaxRAMSize: 1 * 1024 * 1024 * 1024,
	}
	err := srv.Init()
	if err != nil {
		t.Fatal(err)
	}
	return &srv
}

func TestAutoCreatPath(t *testing.T) {
	ast := assert.New(t)

	if _, err := os.Stat(rootpath); err == nil {
		err := os.RemoveAll(rootpath)
		ast.Nil(err)
	}
	getStoreageSrv(t)
	_, err := os.Stat(rootpath)

	ast.Nil(err)
}

func TestNotFound(t *testing.T) {
	initTest(t)
	clear(t)
	srv := getStoreageSrv(t)
	ast := assert.New(t)

	ok, err := srv.HasBlob("wrongid")
	ast.Nil(err)
	ast.False(ok)

	_, err = srv.GetBlobDescription("wrongid")
	ast.NotNil(err)

	var b bytes.Buffer
	err = srv.RetrieveBlob("wrongid", &b)
	ast.NotNil(err)
}

func TestList(t *testing.T) {
	initTest(t)
	clear(t)
	srv := getStoreageSrv(t)

	blobs := make([]string, 0)
	err := srv.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 0, len(blobs))

	err = srv.Close()
	assert.Nil(t, err)
}

func TestCRD(t *testing.T) {
	srv := getStoreageSrv(t)
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
	id, err := srv.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	ast.Equal(id, b.BlobID)

	info, err := srv.GetBlobDescription(id)
	ast.Nil(err)
	ast.Equal(id, info.BlobID)

	var buf bytes.Buffer

	err = srv.RetrieveBlob(id, &buf)
	ast.Nil(err)

	ast.Equal("this is a blob content", buf.String())

	// update
	b.Properties["X-tenant"] = "MCS_2"
	err = srv.UpdateBlobDescription(id, &b)
	ast.Nil(err)

	info, err = srv.GetBlobDescription(id)
	ast.Nil(err)
	ast.Equal(id, info.BlobID)
	ast.Equal("MCS_2", info.Properties["X-tenant"])

	err = srv.DeleteBlob(id)
	ast.Nil(err)

	err = srv.Close()
	ast.Nil(err)
}

func TestMaxCount(t *testing.T) {
	initTest(t)
	clear(t)
	ast := assert.New(t)

	srv := getStoreageSrv(t)
	ids := make([]string, 0)

	for i := 0; i < 50; i++ {
		b := getBlobDescription(strconv.Itoa(i))
		ids = append(ids, b.BlobID)

		r := strings.NewReader("this is a blob content")
		id, err := srv.StoreBlob(b, r)
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, id)
		assert.Equal(t, id, b.BlobID)

		info, err := srv.GetBlobDescription(id)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, id, info.BlobID)
	}

	files := getFiles(t, srv)
	ast.Equal(20, len(files))

	b := *getBlobDescription(strconv.Itoa(20))
	ids = append(ids, b.BlobID)

	r := strings.NewReader("this is a blob content")
	id, err := srv.StoreBlob(&b, r)
	ast.Nil(err)
	assert.NotNil(t, id)
	assert.Equal(t, id, b.BlobID)

	info, err := srv.GetBlobDescription(id)
	ast.Nil(err)
	assert.Equal(t, id, info.BlobID)

	files = getFiles(t, srv)
	ast.Equal(20, len(files))

	for _, id := range files {
		var buf bytes.Buffer
		err := srv.RetrieveBlob(id, &buf)
		ast.Nil(err)
		if err == nil {
			assert.Equal(t, "this is a blob content", buf.String())
		}
	}

	err = srv.Close()
	ast.Nil(err)
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

func getFiles(t *testing.T, srv *FastCache) []string {
	blobs := make([]string, 0)
	err := srv.GetBlobs(func(id string) bool {
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

	srv := getStoreageSrv(t)
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
	id, err := srv.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	ast.Equal(id, b.BlobID)

	info, err := srv.GetBlobDescription(id)
	ast.Nil(err)
	ast.Equal(id, info.BlobID)

	var buf bytes.Buffer

	err = srv.RetrieveBlob(id, &buf)
	ast.Nil(err)

	ast.Equal("this is a blob content", buf.String())

	r = strings.NewReader("this is a blob content")
	id, err = srv.StoreBlob(&b, r)
	ast.NotNil(err)
	ast.Equal(os.ErrExist, err)
	ast.Equal(id, b.BlobID)

	err = srv.DeleteBlob(id)
	ast.Nil(err)

	err = srv.Close()
	ast.Nil(err)
}

func TestNotImplemented(t *testing.T) {
	initTest(t)
	clear(t)
	srv := getStoreageSrv(t)
	ast := assert.New(t)

	ast.Equal(DefaultTnt, srv.GetTenant())

	err := srv.SearchBlobs("hallo", func(id string) bool {
		return true
	})
	ast.ErrorAs(err, &errNotImplemented)

	err = srv.GetAllRetentions(func(r model.RetentionEntry) bool {
		return true
	})
	ast.ErrorAs(err, &errNotImplemented)

	err = srv.AddRetention(&model.RetentionEntry{
		Filename: "string",
		TenantID: DefaultTnt,
		BlobID:   "12345678",
	})
	ast.ErrorAs(err, &errNotImplemented)

	_, err = srv.GetRetention("12345678")
	ast.ErrorAs(err, &errNotImplemented)

	err = srv.DeleteRetention("12345678")
	ast.ErrorAs(err, &errNotImplemented)

	err = srv.ResetRetention("12345678")
	ast.ErrorAs(err, &errNotImplemented)

	err = srv.GetLastError()
	ast.Nil(err)
}
