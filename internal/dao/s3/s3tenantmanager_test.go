package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestS3TenantManager(t *testing.T) {

	dao := S3TenantManager{
		Endpoint:  "http://127.0.0.1:9002",
		Bucket:    "testbucket",
		AccessKey: "D9Q2D6JQGW1MVCC98LQL",
		SecretKey: "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr",
		Insecure:  true, // only for self signed certificates
	}
	err := dao.Init()
	assert.Nil(t, err)
	tenants := make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	assert.Nil(t, err)
	assert.Equal(t, 0, len(tenants))

	ok := dao.HasTenant("easy")
	assert.False(t, ok)

	err = dao.AddTenant("easy")
	assert.Nil(t, err)

	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	assert.Nil(t, err)
	assert.Equal(t, 1, len(tenants))

	ok = dao.HasTenant("easy")
	assert.True(t, ok)

	size := dao.GetSize("easy")
	assert.Equal(t, int64(0), size)

	err = dao.RemoveTenant("easy")
	assert.Nil(t, err)

	ok = dao.HasTenant("easy")
	assert.False(t, ok)

	tenants = make([]string, 0)
	err = dao.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	assert.Nil(t, err)
	assert.Equal(t, 0, len(tenants))
}
