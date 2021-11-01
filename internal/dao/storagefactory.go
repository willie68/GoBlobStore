package dao

import (
	"errors"
	"fmt"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/s3"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
	clog "github.com/willie68/GoBlobStore/internal/logging"
)

const STGCLASS_SIMPLE_FILE = "SimpleFile"
const STGCLASS_S3 = "S3Storage"

// test for interface compatibility

var tenantStores map[string]*interfaces.BlobStorageDao
var tenantDao interfaces.TenantDao

var rtnMgr interfaces.RetentionManager
var cnfg config.Storage
var bck config.Backup

//Init initialise the storage factory
func Init(storage config.Storage, backup config.Backup) error {
	tenantStores = make(map[string]*interfaces.BlobStorageDao)
	cnfg = storage
	bck = backup
	if cnfg.Storageclass == "" {
		return errors.New("no storage class given")
	}

	tntDao, err := createTenantDao(cnfg.Storageclass)
	if err != nil {
		return err
	}
	tenantDao = tntDao

	if storage.RetentionManager == "" {
		return errors.New("no retention class given")
	}

	err = createRetentionManager(storage.RetentionManager)
	if err != nil {
		return err
	}
	return nil
}

//GetStorageDao return the storage dao for the desired tenant
func GetStorageDao(tenant string) (interfaces.BlobStorageDao, error) {
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
func createStorage(tenant string) (interfaces.BlobStorageDao, error) {
	if !tenantDao.HasTenant(tenant) {
		if cnfg.Tenantautoadd {
			tenantDao.AddTenant(tenant)
		} else {
			return nil, errors.New("tenant not exists")
		}
	}
	var dao interfaces.BlobStorageDao
	switch cnfg.Storageclass {
	case STGCLASS_SIMPLE_FILE:
		rootpath, err := getConfigValueAsString("rootpath")
		if err != nil {
			return nil, err
		}
		dao = &simplefile.SimpleFileBlobStorageDao{
			RootPath: rootpath,
			Tenant:   tenant,
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
	case STGCLASS_S3:
		dao, err := getS3Storage(tenant)
		if err != nil {
			return nil, err
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
	}

	if dao == nil {
		return nil, fmt.Errorf("no storage class implementation for \"%s\" found", cnfg.Storageclass)
	}
	return &mainStorageDao{
		rtnMng: rtnMgr,
		stgDao: dao,
		tenant: tenant,
	}, nil
}

func getS3Storage(tenant string) (*s3.S3BlobStorage, error) {
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
	return &s3.S3BlobStorage{
		Endpoint:  endpoint,
		Insecure:  insecure,
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Tenant:    tenant,
		Password:  password,
	}, nil
}

func Close() {
	var tDao interfaces.BlobStorageDao
	for k, v := range tenantStores {
		tDao = *v
		err := tDao.Close()
		if err != nil {
			clog.Logger.Errorf("error closing tenant storage dao: %s\r\n%v,", k, err)
		}
	}

	err := rtnMgr.Close()
	if err != nil {
		clog.Logger.Errorf("error closing retention manager:\r\n%v,", err)
	}

	err = tenantDao.Close()
	if err != nil {
		clog.Logger.Errorf("error closing tenant dao:\r\n%v,", err)
	}
}
