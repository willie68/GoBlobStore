package simplefile

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/utils/slicesutils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	zipfile  = "../../../testdata/mcs.zip"
	rootpath = "../../../testdata/blobstorage"
	tenant   = "MCS"
)

func initTest(t *testing.T) {
	ast := assert.New(t)

	if _, err := os.Stat(rootpath); err == nil {
		err := os.RemoveAll(rootpath)
		ast.Nil(err)
	}
	// getting the zip file and extracting it into the file system
	os.MkdirAll(rootpath, os.ModePerm)

	// getting the zip file and extracting it into the file system
	archive, err := zip.OpenReader(zipfile)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(rootpath, f.Name)
		fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(rootpath)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			fmt.Println("creating directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
}

func getStoreageDao(t *testing.T) SimpleFileBlobStorageDao {
	dao := SimpleFileBlobStorageDao{
		RootPath: rootpath,
		Tenant:   tenant,
	}
	err := dao.Init()
	if err != nil {
		t.Fatal(err)
	}
	return dao
}
func TestTenanthandling(t *testing.T) {
	// Tenant nil
	dao := SimpleFileBlobStorageDao{
		RootPath: rootpath,
	}
	err := dao.Init()
	assert.NotNil(t, err)

	// Tenant empty
	dao = SimpleFileBlobStorageDao{
		RootPath: rootpath,
		Tenant:   "",
	}
	err = dao.Init()
	assert.NotNil(t, err)
}

func TestNotFound(t *testing.T) {
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
	dao := getStoreageDao(t)

	srcPath, _ := filepath.Abs(filepath.Join(rootpath, tenant))
	assert.Equal(t, srcPath, dao.filepath)

	blobs := make([]string, 0)
	err := dao.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 7, len(blobs))
	assert.True(t, slicesutils.Contains(blobs, "004b4987-42fb-43e4-8e13-d6994ce0e6f1"))
	assert.True(t, slicesutils.Contains(blobs, "0000fc02-050a-418a-a701-efd814aa6b36"))

	for _, blobid := range blobs {
		fmt.Println(blobid)
	}
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

	b := model.BlobDescription{
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

func TestRetentionStorage(t *testing.T) {
	ast := assert.New(t)

	dao := getStoreageDao(t)
	ast.NotNil(dao)

	b := model.BlobDescription{
		StoreID:       tenant,
		TenantID:      tenant,
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		Filename:      "test.txt",
		LastAccess:    int(time.Now().UnixNano() / 1000000),
		Retention:     1,
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

	ret := model.RetentionEntry{
		Filename:      "test.txt",
		TenantID:      tenant,
		BlobID:        id,
		CreationDate:  b.CreationDate,
		Retention:     1,
		RetentionBase: 0,
	}

	err = dao.AddRetention(&ret)
	ast.Nil(err)

	rets := make([]model.RetentionEntry, 0)
	dao.GetAllRetentions(func(r model.RetentionEntry) bool {
		rets = append(rets, r)
		return true
	})

	ast.Equal(8, len(rets))
	retDst, err := dao.GetRetention(id)
	ast.Nil(err)

	ast.Equal(ret.BlobID, retDst.BlobID)
	ast.Equal(ret.CreationDate, retDst.CreationDate)
	ast.Equal(ret.Filename, retDst.Filename)
	ast.Equal(ret.Retention, retDst.Retention)
	ast.Equal(ret.RetentionBase, retDst.RetentionBase)

	err = dao.DeleteRetention(id)
	ast.Nil(err)

	err = dao.DeleteBlob(id)
	ast.Nil(err)
}
