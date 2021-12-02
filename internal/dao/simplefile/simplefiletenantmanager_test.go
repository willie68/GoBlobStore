package simplefile

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func initTenantTest(t *testing.T) {
	ast := assert.New(t)

	if _, err := os.Stat(rootpath); err == nil {
		err := os.RemoveAll(rootpath)
		ast.Nil(err)
	}
}
func TestAutoPathCreation(t *testing.T) {
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
	ast.Nil(err)

	_, err = os.Stat(rootpath)

	ast.Nil(err)
}

func TestSimplefileTenantManager(t *testing.T) {
	initTest(t)

	ast := assert.New(t)

	dao := SimpleFileTenantManager{
		RootPath: rootpath,
	}
	err := dao.Init()
	ast.Nil(err)

	_ = dao.RemoveTenant(tenant)

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

	err = dao.RemoveTenant(tenant)
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
