package dao

import (
	"errors"
	"fmt"

	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
)

var _ BlobStorageDao = &simplefile.SimpleFileBlobStorageDao{}

var tenantStores map[string]*BlobStorageDao
var tenantDao TenantDao

var rtgMgrStr string
var retMgn RetentionManager

var storageClass string
var config map[string]interface{}

//Init initialise the storage factory
func Init(cnfg map[string]interface{}) error {
	tenantStores = make(map[string]*BlobStorageDao)
	config = cnfg
	stgClass, ok := cnfg["storageclass"].(string)
	if !ok || stgClass == "" {
		return errors.New("no storage class given")
	}
	storageClass = stgClass

	rtgMgrStr, ok = cnfg["retentionManager"].(string)
	if !ok || rtgMgrStr == "" {
		return errors.New("no retention class given")
	}
	err := createRetentionManager()
	if err != nil {
		return err
	}
	return nil
}

//GetTenantDao returning the tenant for administration tenants
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

// createTenantDao creating a new tenant dao depending on the configuration
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

//GetStorageDao return the storage dao for the desired tenant
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

// createStorage creating a new storage dao for the tenant depending on the configuration
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
		return &mainStorageDao{
			retMng:     retMgn,
			storageDao: dao,
			tenant:     tenant,
		}, nil
	}
	return nil, fmt.Errorf("no storage class implementation for \"%s\" found", storageClass)
}

// createRetentionManager creates a new Retention manager depending o nthe configuration
func createRetentionManager() error {
	switch rtgMgrStr {
	//This is the single node retention manager
	case "SingleRetention":
		retMgn = &SingleRetentionManager{
			tntDao: tenantDao,
		}
	default:
		return fmt.Errorf("no rentention manager found for class: %s", rtgMgrStr)
	}
	return nil
}
