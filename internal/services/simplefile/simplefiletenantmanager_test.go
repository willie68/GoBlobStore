package simplefile

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

func initTenantTest(t *testing.T) {
	ast := assert.New(t)

	if _, err := os.Stat(rootpath); errors.Is(err, os.ErrNotExist) {
		return
	}
	err := os.RemoveAll(rootpath)
	ast.Nil(err)
}
func TestAutoPathCreation(t *testing.T) {
	time.Sleep(1 * time.Second)
	initTenantTest(t)
	ast := assert.New(t)

	if _, err := os.Stat(rootpath); err == nil {
		err := os.RemoveAll(rootpath)
		ast.Nil(err)
	}
	srv := TenantManager{
		RootPath: rootpath,
	}
	err := srv.Init()
	ast.Nil(err, "error: %v", err)

	_, err = os.Stat(rootpath)

	ast.Nil(err)
}

func TestSimplefileTenantManager(t *testing.T) {
	initTenantTest(t)

	ast := assert.New(t)

	srv := TenantManager{
		RootPath: rootpath,
	}
	err := srv.Init()
	ast.Nil(err)

	time.Sleep(1 * time.Second)

	tenants := make([]string, 0)
	err = srv.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(0, len(tenants))

	ok := srv.HasTenant(tenant)
	ast.False(ok)

	err = srv.AddTenant(tenant)
	ast.Nil(err)

	tenants = make([]string, 0)
	err = srv.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(1, len(tenants))

	ok = srv.HasTenant(tenant)
	ast.True(ok)

	size := srv.GetSize(tenant)
	ast.True(size <= 0)

	_, err = srv.RemoveTenant(tenant)
	ast.Nil(err)

	ok = srv.HasTenant(tenant)
	ast.False(ok)

	tenants = make([]string, 0)
	err = srv.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(0, len(tenants))
}

func TestSimplefileTenantManagerConfig(t *testing.T) {
	initTenantTest(t)

	ast := assert.New(t)

	srv := TenantManager{
		RootPath: rootpath,
	}
	err := srv.Init()
	ast.Nil(err)

	err = srv.AddTenant("MCS")
	ast.Nil(err)
	ast.True(srv.HasTenant("MCS"))

	cfn, err := srv.GetConfig("MCS")
	ast.Nil(err)
	ast.Nil(cfn)

	stg := config.Storage{
		Storageclass: "S3",
		Properties:   make(map[string]any),
	}
	stg.Properties["accessKey"] = "accessKey"
	stg.Properties["secretKey"] = "secretKey"

	cfn = &interfaces.TenantConfig{
		Backup: stg,
	}
	err = srv.SetConfig("MCS", *cfn)
	ast.Nil(err)

	cfn2, err := srv.GetConfig("MCS")
	ast.Nil(err)
	ast.NotNil(cfn2)

	ast.Equal(cfn.Backup.Storageclass, cfn2.Backup.Storageclass)
	ast.Equal(cfn.Backup.Properties["accessKey"], cfn2.Backup.Properties["accessKey"])
	ast.Equal(cfn.Backup.Properties["secretKey"], cfn2.Backup.Properties["secretKey"])
}

func TestSize(t *testing.T) {
	initTenantTest(t)

	ast := assert.New(t)

	tntsrv := TenantManager{
		RootPath: rootpath,
	}
	stgsrv := BlobStorage{
		RootPath: rootpath,
		Tenant:   tenant,
	}

	err := stgsrv.Init()
	ast.Nil(err)
	err = tntsrv.Init()
	ast.Nil(err)

	time.Sleep(1 * time.Second)

	tenants := make([]string, 0)
	err = tntsrv.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(0, len(tenants))

	err = tntsrv.AddTenant(tenant)
	ast.Nil(err)
	ast.True(tntsrv.HasTenant(tenant))

	size := tntsrv.GetSize(tenant)
	ast.True(size <= 0)

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
	id, err := stgsrv.StoreBlob(&b, r)
	ast.Nil(err)
	ast.NotNil(id)
	ast.Equal(id, b.BlobID)
	time.Sleep(90 * time.Second)

	size = tntsrv.GetSize(tenant)
	ast.True(size > 0)
}
