package factory

import (
	"fmt"
	"strings"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/s3"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
)

// createTenantDao creating a new tenant dao depending on the configuration
func CreateTenantDao(stg config.Storage) (interfaces.TenantDao, error) {
	stgcl := strings.ToLower(stg.Storageclass)
	switch stgcl {
	case STGCLASS_SIMPLE_FILE:
		rootpath, err := config.GetConfigValueAsString(stg.Properties, "rootpath")
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
		dao, err := getS3TenantManager(stg)
		if err != nil {
			return nil, err
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
		return dao, nil
	}
	return nil, fmt.Errorf("no tenantmanager class implementation for \"%s\" found", stg.Storageclass)
}

func getS3TenantManager(stg config.Storage) (*s3.S3TenantManager, error) {
	endpoint, err := config.GetConfigValueAsString(stg.Properties, "endpoint")
	if err != nil {
		return nil, err
	}
	insecure, err := config.GetConfigValueAsBool(stg.Properties, "insecure")
	if err != nil {
		return nil, err
	}
	bucket, err := config.GetConfigValueAsString(stg.Properties, "bucket")
	if err != nil {
		return nil, err
	}
	accessKey, err := config.GetConfigValueAsString(stg.Properties, "accessKey")
	if err != nil {
		return nil, err
	}
	secretKey, err := config.GetConfigValueAsString(stg.Properties, "secretKey")
	if err != nil {
		return nil, err
	}
	password, err := config.GetConfigValueAsString(stg.Properties, "password")
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
