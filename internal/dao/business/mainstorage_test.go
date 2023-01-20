package business

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/dao/fastcache"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	rootFilePrefix = "../../../testdata/mai"
	tenant         = "test"
	blbcount       = 100
)

var (
	blbPath = filepath.Join(rootFilePrefix, "blbstg")
	cchPath = filepath.Join(rootFilePrefix, "blbcch")
	bckPath = filepath.Join(rootFilePrefix, "bckstg")
	main    interfaces.BlobStorage
)

func initTest(_ *testing.T) {
	stgDao := &simplefile.BlobStorage{
		RootPath: blbPath,
		Tenant:   tenant,
	}
	stgDao.Init()
	cchDao := &fastcache.FastCache{
		RootPath:   cchPath,
		MaxCount:   blbcount,
		MaxRAMSize: 1 * 1024 * 1024,
	}
	cchDao.Init()
	bckDao := &simplefile.BlobStorage{
		RootPath: bckPath,
		Tenant:   tenant,
	}
	bckDao.Init()

	main = &MainStorage{
		StgDao: stgDao,
		CchDao: cchDao,
		BckDao: bckDao,
		Tenant: tenant,
	}

	main.Init()
}

func clear(t *testing.T) {
	err := os.RemoveAll(rootFilePrefix)
	assert.Nil(t, err)
}

func createBlobDescription(id string) model.BlobDescription {
	uuid := utils.GenerateID()

	b := model.BlobDescription{
		BlobID:        uuid,
		StoreID:       tenant,
		TenantID:      tenant,
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      fmt.Sprintf("test_%s.txt", id),
		LastAccess:    time.Now().UnixMilli(),
		Retention:     180000,
		Properties:    make(map[string]any),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = tenant
	b.Properties["X-externalid"] = id
	b.Properties["X-id"] = uuid
	return b
}

func TestTenant(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)

	ast.Equal(tenant, main.GetTenant())
}

func TestManyFiles(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)

	ids := make([]model.BlobDescription, 0)
	for i := 0; i < blbcount; i++ {
		if i%100 == 0 {
			if i%10000 == 0 {
				fmt.Printf(", go routines: %d\r\n", runtime.NumGoroutine())
				fmt.Printf("%d", i/10000)
			}
			fmt.Print(".")
		}
		is := strconv.Itoa(i)

		b, err := createBlob(ast, is)
		ast.Nil(err)
		ast.NotNil(b)

		ids = append(ids, b)
	}
	fmt.Printf(", go routines: %d\r\n", runtime.NumGoroutine())
	for i, b := range ids {
		if i%100 == 0 {
			if i%10000 == 0 {
				fmt.Printf(", go routines: %d\r\n", runtime.NumGoroutine())
				fmt.Printf("%d", i/10000)
			}
			fmt.Print(".")
		}
		checkBlob(ast, b)
	}

	fmt.Printf(", go routines: %d\r\n", runtime.NumGoroutine())

	blobs := make([]string, 0)

	err := main.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	ast.Nil(err)

	bMain := main.(*MainStorage)

	for _, id := range blobs {
		found := false
		for _, b := range ids {
			if b.BlobID == id {
				found = true
			}
		}
		ast.True(found, "didn't found %s", id)
		err := main.DeleteBlob(id)
		ast.Nil(err)

		ok, err := bMain.StgDao.HasBlob(id)
		ast.Nil(err)
		ast.False(ok)

		ok, err = bMain.BckDao.HasBlob(id)
		ast.Nil(err)
		ast.False(ok)

		ok, err = bMain.CchDao.HasBlob(id)
		ast.Nil(err)
		ast.False(ok)
	}
}

func createBlob(ast *assert.Assertions, is string) (model.BlobDescription, error) {
	b := createBlobDescription(is)
	payload := fmt.Sprintf("this is a blob content of %s", is)
	b.BlobURL = payload
	b.ContentLength = int64(len(payload))
	r := strings.NewReader(payload)
	id, err := main.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	ast.Equal(id, b.BlobID)

	return b, err
}

