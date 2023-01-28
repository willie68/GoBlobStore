package business

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/services/simplefile"
)

// testing the tenant managment business part
var (
	tntPath    = filepath.Join(rootFilePrefix, "tntstg")
	tntbckPath = filepath.Join(rootFilePrefix, "tntbckstg")
	tnt        MainTenant
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
	tnt = MainTenant{
		TntSrv: sfTnt,
		BckSrv: bkTnt,
	}
	ast.NotNil(tnt)
	err := tnt.Init()
	ast.Nil(err)
}

func closeTntTest(ast *assert.Assertions) {
	err := tnt.Close()
	ast.Nil(err)
}

func TestCRUDTenantOps(t *testing.T) {
	ast := assert.New(t)
	initTntTest(ast)

	ast.False(tnt.HasTenant(tenant))

	err := tnt.AddTenant(tenant)
	ast.Nil(err)

	time.Sleep(1 * time.Second)

	tenants := make([]string, 0)
	err = tnt.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(1, len(tenants))

	ast.True(tnt.HasTenant(tenant))
	ast.True(tnt.GetSize(tenant) <= 0)

	_, err = tnt.RemoveTenant(tenant)
	ast.Nil(err)
	ast.False(tnt.HasTenant(tenant))

	tenants = make([]string, 0)
	err = tnt.GetTenants(func(t string) bool {
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

	_, err := tnt.RemoveTenant(tenant)
	ast.NotNil(err)

	time.Sleep(1 * time.Second)

	tenants := make([]string, 0)
	err = tnt.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(0, len(tenants))

	ast.False(tnt.HasTenant(tenant))
	ast.True(tnt.GetSize(tenant) <= 0)

	closeTntTest(ast)
}

func TestNilTntSrv(t *testing.T) {
	ast := assert.New(t)

	tnt := MainTenant{}

	err := tnt.Init()
	ast.NotNil(err)
}
