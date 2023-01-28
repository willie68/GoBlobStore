package s3

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/internal/utils/readercomp"
	"github.com/willie68/GoBlobStore/pkg/model"
)

var (
	tntsrv TenantManager
)

const (
	tenant   = "mcs"
	pdffile  = "../../../testdata/pdf.pdf"
	testfile = "../../../testdata/pdf_dst.pdf"
)

// TODO all tests are skipped
func setup(t *testing.T) {
	tntsrv = TenantManager{
		Endpoint:  "http://127.0.0.1:9002",
		Bucket:    "testbucket",
		AccessKey: "D9Q2D6JQGW1MVCC98LQL",
		SecretKey: "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr",
		Insecure:  true, // only for self signed certificates
	}
	err := tntsrv.Init()
	assert.Nil(t, err)

	ok := tntsrv.HasTenant(tenant)
	if !ok {
		err = tntsrv.AddTenant(tenant)
		assert.Nil(t, err)
	}
}

func closeTest(t *testing.T) {
	_, err := tntsrv.RemoveTenant(tenant)
	assert.Nil(t, err)
}

func createSrv() (BlobStorage, error) {
	srv := BlobStorage{
		Endpoint:  "http://127.0.0.1:9002",
		Bucket:    "testbucket",
		AccessKey: "D9Q2D6JQGW1MVCC98LQL",
		SecretKey: "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr",
		Insecure:  true, // only for self signed certificates
		Tenant:    tenant,
	}
	err := srv.Init()
	return srv, err
}

func TestS3Init(t *testing.T) {
	t.SkipNow()
	setup(t)
	ast := assert.New(t)
	srv, err := createSrv()

	ast.Nil(err)
	ast.NotNil(srv)

	closeTest(t)
}

func TestCheckUnknownBlob(t *testing.T) {
	t.SkipNow()
	setup(t)
	ast := assert.New(t)
	srv, err := createSrv()

	ast.Nil(err)
	ast.NotNil(srv)

	ok, err := srv.HasBlob("murks")
	ast.Nil(err)
	ast.False(ok)

	closeTest(t)
}
func TestCheckEmptyTenant(t *testing.T) {
	t.SkipNow()
	setup(t)
	ast := assert.New(t)
	srv := BlobStorage{
		Endpoint:  "http://127.0.0.1:9002",
		Bucket:    "testbucket",
		AccessKey: "D9Q2D6JQGW1MVCC98LQL",
		SecretKey: "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr",
		Insecure:  true, // only for self signed certificates
	}
	err := srv.Init()

	ast.NotNil(err)

	closeTest(t)
}

func TestCRUDBlob(t *testing.T) {
	t.SkipNow()
	setup(t)
	ast := assert.New(t)
	srv, err := createSrv()

	ast.Nil(err)
	ast.NotNil(srv)
	fileInfo, err := os.Lstat(pdffile)
	ast.Nil(err)
	ast.NotNil(fileInfo)

	b := model.BlobDescription{
		ContentType:   "application/pdf",
		CreationDate:  time.Now().UnixMilli(),
		ContentLength: fileInfo.Size(),
		Filename:      fileInfo.Name(),
		TenantID:      tenant,
		Retention:     0,
		Properties:    make(map[string]any),
	}
	b.Properties["X-tenant"] = "MCS"

	r, err := os.Open(pdffile)
	ast.Nil(err)
	ast.NotNil(r)

	id, err := srv.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	err = r.Close()
	ast.Nil(err)

	fmt.Printf("blob id: %s", id)
	ok, err := srv.HasBlob(id)
	ast.Nil(err)
	ast.True(ok)

	d, err := srv.GetBlobDescription(id)
	ast.Nil(err)
	ast.NotNil(d)

	ast.Equal(b.ContentType, d.ContentType)
	ast.Equal(id, d.BlobID)
	ast.Equal(b.ContentLength, d.ContentLength)
	ast.Equal(b.Filename, d.Filename)

	w, err := os.Create(testfile)
	ast.Nil(err)
	err = srv.RetrieveBlob(id, w)
	ast.Nil(err)
	err = w.Close()
	ast.Nil(err)

	ok, err = readercomp.FilesEqual(pdffile, testfile)
	ast.Nil(err)
	ast.True(ok)

	b.Properties["X-tenant"] = "MCS_2"
	err = srv.UpdateBlobDescription(id, &b)
	ast.Nil(err)

	d, err = srv.GetBlobDescription(id)
	ast.Nil(err)
	ast.Equal(id, d.BlobID)
	ast.Equal("MCS_2", d.Properties["X-tenant"])

	blobs := make([]string, 0)
	err = srv.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(blobs))

	err = srv.DeleteBlob(id)
	ast.Nil(err)

	ok, err = srv.HasBlob(id)
	ast.Nil(err)
	ast.False(ok)

	closeTest(t)
}

