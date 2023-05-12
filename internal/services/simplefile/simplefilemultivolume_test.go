package simplefile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nsf/jsondiff"
	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	sfmvRootPath      = "../../../testdata/mv/"
	sfmvSimpleContent = "this is a blob content"
)

var vols = []string{"mvn01", "mvn02", "mvn03"}

func initSFMVTest(t *testing.T) {
	ast := assert.New(t)
	clear(t)

	for _, v := range vols {
		err := os.MkdirAll(filepath.Join(sfmvRootPath, v), os.ModePerm)
		ast.Nil(err)
	}
}

func clear(t *testing.T) {
	// getting the zip file and extracting it into the file system
	if _, err := os.Stat(sfmvRootPath); err == nil {
		err := os.RemoveAll(sfmvRootPath)
		assert.Nil(t, err)
	}
	err := os.MkdirAll(sfmvRootPath, os.ModePerm)
	assert.Nil(t, err)
}

func getSFMVStoreageSrv(t *testing.T) MultiVolumeStorage {
	srv := MultiVolumeStorage{
		RootPath: sfmvRootPath,
		Tenant:   tenant,
	}
	err := srv.Init()
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func TestSimpleFileMultiVolumeSrvNoTenant(t *testing.T) {
	ast := assert.New(t)
	srv := MultiVolumeStorage{
		RootPath: sfmvRootPath,
	}

	err := srv.Init()
	ast.NotNil(err)
}

func TestSimpleFileMultiVolumeSrv_Init(t *testing.T) {
	ast := assert.New(t)
	initSFMVTest(t)

	srv := getSFMVStoreageSrv(t)
	ast.NotNil(srv)

	for _, v := range vols {
		vi := srv.volMan.Info(v)
		ast.NotNil(vi)
		_, err := os.Stat(filepath.Join(vi.Path, ".volumeinfo"))
		ast.Nil(err)
	}

	err := srv.GetLastError()
	ast.Equal(ErrNotImplemented, err)
}

func TestSFMVSrvGeneral(t *testing.T) {
	ast := assert.New(t)
	initSFMVTest(t)

	srv := getSFMVStoreageSrv(t)
	ast.NotNil(srv)
	ast.Equal(ErrNotImplemented, srv.SearchBlobs("", func(id string) bool {
		return true
	}), "search should not be implemented")

	err := srv.Close()
	ast.Nil(err, "error closing srv")
}

func TestSFMVSrvTenant(t *testing.T) {
	ast := assert.New(t)
	initSFMVTest(t)

	srv := getSFMVStoreageSrv(t)
	ast.NotNil(srv)
	ast.Equal(tenant, srv.GetTenant())
}

func TestSFMVSrvStoreOneBlobCRUD(t *testing.T) {
	ast := assert.New(t)
	initSFMVTest(t)

	srv := getSFMVStoreageSrv(t)
	ast.NotNil(srv, "can't init srv")

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

	r := strings.NewReader(sfmvSimpleContent)
	id, err := srv.StoreBlob(&b, r)
	if err != nil {
		t.Fatalf("fatal error in storage: %v", err)
	}
	assert.NotNil(t, id, "no id given")

	ok, err := srv.HasBlob(id)
	ast.Nil(err, "HasBlob throws error")
	ast.True(ok, "blob id '%s' is unknow", id)

	bd, err := srv.GetBlobDescription(id)
	ast.Nil(err, "GetBlobDescription throws error")
	ast.NotNil(bd, "blob description for id '%s' is unknow", id)

	js1, err := b.MarshalJSON()
	ast.Nil(err, "json marshall throws error blob description src")

	js2, err := bd.MarshalJSON()
	ast.Nil(err, "json marshall throws error blob description dest")
	lopts := jsondiff.DefaultJSONOptions()
	diffEnum, diff := jsondiff.Compare(js1, js2, &lopts)

	fmt.Printf("diff: %v: %v", diffEnum, diff)

	var buf bytes.Buffer
	err = srv.RetrieveBlob(id, &buf)
	ast.Nil(err, "RetriveBlob throws error")
	ast.Equal(sfmvSimpleContent, buf.String(), "content not equal")

	err = srv.DeleteBlob(id)
	ast.Nil(err, "DeleteBlob throws error")
}

func TestSFMVSrvStoreOneBlobExtend(t *testing.T) {
	ast := assert.New(t)
	initSFMVTest(t)

	srv := getSFMVStoreageSrv(t)
	ast.NotNil(srv, "can't init srv")

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

	r := strings.NewReader(sfmvSimpleContent)
	id, err := srv.StoreBlob(&b, r)
	if err != nil {
		t.Fatalf("fatal error in storage: %v", err)
	}
	assert.NotNil(t, id, "no id given")

	ok, err := srv.HasBlob(id)
	ast.Nil(err, "HasBlob throws error")
	ast.True(ok, "blob id '%s' is unknow", id)

	time.Sleep(1 * time.Second)

	ci, err := srv.CheckBlob(id)
	if err != nil {
		t.Fatalf("CheckBlob throws error: %v", err)
	}
	ast.NotNil(ci, "checkblob is nil")
	if ci != nil {
		ast.True(ci.Healthy, "checkblob is not healthy")
	}

	err = srv.DeleteBlob(id)
	ast.Nil(err, "DeleteBlob throws error")
}

func TestSFMVSrvRetentionCRUD(t *testing.T) {
	ast := assert.New(t)
	initSFMVTest(t)

	srv := getSFMVStoreageSrv(t)
	ast.NotNil(srv, "can't init srv")
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

	r := strings.NewReader(sfmvSimpleContent)
	id, err := srv.StoreBlob(&b, r)
	if err != nil {
		t.Fatalf("fatal error in storage: %v", err)
	}
	assert.NotNil(t, id, "no id given")

	rt := model.RetentionEntry{
		TenantID:      tenant,
		BlobID:        id,
		CreationDate:  time.Now().UnixMilli(),
		Retention:     int64(12345678),
		RetentionBase: 0,
		Filename:      "filename",
	}

	err = srv.AddRetention(&rt)
	ast.Nil(err, "AddRetention throws error")

	rt2, err := srv.GetRetention(rt.BlobID)
	ast.Nil(err, "GetRetention throws error")
	ast.Equal(rt.BlobID, rt2.BlobID, "blob id is not equal")
	ast.Equal(rt.Retention, rt.Retention, "retention is not equal")

	time.Sleep(1 * time.Second)

	err = srv.ResetRetention(rt.BlobID)
	ast.Nil(err, "ResetRetention throws error")

	rt2, err = srv.GetRetention(rt.BlobID)
	ast.Nil(err, "GetRetention throws error")
	ast.Equal(rt.BlobID, rt2.BlobID, "blob id is not equal")
	ast.Equal(rt.Retention, rt.Retention, "retention is not equal")
	ast.NotEqual(0, rt.RetentionBase, "RetentionBase not set")

	count := 0
	err = srv.GetAllRetentions(func(r model.RetentionEntry) bool {
		ast.Equal(rt.BlobID, r.BlobID, "GetAllRetentions failed")
		count++
		return true
	})
	ast.Nil(err, "GetAllRetentions throws error")
	ast.Equal(1, count, "GetAllRetentions wrong count")

	err = srv.DeleteRetention(rt.BlobID)
	ast.Nil(err, "DeleteRetention throws error")

	err = srv.DeleteBlob(id)
	ast.Nil(err, "DeleteBlob throws error")
}

func TestSFMVSrvStoreMultiBlobs(t *testing.T) {
	ast := assert.New(t)
	initSFMVTest(t)

	srv := getSFMVStoreageSrv(t)
	ast.NotNil(srv, "can't init srv")
	ids := make([]string, 0)

	for i := 1; i <= 1000; i++ {
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
		b.Properties["X-count"] = i
		b.Properties["X-user"] = []string{"Hallo", "Hallo2"}
		b.Properties["X-retention"] = []int{123456}
		b.Properties["X-tenant"] = "MCS"

		r := strings.NewReader(sfmvSimpleContent)
		id, err := srv.StoreBlob(&b, r)
		if err != nil {
			t.Fatalf("fatal error in storage: %v", err)
		}
		assert.NotNil(t, id, "no id given")

		ok, err := srv.HasBlob(id)
		ast.Nil(err, "HasBlob throws error")
		ast.True(ok, "blob id '%s' is unknow", id)

		var buf bytes.Buffer
		err = srv.RetrieveBlob(id, &buf)
		ast.Nil(err, "RetriveBlob throws error")
		ast.Equal(sfmvSimpleContent, buf.String(), "content not equal")

		ids = append(ids, id)
	}

	for _, id := range ids {
		err := srv.DeleteBlob(id)
		ast.Nil(err, "DeleteBlob throws error")
	}
}
