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
	"github.com/willie68/GoBlobStore/internal/services/fastcache"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/internal/services/simplefile"
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
	tntmgr  interfaces.TenantManager
)

func initTest(t *testing.T) {
	ast := assert.New(t)
	stgsrv := &simplefile.BlobStorage{
		RootPath: blbPath,
		Tenant:   tenant,
	}
	stgsrv.Init()
	cchsrv := &fastcache.FastCache{
		RootPath:   cchPath,
		MaxCount:   blbcount,
		MaxRAMSize: 1 * 1024 * 1024,
	}
	cchsrv.Init()
	bcksrv := &simplefile.BlobStorage{
		RootPath: bckPath,
		Tenant:   tenant,
	}
	bcksrv.Init()

	tntmgr = &simplefile.TenantManager{
		RootPath: blbPath,
	}
	err := tntmgr.Init()
	ast.Nil(err)

	err = tntmgr.AddTenant(tenant)
	ast.Nil(err)

	main = &MainStorage{
		StgSrv: stgsrv,
		CchSrv: cchsrv,
		BckSrv: bcksrv,
		Tenant: tenant,
		TntMgr: tntmgr,
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

func TestSingleFile(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)

	b, err := createBlob(ast, "01")
	ast.Nil(err)
	ast.NotNil(b)

	bd, err := main.GetBlobDescription(b.BlobID)
	ast.Nil(err)
	ast.NotNil(bd)
	time.Sleep(1 * time.Second)

	size := tntmgr.GetSize(main.GetTenant())
	ast.Equal(bd.ContentLength, size)

	err = main.DeleteBlob(b.BlobID)
	ast.Nil(err)
	time.Sleep(1 * time.Second)

	size = tntmgr.GetSize(main.GetTenant())
	ast.Equal(int64(0), size)
}

func TestManyFiles(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)

	ids := createManyFiles(ast)

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

		ok, err := bMain.StgSrv.HasBlob(id)
		ast.Nil(err)
		ast.False(ok)

		ok, err = bMain.BckSrv.HasBlob(id)
		ast.Nil(err)
		ast.False(ok)

		ok, err = bMain.CchSrv.HasBlob(id)
		ast.Nil(err)
		ast.False(ok)
	}
}

func createManyFiles(ast *assert.Assertions) []model.BlobDescription {
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
	return ids
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

func TestMaster(t *testing.T) {
	t.SkipNow()
	for i := 0; i < 100; i++ {
		t.Logf("%d test iteration", i)
		TestAutoRestoreByDescription(t)
	}
}

func TestAutoRestoreByDescription(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)
	bMain := main.(*MainStorage)
	bMain.Bcksyncmode = true
	// disable caching
	cchsrv := bMain.CchSrv
	bMain.CchSrv = nil
	cchsrv.Close()

	ast.Nil(bMain.CchSrv)

	is := "12345"
	// adding a blob
	b, err := createBlob(ast, is)
	ast.Nil(err)
	ast.NotNil(b)

	id := b.BlobID
	ok, err := bMain.StgSrv.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)

	// remove it from primary storage
	err = bMain.StgSrv.DeleteBlob(id)
	ast.Nil(err)
	time.Sleep(1 * time.Millisecond)
	ok, err = bMain.StgSrv.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	bd, err := bMain.StgSrv.GetBlobDescription(id)
	ast.NotNil(err)
	ast.Nil(bd)

	// getting blobdescription
	bd, err = main.GetBlobDescription(id)
	ast.Nil(err)
	ast.NotNil(bd)
	time.Sleep(10 * time.Millisecond)

	// checking if present in prime storage
	ok, err = bMain.StgSrv.HasBlob(id)
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
	cchSrv := bMain.CchSrv
	bMain.CchSrv = nil
	cchSrv.Close()

	ast.Nil(bMain.CchSrv)

	is := "wi_12345"
	// adding a blob
	b, err := createBlob(ast, is)
	ast.Nil(err)
	ast.NotNil(b)

	id := b.BlobID
	ok, err := bMain.StgSrv.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)

	// remove it from primary storage
	bMain.StgSrv.DeleteBlob(id)
	time.Sleep(1 * time.Second)
	ok, err = bMain.StgSrv.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	bd, err := bMain.StgSrv.GetBlobDescription(id)
	ast.NotNil(err)
	ast.Nil(bd)

	// getting blobdescription
	var buf bytes.Buffer
	err = main.RetrieveBlob(b.BlobID, &buf)
	ast.Nil(err)
	time.Sleep(1 * time.Second)

	// checking if present in primstorage
	ok, err = bMain.StgSrv.HasBlob(id)
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
	cchSrv := bMain.CchSrv
	bMain.CchSrv = nil
	cchSrv.Close()

	ast.Nil(bMain.CchSrv)

	is := "12345"
	// adding a blob
	b, err := createBlob(ast, is)
	ast.Nil(err)
	ast.NotNil(b)

	id := b.BlobID
	ok, err := bMain.StgSrv.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)

	// remove it from primary storage
	bMain.StgSrv.DeleteBlob(id)
	time.Sleep(1 * time.Second)
	ok, err = bMain.StgSrv.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	bd, err := bMain.StgSrv.GetBlobDescription(id)
	ast.NotNil(err)
	ast.Nil(bd)

	ok, err = bMain.StgSrv.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	// getting blobdescription
	ok, err = main.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)
	time.Sleep(1 * time.Second)

	// checking if present in primstorage
	ok, err = bMain.StgSrv.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)
}
