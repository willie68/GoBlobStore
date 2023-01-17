package migration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/fastcache"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	rootFilePrefix = "../../../testdata/check/"
	tenant         = "chktnt"
	blbPath        = rootFilePrefix + "blbstg"
	cchPath        = rootFilePrefix + "blbcch"
	bckPath        = rootFilePrefix + "bckstg"
)

type JSONResult struct {
	Tenant       string
	Cache        []CheckResultLine
	CacheCount   int
	Primary      []CheckResultLine
	PrimaryCount int
	Backup       []CheckResultLine
	BackupCount  int
}

var main *business.MainStorageDao

func initChkTest(_ *testing.T) {
	stgDao := &simplefile.SimpleFileBlobStorageDao{
		RootPath: blbPath,
		Tenant:   tenant,
	}
	stgDao.Init()
	cchDao := &fastcache.FastCache{
		RootPath:   cchPath,
		MaxCount:   1000,
		MaxRAMSize: 1 * 1024 * 1024,
	}
	cchDao.Init()
	bckDao := &simplefile.SimpleFileBlobStorageDao{
		RootPath: bckPath,
		Tenant:   tenant,
	}
	bckDao.Init()

	main = &business.MainStorageDao{
		StgDao:      stgDao,
		CchDao:      cchDao,
		BckDao:      bckDao,
		Tenant:      tenant,
		Bcksyncmode: true,
	}

	main.Init()
}

func clear(t *testing.T) {
	// getting the zip file and extracting it into the file system
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
		Properties:    make(map[string]interface{}),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = tenant
	b.Properties["X-externalid"] = id
	b.Properties["X-id"] = uuid
	return b
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

	json, err := json.Marshal(b)
	ast.Nil(err)

	ast.Equal(b.BlobURL, buf.String(), fmt.Sprintf("payload doesn't match: %s", json))
}

func getResult(id string, res []CheckResultLine) (CheckResultLine, bool) {
	for _, r := range res {
		if id == r.ID {
			return r, true
		}
	}
	return CheckResultLine{}, false
}

func prepare(ast *assert.Assertions) []string {
	ids := make([]model.BlobDescription, 0)
	for i := 0; i < 100; i++ {
		is := strconv.Itoa(i)

		b, err := createBlob(ast, is)
		ast.Nil(err)
		ast.NotNil(b)

		ids = append(ids, b)
	}

	for _, b := range ids {
		checkBlob(ast, b)
	}

	blobs := make([]string, 0)

	err := main.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(100, len(blobs))
	return blobs
}

func check(ast *assert.Assertions) JSONResult {
	cctx := CheckContext{
		TenantID: main.Tenant,
		Primary:  main.StgDao,
		Backup:   main.BckDao,
		Cache:    main.CchDao,
	}

	file, err := cctx.CheckStorage()
	ast.Nil(err)
	ast.NotNil(file)

	byteValue, err := ioutil.ReadFile(file)
	ast.Nil(err)
	var res JSONResult
	err = json.Unmarshal(byteValue, &res)
	ast.Nil(err)

	// checking error attemps
	// cache invalid
	return res
}

func buildFilename(path string, tnt string, id string, ext string) (string, error) {
	fp := path
	fp = filepath.Join(fp, tnt)
	fp = filepath.Join(fp, id[:2])
	fp = filepath.Join(fp, id[2:4])
	err := os.MkdirAll(fp, os.ModePerm)
	if err != nil {
		return "", err
	}
	return filepath.Join(fp, fmt.Sprintf("%s%s", id, ext)), nil
}

func TestCheck(t *testing.T) {
	clear(t)
	initChkTest(t)

	ast := assert.New(t)
	ast.NotNil(main)

	// prepare tests
	blobs := prepare(ast)

	// Test1: Delete Blob only from primary storage
	Test1ID := blobs[0]
	main.StgDao.DeleteBlob(Test1ID)

	// Test2: Delete Blob from backup storage
	Test2ID := blobs[1]
	main.BckDao.DeleteBlob(Test2ID)

	// Test3: Change Blob content in backup storage
	Test3ID := blobs[2]
	fp, err := buildFilename(bckPath, tenant, Test3ID, ".bin")
	ast.Nil(err)
	_, err = os.Stat(fp)
	ast.Nil(err)
	ast.False(errors.Is(err, os.ErrNotExist))
	err = os.WriteFile(fp, []byte("changed content"), 0644)
	ast.Nil(err)

	// Test4: Change Blob content in primary storage
	Test4ID := blobs[3]
	fp, err = buildFilename(blbPath, tenant, Test4ID, ".bin")
	ast.Nil(err)
	_, err = os.Stat(fp)
	ast.Nil(err)
	ast.False(errors.Is(err, os.ErrNotExist))
	err = os.WriteFile(fp, []byte("changed content"), 0644)
	ast.Nil(err)

	// Test5: Delete Blob only from cache
	Test5ID := blobs[4]
	main.CchDao.DeleteBlob(Test5ID)

	time.Sleep(1 * time.Second)
	// checking
	res := check(ast)

	// nominal
	ast.Equal(99, res.CacheCount, "cache count")
	ast.Equal(99, res.PrimaryCount, "primary count")
	ast.Equal(99, res.BackupCount, "backup count")

	// Test 1: cache inconsistent
	r, ok := getResult(Test1ID, res.Cache)
	ast.True(ok)
	ast.Equal(true, r.HasError)

	// and Backup has the entry for this
	r, ok = getResult(Test1ID, res.Backup)
	ast.True(ok)
	ast.Equal(true, r.HasError)

	// Test 2: primary has InBackup false flag
	r, ok = getResult(Test2ID, res.Primary)
	ast.True(ok)
	ast.Equal(true, r.HasError)
	ast.Equal(false, r.InBackup)

	// Test 3: primary has BackupHashOK false flag
	r, ok = getResult(Test3ID, res.Primary)
	ast.True(ok)
	ast.Equal(true, r.HasError)
	ast.Equal(false, r.BackupHashOK)
	ast.Equal(true, r.PrimaryHashOK)

	// Test 4: primary has PrimaryHashOK false flag
	r, ok = getResult(Test4ID, res.Primary)
	ast.True(ok)
	ast.Equal(true, r.HasError)
	ast.Equal(true, r.BackupHashOK)
	ast.Equal(false, r.PrimaryHashOK)

	// Test 5: primary has InCache false flag
	r, ok = getResult(Test5ID, res.Primary)
	ast.True(ok)
	ast.Equal(false, r.HasError)
	ast.Equal(false, r.InCache)

	for _, id := range blobs {
		if id == Test1ID {
			continue
		}
		r, ok = getResult(id, res.Primary)
		ast.True(ok)
		if id != Test2ID && id != Test3ID && id != Test4ID {
			ast.Equal(false, r.HasError)
		}
		if id != Test3ID {
			ast.Equal(true, r.BackupHashOK)
		}
		if id != Test4ID {
			ast.Equal(true, r.PrimaryHashOK)
		}

		if id != Test5ID {
			ast.Equal(true, r.InCache)
		}

		if id != Test2ID {
			ast.Equal(true, r.InBackup)
		}
	}
}
