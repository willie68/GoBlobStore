package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestS3TenantManager(t *testing.T) {
	ast := assert.New(t)
	dao := S3TenantManager{
		Endpoint:  "http://127.0.0.1:9002",
		Bucket:    "testbucket",
		AccessKey: "D9Q2D6JQGW1MVCC98LQL",
		SecretKey: "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr",
		Insecure:  true, // only for self signed certificates
	}
	err := dao.Init()
	ast.Nil(err)
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

	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(1, len(tenants))

	ok = dao.HasTenant(tenant)
	ast.True(ok)

	err = dao.AddTenant(tenant)
	ast.NotNil(t, err)

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
