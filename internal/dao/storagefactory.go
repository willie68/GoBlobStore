package dao

import (
	"errors"
	"fmt"

	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
)

var _ BlobStorageDao = &simplefile.SimpleFileBlobStorageDao{}

var tenantStores map[string]*BlobStorageDao
var tenantDao TenantDao

var storageClass string
var config map[string]interface{}

func Init(cnfg map[string]interface{}) error {
	tenantStores = make(map[string]*BlobStorageDao)
	config = cnfg
	stgClass, ok := cnfg["storageclass"].(string)
	if !ok {
		return errors.New("no storage class given")
	}
	if stgClass == "" {
		return errors.New("no dao implemented")
	}
	storageClass = stgClass
	return nil
}

func GetTenantDao() (TenantDao, error) {
	if tenantDao == nil {
		tDao, err := createTenantDao()
		if err != nil {
			return nil, err
		}
		tenantDao = tDao
	}
	return tenantDao, nil
}

func createTenantDao() (TenantDao, error) {
	switch storageClass {
	case "SimpleFile":
		if _, ok := config["rootpath"]; !ok {
			return nil, fmt.Errorf("missing config value for %s", "rootpath")
		}
		rootpath, ok := config["rootpath"].(string)
		if !ok {
			return nil, fmt.Errorf("config value for %s is not a string", "rootpath")
		}
		dao := &simplefile.SimpleFileTenantManager{
			RootPath: rootpath,
		}
		err := dao.Init()
		if err != nil {
			return nil, err
		}
		return dao, nil
	}
	return nil, fmt.Errorf("no tenantmanager class implementation for \"%s\" found", storageClass)
}

func GetStorageDao(tenant string) (BlobStorageDao, error) {
	storageDao, ok := tenantStores[tenant]
	if !ok {
		storageDao, err := createStorage(tenant)
		if err != nil {
			return nil, err
		}
		tenantStores[tenant] = &storageDao
		return storageDao, nil
	}
	return *storageDao, nil
}

func createStorage(tenant string) (BlobStorageDao, error) {
	switch storageClass {
	case "SimpleFile":
		if _, ok := config["rootpath"]; !ok {
			return nil, fmt.Errorf("missing config value for %s", "rootpath")
		}
		rootpath, ok := config["rootpath"].(string)
		if !ok {
			return nil, fmt.Errorf("config value for %s is not a string", "rootpath")
		}
		dao := &simplefile.SimpleFileBlobStorageDao{
			RootPath: rootpath,
			Tenant:   tenant,
		}
		err := dao.Init()
		if err != nil {
			return nil, err
		}
		return dao, nil
	}
	return nil, fmt.Errorf("no storage class implementation for \"%s\" found", storageClass)
}
