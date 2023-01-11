package simplefile

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
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
	dao := SimpleFileTenantManager{
		RootPath: rootpath,
	}
	err := dao.Init()
	ast.Nil(err, "error: %v", err)

	_, err = os.Stat(rootpath)

	ast.Nil(err)
}

func TestSimplefileTenantManager(t *testing.T) {
	initTenantTest(t)

	ast := assert.New(t)

	dao := SimpleFileTenantManager{
		RootPath: rootpath,
	}
	err := dao.Init()
	ast.Nil(err)

	time.Sleep(1 * time.Second)

	tenants := make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(0, len(tenants))

	ok := dao.HasTenant(tenant)
	ast.False(ok)

	err = dao.AddTenant(tenant)
	ast.Nil(err)

	tenants = make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(1, len(tenants))

	ok = dao.HasTenant(tenant)
	ast.True(ok)

	size := dao.GetSize(tenant)
	ast.Equal(int64(0), size)

	_, err = dao.RemoveTenant(tenant)
	ast.Nil(err)

	ok = dao.HasTenant(tenant)
	ast.False(ok)

	tenants = make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(0, len(tenants))
}

func TestSimplefileTenantManagerConfig(t *testing.T) {
	initTenantTest(t)

	ast := assert.New(t)

	dao := SimpleFileTenantManager{
		RootPath: rootpath,
	}
	err := dao.Init()
	ast.Nil(err)

	dao.AddTenant("MCS")
	ast.True(dao.HasTenant("MCS"))

	cfn, err := dao.GetConfig("MCS")
	ast.Nil(err)
	ast.Nil(cfn)

	stg := config.Storage{
		Storageclass: "S3",
		Properties:   make(map[string]interface{}),
	}
	stg.Properties["accessKey"] = "accessKey"
	stg.Properties["secretKey"] = "secretKey"

	cfn = &interfaces.TenantConfig{
		Backup: stg,
	}
	err = dao.SetConfig("MCS", *cfn)
	ast.Nil(err)

	cfn2, err := dao.GetConfig("MCS")
	ast.Nil(err)
	ast.NotNil(cfn2)

	ast.Equal(cfn.Backup.Storageclass, cfn2.Backup.Storageclass)
	ast.Equal(cfn.Backup.Properties["accessKey"], cfn2.Backup.Properties["accessKey"])
	ast.Equal(cfn.Backup.Properties["secretKey"], cfn2.Backup.Properties["secretKey"])
}
