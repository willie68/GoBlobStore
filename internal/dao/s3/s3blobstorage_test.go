package s3

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/pkg/model"
)

var (
	tntDao S3TenantManager
)

const (
	tenant  = "easy"
	pdffile = "../../../testdata/pdf.pdf"
)

func setup(t *testing.T) {
	tntDao = S3TenantManager{
		Endpoint:  "http://127.0.0.1:9002",
		Bucket:    "testbucket",
		AccessKey: "D9Q2D6JQGW1MVCC98LQL",
		SecretKey: "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr",
		Insecure:  true, // only for self signed certificates
	}
	err := tntDao.Init()
	assert.Nil(t, err)

	ok := tntDao.HasTenant(tenant)
	if !ok {
		tntDao.AddTenant(tenant)
	}
}

func close(t *testing.T) {
	tntDao.RemoveTenant(tenant)
}

func createDao() (S3BlobStorage, error) {
	dao := S3BlobStorage{
		Endpoint:  "http://127.0.0.1:9002",
		Bucket:    "testbucket",
		AccessKey: "D9Q2D6JQGW1MVCC98LQL",
		SecretKey: "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr",
		Insecure:  true, // only for self signed certificates
		Tenant:    tenant,
	}
	err := dao.Init()
	return dao, err
}

func TestS3Init(t *testing.T) {
	setup(t)
	ast := assert.New(t)
	dao, err := createDao()

	ast.Nil(err)
	ast.NotNil(dao)

	close(t)
}

func TestCheckUnknownBlob(t *testing.T) {
	setup(t)
	ast := assert.New(t)
	dao, err := createDao()

	ast.Nil(err)
	ast.NotNil(dao)

	ok, err := dao.HasBlob("murks")
	ast.Nil(err)
	ast.False(ok)

	close(t)
}
func TestCheckEmptyTenant(t *testing.T) {
	setup(t)
	ast := assert.New(t)
	dao := S3BlobStorage{
		Endpoint:  "http://127.0.0.1:9002",
		Bucket:    "testbucket",
		AccessKey: "D9Q2D6JQGW1MVCC98LQL",
		SecretKey: "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr",
		Insecure:  true, // only for self signed certificates
	}
	err := dao.Init()

	ast.NotNil(err)

	close(t)
}

func TestCRUDBlob(t *testing.T) {
	setup(t)
	ast := assert.New(t)
	dao, err := createDao()

	ast.Nil(err)
	ast.NotNil(dao)
	fileInfo, err := os.Lstat(pdffile)
	ast.Nil(err)
	ast.NotNil(fileInfo)

	b := model.BlobDescription{
		ContentType:   "application/pdf",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		ContentLength: fileInfo.Size(),
		Filename:      fileInfo.Name(),
		TenantID:      tenant,
		Retention:     0,
	}

	r, err := os.Open(pdffile)
	ast.Nil(err)
	ast.NotNil(r)

	id, err := dao.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	r.Close()

	fmt.Printf("blob id: %s", id)
	ok, err := dao.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)

	d, err := dao.GetBlobDescription(id)
	ast.Nil(err)
	ast.NotNil(d)

	ast.Equal(b.ContentType, d.ContentType)
	ast.Equal(id, d.BlobID)
	ast.Equal(b.ContentLength, d.ContentLength)
	ast.Equal(b.Filename, d.Filename)

	dao.RetrieveBlob(id, w)

	err = dao.DeleteBlob(id)
	ast.Nil(err)

	ok, err = dao.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	close(t)
}
