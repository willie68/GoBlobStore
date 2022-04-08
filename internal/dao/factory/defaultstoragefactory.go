package factory

import (
	"errors"
	"fmt"
	"strings"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/bluge"
	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/fastcache"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/mongodb"
	"github.com/willie68/GoBlobStore/internal/dao/s3"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
	log "github.com/willie68/GoBlobStore/internal/logging"
)

const STGCLASS_SIMPLE_FILE = "SimpleFile"
const STGCLASS_S3 = "S3Storage"

const STGCLASS_FASTCACHE = "FastCache"

var NO_STG_ERROR = errors.New("no storage class given")

type DefaultStorageFactory struct {
	TenantDao    interfaces.TenantDao
	RtnMgr       interfaces.RetentionManager
	CchDao       interfaces.BlobStorageDao
	tenantStores map[string]*interfaces.BlobStorageDao
	cnfg         config.Engine
}

func (d *DefaultStorageFactory) Init(storage config.Engine, rtnm interfaces.RetentionManager) error {
	d.tenantStores = make(map[string]*interfaces.BlobStorageDao)
	d.cnfg = storage
	d.RtnMgr = rtnm
	if d.cnfg.Index.Storageclass != "" {
		d.initIndex(d.cnfg.Index)
	}
	return nil
}

//GetStorageDao return the storage dao for the desired tenant
func (d *DefaultStorageFactory) GetStorageDao(tenant string) (interfaces.BlobStorageDao, error) {
	storageDao, ok := d.tenantStores[tenant]
	if !ok {
		storageDao, err := d.createStorage(tenant)
		if err != nil {
			log.Logger.Errorf("can't create storage for tenant: %s\n %v", tenant, err)
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

	idxdao, err := d.getImplIdxDao(d.cnfg.Index, tenant)
	if err != nil {
		return nil, err
	}

	mdao := &business.MainStorageDao{
		Bcksyncmode: d.cnfg.BackupSyncmode,
		RtnMng:      d.RtnMgr,
		StgDao:      dao,
		BckDao:      bckdao,
		CchDao:      cchdao,
		IdxDao:      idxdao,
		Tenant:      tenant,
	}
	err = mdao.Init()
	if err != nil {
		return nil, err
	}
	return mdao, nil
}

func (d *DefaultStorageFactory) getImplIdxDao(stg config.Storage, tenant string) (interfaces.Index, error) {
	var dao interfaces.Index

	if stg.Storageclass != "" {
		s := stg.Storageclass
		s = strings.ToLower(s)
		switch s {
		case bluge.BLUGE_INDEX:
			dao = &bluge.Index{
				Tenant: tenant,
			}
			err := dao.Init()
			if err != nil {
				return nil, err
			}
		case mongodb.MONGO_INDEX:
			dao = &mongodb.Index{
				Tenant: tenant,
			}
			err := dao.Init()
			if err != nil {
				return nil, err
			}
		}
	}
	if dao == nil {
		return nil, fmt.Errorf("no searcher indexer class implementation for \"%s\" found. %w", stg.Storageclass, NO_STG_ERROR)
	}
	return dao, nil
}

func (d *DefaultStorageFactory) getImplStgDao(stg config.Storage, tenant string) (interfaces.BlobStorageDao, error) {
	var dao interfaces.BlobStorageDao
	var err error
	switch stg.Storageclass {
	case STGCLASS_SIMPLE_FILE:
		rootpath, err := config.GetConfigValueAsString(stg.Properties, "rootpath")
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
	}

	if dao == nil {
		return nil, fmt.Errorf("no storage class implementation for \"%s\" found. %w", stg.Storageclass, NO_STG_ERROR)
	}
	return dao, nil
}

func (d *DefaultStorageFactory) getS3Storage(stg config.Storage, tenant string) (*s3.S3BlobStorage, error) {
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

func (d *DefaultStorageFactory) getFastcache(stg config.Storage, tenant string) (interfaces.BlobStorageDao, error) {
	// as cache there will be always the same instance delivered
	if d.CchDao == nil {

		rootpath, err := config.GetConfigValueAsString(stg.Properties, "rootpath")
		if err != nil {
			return nil, err
		}
		maxcount, err := config.GetConfigValueAsInt(stg.Properties, "maxcount")
		if err != nil {
			return nil, err
		}
		ramusage, err := config.GetConfigValueAsInt(stg.Properties, "maxramusage")
		if err != nil {
			return nil, err
		}
		mffrs, err := config.GetConfigValueAsInt(stg.Properties, "maxfilesizeforram")
		if err != nil {
			mffrs = fastcache.Defaultmffrs
		}
		d.CchDao = &fastcache.FastCache{
			RootPath:          rootpath,
			MaxCount:          maxcount,
			MaxRamSize:        ramusage,
			MaxFileSizeForRAM: mffrs,
		}
		err = d.CchDao.Init()
		if err != nil {
			return nil, err
		}
	}
	return d.CchDao, nil
}

func (d *DefaultStorageFactory) initIndex(cnfg config.Storage) error {
	//TODO initialise the index storage
	s := cnfg.Storageclass
	s = strings.ToLower(s)
	switch s {
	case bluge.BLUGE_INDEX:
		bluge.InitBluge(cnfg.Properties)
	case mongodb.MONGO_INDEX:
		mongodb.InitMongoDB(cnfg.Properties)
	}
	return nil
}

func (d *DefaultStorageFactory) Close() error {
	var tDao interfaces.BlobStorageDao
	for k, v := range d.tenantStores {
		tDao = *v
		err := tDao.Close()
		if err != nil {
			log.Logger.Errorf("error closing tenant storage dao: %s\r\n%v,", k, err)
		}
	}
	return nil
}
