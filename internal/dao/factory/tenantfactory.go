package factory

import (
	"fmt"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/s3"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
)

// createTenantDao creating a new tenant dao depending on the configuration
func CreateTenantDao(stgCfng config.Storage) (interfaces.TenantDao, error) {
	switch stgCfng.Storageclass {
	case STGCLASS_SIMPLE_FILE:
		rootpath, err := config.GetConfigValueAsString(stgCfng, "rootpath")
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
		dao, err := getS3TenantManager(stgCfng)
		if err != nil {
			return nil, err
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
		return dao, nil
	}
	return nil, fmt.Errorf("no tenantmanager class implementation for \"%s\" found", stgCfng.Storageclass)
}

func getS3TenantManager(stgCfng config.Storage) (*s3.S3TenantManager, error) {
	endpoint, err := config.GetConfigValueAsString(stgCfng, "endpoint")
	if err != nil {
		return nil, err
	}
	insecure, err := config.GetConfigValueAsBool(stgCfng, "insecure")
	if err != nil {
		return nil, err
	}
	bucket, err := config.GetConfigValueAsString(stgCfng, "bucket")
	if err != nil {
		return nil, err
	}
	accessKey, err := config.GetConfigValueAsString(stgCfng, "accessKey")
	if err != nil {
		return nil, err
	}
	secretKey, err := config.GetConfigValueAsString(stgCfng, "secretKey")
	if err != nil {
		return nil, err
	}
	password, err := config.GetConfigValueAsString(stgCfng, "password")
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
