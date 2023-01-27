package business

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
)

// testing the tenant managment business part
var (
	tntPath    = filepath.Join(rootFilePrefix, "tntstg")
	tntbckPath = filepath.Join(rootFilePrefix, "tntbckstg")
	dao        MainTenant
)

func initTntTest(ast *assert.Assertions) {
	if _, err := os.Stat(tntPath); err == nil {
		err := os.RemoveAll(tntPath)
		ast.Nil(err)
	}

	if _, err := os.Stat(tntbckPath); err == nil {
		err := os.RemoveAll(tntbckPath)
		ast.Nil(err)
	}

	sfTnt := &simplefile.TenantManager{
		RootPath: tntPath,
	}
	bkTnt := &simplefile.TenantManager{
		RootPath: tntbckPath,
	}
	dao = MainTenant{
		TntDao: sfTnt,
		BckDao: bkTnt,
	}
	ast.NotNil(dao)
	err := dao.Init()
	ast.Nil(err)
}

func closeTntTest(ast *assert.Assertions) {
	err := dao.Close()
	ast.Nil(err)
}

func TestCRUDTenantOps(t *testing.T) {
	ast := assert.New(t)
	initTntTest(ast)

	ast.False(dao.HasTenant(tenant))

	err := dao.AddTenant(tenant)
	ast.Nil(err)

	time.Sleep(1 * time.Second)

	tenants := make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(1, len(tenants))

	ast.True(dao.HasTenant(tenant))
	ast.True(dao.GetSize(tenant) <= 0)

	_, err = dao.RemoveTenant(tenant)
	ast.Nil(err)
	ast.False(dao.HasTenant(tenant))

	tenants = make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(0, len(tenants))

	closeTntTest(ast)
}

func TestUnknownTenant(t *testing.T) {
	ast := assert.New(t)

	initTntTest(ast)

	_, err := dao.RemoveTenant(tenant)
	ast.NotNil(err)

	time.Sleep(1 * time.Second)

	tenants := make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(0, len(tenants))

	ast.False(dao.HasTenant(tenant))
	ast.True(dao.GetSize(tenant) <= 0)

	closeTntTest(ast)
}

func TestNilTntDao(t *testing.T) {
	ast := assert.New(t)

	dao := MainTenant{}

	err := dao.Init()
	ast.NotNil(err)
}
