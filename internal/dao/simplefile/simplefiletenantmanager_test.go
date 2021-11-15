package simplefile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimplefileTenantManager(t *testing.T) {

	dao := SimpleFileTenantManager{
		RootPath: rootpath,
	}
	err := dao.Init()
	assert.Nil(t, err)

	_ = dao.RemoveTenant(tenant)

	tenants := make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	assert.Nil(t, err)
	assert.Equal(t, 1, len(tenants))

	ok := dao.HasTenant(tenant)
	assert.False(t, ok)

	err = dao.AddTenant(tenant)
	assert.Nil(t, err)

	tenants = make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	assert.Nil(t, err)
	assert.Equal(t, 2, len(tenants))

	ok = dao.HasTenant(tenant)
	assert.True(t, ok)

	size := dao.GetSize(tenant)
	assert.Equal(t, int64(0), size)

	err = dao.RemoveTenant(tenant)
	assert.Nil(t, err)

	ok = dao.HasTenant(tenant)
	assert.False(t, ok)

	tenants = make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	assert.Nil(t, err)
	assert.Equal(t, 1, len(tenants))
}
