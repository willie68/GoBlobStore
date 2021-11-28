package factory

import (
	"errors"
	"fmt"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/fastcache"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/s3"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
	clog "github.com/willie68/GoBlobStore/internal/logging"
)

const STGCLASS_SIMPLE_FILE = "SimpleFile"
const STGCLASS_S3 = "S3Storage"

const STGCLASS_FASTCACHE = "FastCache"

var NO_STG_ERROR = errors.New("no storage class given")

type DefaultStorageFactory struct {
	TenantDao    interfaces.TenantDao
	RtnMgr       interfaces.RetentionManager
	tenantStores map[string]*interfaces.BlobStorageDao
	cnfg         config.Engine
}

func (d *DefaultStorageFactory) Init(storage config.Engine, rtnm interfaces.RetentionManager) error {
	d.tenantStores = make(map[string]*interfaces.BlobStorageDao)
	d.cnfg = storage
	d.RtnMgr = rtnm
	return nil
}

//GetStorageDao return the storage dao for the desired tenant
func (d *DefaultStorageFactory) GetStorageDao(tenant string) (interfaces.BlobStorageDao, error) {
	storageDao, ok := d.tenantStores[tenant]
	if !ok {
		storageDao, err := d.createStorage(tenant)
		if err != nil {
			clog.Logger.Errorf("can't create storage for tenant: %s\n %v", tenant, err)
			return nil, err
		}
		d.tenantStores[tenant] = &storageDao
		return storageDao, nil
	}
	return *storageDao, nil
}

// createStorage creating a new storage dao for the tenant depending on the configuration
func (d *DefaultStorageFactory) createStorage(tenant string) (interfaces.BlobStorageDao, error) {
	if !d.TenantDao.HasTenant(tenant) {
		if d.cnfg.Tenantautoadd {
			d.TenantDao.AddTenant(tenant)
		} else {
			return nil, errors.New("tenant not exists")
		}
	}
	dao, err := d.getImplStgDao(d.cnfg.Storage, tenant)
	if err != nil {
		return nil, err
	}

	bckdao, err := d.getImplStgDao(d.cnfg.Backup, tenant)
	if err != nil && !errors.Is(err, NO_STG_ERROR) {
		return nil, err
	}

	cchdao, err := d.getImplStgDao(d.cnfg.Cache, "blbstg")
	if err != nil && !errors.Is(err, NO_STG_ERROR) {
		return nil, err
	}

	return &business.MainStorageDao{
		Bcksyncmode: d.cnfg.BackupSyncmode,
		RtnMng:      d.RtnMgr,
		StgDao:      dao,
		BckDao:      bckdao,
		CchDao:      cchdao,
		Tenant:      tenant,
	}, nil
}

func (d *DefaultStorageFactory) getImplStgDao(stg config.Storage, tenant string) (interfaces.BlobStorageDao, error) {
	var dao interfaces.BlobStorageDao
	var err error
	switch stg.Storageclass {
	case STGCLASS_SIMPLE_FILE:
		rootpath, err := config.GetConfigValueAsString(stg, "rootpath")
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
		dao, err = d.getS3Storage(stg, tenant)
		if err != nil {
			return nil, err
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
	case STGCLASS_FASTCACHE:
		dao, err = d.getFastcache(stg, tenant)
		if err != nil {
			return nil, err
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
	}

	if dao == nil {
		return nil, fmt.Errorf("no storage class implementation for \"%s\" found. %w", stg.Storageclass, NO_STG_ERROR)
	}
	return dao, nil
}

func (d *DefaultStorageFactory) getS3Storage(stg config.Storage, tenant string) (*s3.S3BlobStorage, error) {
	endpoint, err := config.GetConfigValueAsString(stg, "endpoint")
	if err != nil {
		return nil, err
	}
	insecure, err := config.GetConfigValueAsBool(stg, "insecure")
	if err != nil {
		return nil, err
	}
	bucket, err := config.GetConfigValueAsString(stg, "bucket")
	if err != nil {
		return nil, err
	}
	accessKey, err := config.GetConfigValueAsString(stg, "accessKey")
	if err != nil {
		return nil, err
	}
	secretKey, err := config.GetConfigValueAsString(stg, "secretKey")
	if err != nil {
		return nil, err
	}
	password, err := config.GetConfigValueAsString(stg, "password")
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

func (d *DefaultStorageFactory) getFastcache(stg config.Storage, tenant string) (*fastcache.FastCache, error) {
	rootpath, err := config.GetConfigValueAsString(stg, "rootpath")
	if err != nil {
		return nil, err
	}
	maxcount, err := config.GetConfigValueAsInt(stg, "maxcount")
	if err != nil {
		return nil, err
	}
	ramusage, err := config.GetConfigValueAsInt(stg, "maxramusage")
	if err != nil {
		return nil, err
	}
	return &fastcache.FastCache{
		RootPath:   rootpath,
		MaxCount:   maxcount,
		MaxRamSize: ramusage,
	}, nil
}

func (d *DefaultStorageFactory) Close() error {
	var tDao interfaces.BlobStorageDao
	for k, v := range d.tenantStores {
		tDao = *v
		err := tDao.Close()
		if err != nil {
			clog.Logger.Errorf("error closing tenant storage dao: %s\r\n%v,", k, err)
		}
	}
	return nil
}
