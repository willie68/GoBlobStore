package dao

import (
	"errors"
	"fmt"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/s3"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
)

//GetTenantDao returning the tenant for administration tenants
func GetTenantDao() (interfaces.TenantDao, error) {
	if tenantDao == nil {
		return nil, errors.New("no tenantdao present")
	}
	return tenantDao, nil
}

// createTenantDao creating a new tenant dao depending on the configuration
func createTenantDao(stgClass string) (interfaces.TenantDao, error) {
	switch stgClass {
	case STGCLASS_SIMPLE_FILE:
		rootpath, err := getConfigValueAsString("rootpath")
		if err != nil {
			return nil, err
		}
		dao := &simplefile.SimpleFileTenantManager{
			RootPath: rootpath,
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
		return dao, nil
	case STGCLASS_S3:
		dao, err := getS3TenantManager()
		if err != nil {
			return nil, err
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
		return dao, nil
	}
	return nil, fmt.Errorf("no tenantmanager class implementation for \"%s\" found", cnfg.Storageclass)
}

func getS3TenantManager() (*s3.S3TenantManager, error) {
	endpoint, err := getConfigValueAsString("endpoint")
	if err != nil {
		return nil, err
	}
	insecure, err := getConfigValueAsBool("insecure")
	if err != nil {
		return nil, err
	}
	bucket, err := getConfigValueAsString("bucket")
	if err != nil {
		return nil, err
	}
	accessKey, err := getConfigValueAsString("accessKey")
	if err != nil {
		return nil, err
	}
	secretKey, err := getConfigValueAsString("secretKey")
	if err != nil {
		return nil, err
	}
	password, err := getConfigValueAsString("password")
	if err != nil {
		return nil, err
	}
	return &s3.S3TenantManager{
		Endpoint:  endpoint,
		Insecure:  insecure,
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Password:  password,
	}, nil
}