func checkBlob(ast *assert.Assertions, b model.BlobDescription) {
	ok, err := main.HasBlob(b.BlobID)
	ast.Nil(err, fmt.Sprintf("id: %s", b.BlobID))
	ast.True(ok)

	info, err := main.GetBlobDescription(b.BlobID)
	ast.Nil(err, fmt.Sprintf("id: %s", b.BlobID))
	ast.Equal(b.BlobID, info.BlobID)

	var buf bytes.Buffer

	err = main.RetrieveBlob(b.BlobID, &buf)
	ast.Nil(err)

	jsn, err := json.Marshal(b)
	ast.Nil(err)

	ast.Equal(b.BlobURL, buf.String(), fmt.Sprintf("payload doesn't match: %s", jsn))
}

func TestAutoRestoreByDescription(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)
	bMain := main.(*MainStorage)
	bMain.Bcksyncmode = true
	// disable caching
	cchDao := bMain.CchDao
	bMain.CchDao = nil
	cchDao.Close()

	ast.Nil(bMain.CchDao)

	is := "12345"
	// adding a blob
	b, err := createBlob(ast, is)
	ast.Nil(err)
	ast.NotNil(b)

	id := b.BlobID
	ok, err := bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)

	// remove it from primary storage
	bMain.StgDao.DeleteBlob(id)
	time.Sleep(1 * time.Second)
	ok, err = bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	bd, err := bMain.StgDao.GetBlobDescription(id)
	ast.NotNil(err)
	ast.Nil(bd)

	// getting blobdescription
	bd, err = main.GetBlobDescription(id)
	ast.Nil(err)
	ast.NotNil(bd)
	time.Sleep(1 * time.Second)

	// checking if present in primstorage
	ok, err = bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)
}

func TestAutoRestoreByContent(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)
	bMain := main.(*MainStorage)
	bMain.Bcksyncmode = true
	// disable caching
	cchDao := bMain.CchDao
	bMain.CchDao = nil
	cchDao.Close()

	ast.Nil(bMain.CchDao)

	is := "wi_12345"
	// adding a blob
	b, err := createBlob(ast, is)
	ast.Nil(err)
	ast.NotNil(b)

	id := b.BlobID
	ok, err := bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)

	// remove it from primary storage
	bMain.StgDao.DeleteBlob(id)
	time.Sleep(1 * time.Second)
	ok, err = bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	bd, err := bMain.StgDao.GetBlobDescription(id)
	ast.NotNil(err)
	ast.Nil(bd)

	// getting blobdescription
	var buf bytes.Buffer
	err = main.RetrieveBlob(b.BlobID, &buf)
	ast.Nil(err)
	time.Sleep(1 * time.Second)

	// checking if present in primstorage
	ok, err = bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)
}

func TestAutoRestoreByHasId(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)
	bMain := main.(*MainStorage)
	bMain.Bcksyncmode = true
	// disable caching
	cchDao := bMain.CchDao
	bMain.CchDao = nil
	cchDao.Close()

	ast.Nil(bMain.CchDao)

	is := "12345"
	// adding a blob
	b, err := createBlob(ast, is)
	ast.Nil(err)
	ast.NotNil(b)

	id := b.BlobID
	ok, err := bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)

	// remove it from primary storage
	bMain.StgDao.DeleteBlob(id)
	time.Sleep(1 * time.Second)
	ok, err = bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	bd, err := bMain.StgDao.GetBlobDescription(id)
	ast.NotNil(err)
	ast.Nil(bd)

	ok, err = bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	// getting blobdescription
	ok, err = main.HasBlob(b.BlobID)
	ast.Nil(err)
	ast.True(ok)
	time.Sleep(1 * time.Second)

	// checking if present in primstorage
	ok, err = bMain.StgDao.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)
}
