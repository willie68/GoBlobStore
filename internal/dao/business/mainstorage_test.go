package business

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
	rootFilePrefix = "R:/"
	tenant         = "test"
)

var main interfaces.BlobStorageDao

func initTest(t *testing.T) {
	stgDao := &simplefile.SimpleFileBlobStorageDao{
		RootPath: rootFilePrefix + "blbstg",
		Tenant:   tenant,
	}
	stgDao.Init()
	cchDao := &fastcache.FastCache{
		RootPath:   rootFilePrefix + "blbcch",
		MaxCount:   100000,
		MaxRamSize: 1 * 1024 * 1024 * 1024,
	}
	cchDao.Init()
	main = &MainStorageDao{
		StgDao: stgDao,
		CchDao: cchDao,
	}

	main.Init()
}

func clear(t *testing.T) {
	// getting the zip file and extracting it into the file system
	err := removeContents(rootFilePrefix)
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
		os.RemoveAll(filepath.Join(dir, name))
	}
	return nil
}

func createBlobDescription(id string) model.BlobDescription {
	uuid := utils.GenerateID()

	b := model.BlobDescription{
		BlobID:        uuid,
		StoreID:       tenant,
		TenantID:      tenant,
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		Filename:      fmt.Sprintf("test_%s.txt", id),
		LastAccess:    int(time.Now().UnixNano() / 1000000),
		Retention:     180000,
		Properties:    make(map[string]interface{}),
	}
	b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
	b.Properties["X-retention"] = []int{123456}
	b.Properties["X-tenant"] = tenant
	b.Properties["X-externalid"] = id
	return b
}

func TestManyFiles(t *testing.T) {
	clear(t)
	initTest(t)
	ast := assert.New(t)
	ast.NotNil(main)

	ids := make([]model.BlobDescription, 0)
	for i := 0; i < 100000; i++ {
		if i%100 == 0 {
			if i%10000 == 0 {
				fmt.Println()
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

	for _, b := range ids {
		fmt.Printf("%s: id: %s\r\n", b.Properties["X-externalid"], b.BlobID)
	}
}

func createBlob(ast *assert.Assertions, is string) (model.BlobDescription, error) {
	b := createBlobDescription(is)
	payload := fmt.Sprintf("this is a blob content of %s", is)
	b.Properties["payload"] = payload
	b.ContentLength = int64(len(payload))
	r := strings.NewReader(payload)
	id, err := main.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	ast.Equal(id, b.BlobID)

	time.Sleep(10 * time.Millisecond)

	info, err := main.GetBlobDescription(id)
	ast.Nil(err, fmt.Sprintf("is: %s", is))
	ast.Equal(id, info.BlobID)

	var buf bytes.Buffer

	err = main.RetrieveBlob(id, &buf)
	ast.Nil(err)

	ast.Equal(payload, buf.String())
	return b, err
}
