package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

// TODO all tests are skipped
func TestS3TenantManager(t *testing.T) {
	if skip {
		t.SkipNow()
	}
	ast := assert.New(t)
	srv := TenantManager{
		Endpoint:  s3_endpoint,
		Bucket:    s3_bucket,
		AccessKey: s3_accessKey,
		SecretKey: s3_secretKey,
		Insecure:  s3_insecure, // only for self signed certificates
	}
	err := srv.Init()
	ast.Nil(err)
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

	err = srv.GetTenants(func(t string) bool {
		tenants = append(tenants, t)
		return true
	})

	ast.Nil(err)
	ast.Equal(1, len(tenants))

	ok = srv.HasTenant(tenant)
	ast.True(ok)

	err = srv.AddTenant(tenant)
	ast.NotNil(t, err)

	size := srv.GetSize(tenant)
	ast.Equal(int64(0), size)

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
	if skip {
		t.SkipNow()
	}
	ast := assert.New(t)
	srv := TenantManager{
		Endpoint:  s3_endpoint,
		Bucket:    s3_bucket,
		AccessKey: s3_accessKey,
		SecretKey: s3_secretKey,
		Insecure:  s3_insecure, // only for self signed certificates
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
	_, err = srv.RemoveTenant("MCS")
	ast.Nil(err)
}