func TestRetentionStorage(t *testing.T) {
	t.SkipNow()
	setup(t)

	ast := assert.New(t)
	srv, err := createSrv()
	ast.Nil(err)
	ast.NotNil(srv)

	blobID := "12345678"

	ret := model.RetentionEntry{
		Filename:      pdffile,
		TenantID:      tenant,
		BlobID:        blobID,
		CreationDate:  time.Now().UnixMilli(),
		Retention:     1,
		RetentionBase: 0,
	}

	err = srv.AddRetention(&ret)
	ast.Nil(err)

	rets := make([]model.RetentionEntry, 0)
	err = srv.GetAllRetentions(func(r model.RetentionEntry) bool {
		rets = append(rets, r)
		return true
	})
	ast.Nil(err)

	ast.Equal(1, len(rets))
	retDst := rets[0]

	ast.Equal(ret.BlobID, retDst.BlobID)
	ast.Equal(ret.CreationDate, retDst.CreationDate)
	ast.Equal(ret.Filename, retDst.Filename)
	ast.Equal(ret.Retention, retDst.Retention)
	ast.Equal(ret.RetentionBase, retDst.RetentionBase)

	retDst, err = srv.GetRetention(blobID)
	ast.Nil(err)

	ast.Equal(ret.BlobID, retDst.BlobID)
	ast.Equal(ret.CreationDate, retDst.CreationDate)
	ast.Equal(ret.Filename, retDst.Filename)
	ast.Equal(ret.Retention, retDst.Retention)
	ast.Equal(ret.RetentionBase, retDst.RetentionBase)

	err = srv.DeleteRetention(blobID)
	ast.Nil(err)

	closeTest(t)
}

func TestCRUDBlobWID(t *testing.T) {
	t.SkipNow()
	setup(t)
	ast := assert.New(t)
	uuid := utils.GenerateID()
	srv, err := createSrv()

	ast.Nil(err)
	ast.NotNil(srv)
	fileInfo, err := os.Lstat(pdffile)
	ast.Nil(err)
	ast.NotNil(fileInfo)

	b := model.BlobDescription{
		BlobID:        uuid,
		ContentType:   "application/pdf",
		CreationDate:  time.Now().UnixMilli(),
		ContentLength: fileInfo.Size(),
		Filename:      fileInfo.Name(),
		TenantID:      tenant,
		Retention:     0,
		Properties:    make(map[string]any),
	}
	b.Properties["X-tenant"] = "MCS"

	r, err := os.Open(pdffile)
	ast.Nil(err)
	ast.NotNil(r)

	id, err := srv.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	ast.Equal(id, uuid)
	err = r.Close()
	ast.Nil(err)

	fmt.Printf("blob id: %s", id)
	ok, err := srv.HasBlob(uuid)
	ast.Nil(err)
	ast.True(ok)

	d, err := srv.GetBlobDescription(uuid)
	ast.Nil(err)
	ast.NotNil(d)

	ast.Equal(b.ContentType, d.ContentType)
	ast.Equal(id, d.BlobID)
	ast.Equal(uuid, d.BlobID)
	ast.Equal(b.ContentLength, d.ContentLength)
	ast.Equal(b.Filename, d.Filename)

	w, err := os.Create(testfile)
	ast.Nil(err)
	err = srv.RetrieveBlob(uuid, w)
	ast.Nil(err)
	err = w.Close()
	ast.Nil(err)

	ok, err = readercomp.FilesEqual(pdffile, testfile)
	ast.Nil(err)
	ast.True(ok)

	b.Properties["X-tenant"] = "MCS_2"
	err = srv.UpdateBlobDescription(uuid, &b)
	ast.Nil(err)

	d, err = srv.GetBlobDescription(uuid)
	ast.Nil(err)
	ast.Equal(uuid, d.BlobID)
	ast.Equal("MCS_2", d.Properties["X-tenant"])

	blobs := make([]string, 0)
	err = srv.GetBlobs(func(id string) bool {
		blobs = append(blobs, id)
		return true
	})
	ast.Nil(err)
	ast.Equal(1, len(blobs))

	err = srv.DeleteBlob(uuid)
	ast.Nil(err)

	ok, err = srv.HasBlob(uuid)
	ast.Nil(err)
	ast.False(ok)

	closeTest(t)
}
