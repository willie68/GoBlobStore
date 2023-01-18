package simplefile

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/internal/utils/slicesutils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

func getSFStoreageDao(t *testing.T) BlobStorage {
	dao := BlobStorage{
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
	dao := BlobStorage{
		RootPath: rootpath,
	}
	err := dao.Init()
	assert.NotNil(t, err)

	// Tenant empty
	dao = BlobStorage{
		RootPath: rootpath,
		Tenant:   "",
	}
	err = dao.Init()
	assert.NotNil(t, err)
}

func TestNotFound(t *testing.T) {
	dao := getSFStoreageDao(t)
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
	dao := getSFStoreageDao(t)

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
	dao := getSFStoreageDao(t)
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

func TestCRUD(t *testing.T) {
	ast := assert.New(t)
	dao := getSFStoreageDao(t)

	b := model.BlobDescription{
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

func TestRetentionStorage(t *testing.T) {
	ast := assert.New(t)

	dao := getSFStoreageDao(t)
	ast.NotNil(dao)

	b := model.BlobDescription{
		StoreID:       tenant,
		TenantID:      tenant,
		ContentLength: 22,
		ContentType:   "text/plain",
		CreationDate:  time.Now().UnixMilli(),
		Filename:      "test.txt",
		LastAccess:    time.Now().UnixMilli(),
		Retention:     1,
		Properties:    make(map[string]any),
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

func TestBlobCheck(t *testing.T) {
	initTest(t)

	ast := assert.New(t)

	dao := getSFStoreageDao(t)
	ast.NotNil(dao)

	blobs := make([]string, 0)
	err := dao.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	ast.Nil(err)

	res, err := dao.CheckBlob("001a7543-cb7a-4c2c-9c23-1bb6b248034c")
	ast.Nil(err)
	ast.False(res.Healthy, "id: %s: %s", "001a7543-cb7a-4c2c-9c23-1bb6b248034c", res.Message)

	res, err = dao.CheckBlob("0000fc02-050a-418a-a701-efd814aa6b36")
	ast.Nil(err)
	ast.True(res.Healthy, "id: %s: %s", "0000fc02-050a-418a-a701-efd814aa6b36", res.Message)

	res, err = dao.CheckBlob("004b4987-42fb-43e4-8e13-d6994ce0e6f1")
	ast.Nil(err)
	ast.True(res.Healthy, "id: %s: %s", "004b4987-42fb-43e4-8e13-d6994ce0e6f1", res.Message)
}

func TestCRUDWithGivenID(t *testing.T) {
	ast := assert.New(t)
	uuid := utils.GenerateID()
	dao := getSFStoreageDao(t)

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
	ast.Equal(id, uuid)

	info, err := dao.GetBlobDescription(uuid)
	ast.Nil(err)
	ast.Equal(id, info.BlobID)
	ast.Equal(id, uuid)

	var buf bytes.Buffer

	err = dao.RetrieveBlob(uuid, &buf)
	ast.Nil(err)
	ast.Equal("this is a blob content", buf.String())

	b.Properties["X-tenant"] = "MCS_2"
	err = dao.UpdateBlobDescription(id, &b)
	ast.Nil(err)

	info, err = dao.GetBlobDescription(uuid)
	ast.Nil(err)
	ast.Equal(id, info.BlobID)
	ast.Equal("MCS_2", info.Properties["X-tenant"])

	err = dao.DeleteBlob(uuid)
	ast.Nil(err)

	dao.Close()
}
