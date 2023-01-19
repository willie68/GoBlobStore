package factory

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/bluge"
	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/fastcache"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/mongodb"
	"github.com/willie68/GoBlobStore/internal/dao/noindex"
	"github.com/willie68/GoBlobStore/internal/dao/s3"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
	log "github.com/willie68/GoBlobStore/internal/logging"
)

// name of storage classes
const (
	STGClassSimpleFile = "simplefile"
	STGClassS3         = "s3storage"
	STGClassFastcache  = "fastcache"
	STGClassSFMV       = "sfmv"
)

// ErrNoStg error for no storage class given
var ErrNoStg = errors.New("no storage class given")

// just to check interface compatibility
var _ interfaces.StorageFactory = &DefaultStorageFactory{}

// DefaultStorageFactory the struct for the default storage factory
type DefaultStorageFactory struct {
	TenantMgr    interfaces.TenantManager
	RtnMgr       interfaces.RetentionManager
	CchDao       interfaces.BlobStorage
	tenantStores sync.Map
	cnfg         config.Engine
}

// Init initialize the factory
func (d *DefaultStorageFactory) Init(storage config.Engine, rtnm interfaces.RetentionManager) error {
	d.tenantStores = sync.Map{}
	d.cnfg = storage
	d.RtnMgr = rtnm
	if d.cnfg.Index.Storageclass != "" {
		err := d.initIndex(d.cnfg.Index)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetStorage return the storage for the desired tenant
func (d *DefaultStorageFactory) GetStorage(tenant string) (interfaces.BlobStorage, error) {
	dao, ok := d.tenantStores.Load(tenant)
	if !ok {
		storageDao, err := d.createStorage(tenant)
		if err != nil {
			log.Logger.Errorf("can't create storage for tenant: %s\n %v", tenant, err)
			return nil, err
		}
		d.tenantStores.Store(tenant, &storageDao)
		return storageDao, nil
	}
	storageDao, ok := dao.(*interfaces.BlobStorage)
	if !ok {
		return nil, fmt.Errorf("wrong type in map for tenant %s", tenant)
	}
	return *storageDao, nil
}

// RemoveStorage removes a tenant storage from the cache
func (d *DefaultStorageFactory) RemoveStorage(tenant string) error {
	dao, ok := d.tenantStores.Load(tenant)
	if ok {
		storageDao, ok := dao.(*interfaces.BlobStorage)
		if ok {
			err := (*storageDao).Close()
			if err != nil {
				log.Logger.Errorf("can't close storage for tenant: %s\n %v", tenant, err)
				return err
			}
		}
		d.tenantStores.Delete(tenant)
	}
	return nil
}

// createStorage creating a new storage dao for the tenant depending on the configuration
func (d *DefaultStorageFactory) createStorage(tenant string) (interfaces.BlobStorage, error) {
	if !d.TenantMgr.HasTenant(tenant) {
		if !d.cnfg.Tenantautoadd {
			return nil, errors.New("tenant not exists")
		}
		err := d.TenantMgr.AddTenant(tenant)
		if err != nil {
			return nil, err
		}
	}
	dao, err := d.getImplStg(d.cnfg.Storage, tenant)
	if err != nil {
		return nil, err
	}

	bckdao, err := d.getImplStg(d.cnfg.Backup, tenant)
	if err != nil && !errors.Is(err, ErrNoStg) {
		return nil, err
	}

	cchdao, err := d.getImplStg(d.cnfg.Cache, "blbstg")
	if err != nil && !errors.Is(err, ErrNoStg) {
		return nil, err
	}

	idxdao, err := d.getImplIdx(d.cnfg.Index, tenant)
	if err != nil {
		return nil, err
	}

	// creating the tenant specific backup storage
	// an error in this part should prevent the startup of the service,
	// so the last error will be stored into the tenant main storage dao
	var lasterror error
	tntBckDao, err := d.getTntBck(tenant)
	if err != nil {
		lasterror = err
	}

	mdao := &business.MainStorage{
		Bcksyncmode: d.cnfg.BackupSyncmode,
		RtnMng:      d.RtnMgr,
		StgDao:      dao,
		BckDao:      bckdao,
		CchDao:      cchdao,
		IdxDao:      idxdao,
		Tenant:      tenant,
		TntBckDao:   tntBckDao,
		TntError:    lasterror,
	}
	err = mdao.Init()
	if err != nil {
		return nil, err
	}
	return mdao, nil
}

func (d *DefaultStorageFactory) getImplIdx(stg config.Storage, tenant string) (interfaces.Index, error) {
	var dao interfaces.Index
	if stg.Storageclass != "" {
		s := stg.Storageclass
		s = strings.ToLower(s)
		switch s {
		case bluge.BlugeIndex:
			dao = &bluge.Index{
				Tenant: tenant,
			}
			err := dao.Init()
			if err != nil {
				return nil, err
			}
		case mongodb.MongoIndex:
			dao = &mongodb.Index{
				Tenant: tenant,
			}
			err := dao.Init()
			if err != nil {
				return nil, err
			}
		}
	} else {
		dao = &noindex.Index{}
	}
	if dao == nil {
		return nil, fmt.Errorf("no searcher indexer class implementation for \"%s\" found. %w", stg.Storageclass, ErrNoStg)
	}
	return dao, nil
}

func (d *DefaultStorageFactory) getTntBck(tenant string) (interfaces.BlobStorage, error) {
	tntCfg, err := d.TenantMgr.GetConfig(tenant)
	if err != nil {
		return nil, err
	}

	var lasterror error
	var tntBckDao interfaces.BlobStorage
	if tntCfg != nil {
		// we have to set a password and client side encryption is not supported
		tntCfg.Backup.Properties["password"] = tenant
		tntCfg.Backup.Properties["insecure"] = true
		tntBckDao, err = d.getImplStg(tntCfg.Backup, tenant)
		if err != nil {
			log.Logger.Errorf("Tnt: %s, error in tenant backup storage creation: %v", tenant, err)
			lasterror = err
		}
	}
	if lasterror != nil {
		return nil, lasterror
	}
	return tntBckDao, nil
}

func (d *DefaultStorageFactory) getImplStg(stg config.Storage, tenant string) (interfaces.BlobStorage, error) {
	var dao interfaces.BlobStorage
	var err error
	stgcl := strings.ToLower(stg.Storageclass)
	switch stgcl {
	case STGClassSFMV:
		rootpath, err := config.GetConfigValueAsString(stg.Properties, "rootpath")
		if err != nil {
			return nil, err
		}
		dao = &simplefile.MultiVolumeStorage{
			RootPath: rootpath,
			Tenant:   tenant,
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
	case STGClassSimpleFile:
		rootpath, err := config.GetConfigValueAsString(stg.Properties, "rootpath")
		if err != nil {
			return nil, err
		}
		dao = &simplefile.BlobStorage{
			RootPath: rootpath,
			Tenant:   tenant,
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
	case STGClassS3:
		dao, err = d.getS3Storage(stg, tenant)
		if err != nil {
			return nil, err
		}
		err = dao.Init()
		if err != nil {
			return nil, err
		}
	case STGClassFastcache:
		dao, err = d.getFastcache(stg, tenant)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("no storage class implementation for \"%s\" found. %w", stg.Storageclass, ErrNoStg)
	}

	return dao, nil
}

func (d *DefaultStorageFactory) getS3Storage(stg config.Storage, tenant string) (*s3.BlobStorage, error) {
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
	password := tenant
	if !insecure {
		password, err = config.GetConfigValueAsString(stg.Properties, "password")
		if err != nil {
			return nil, err
		}
	}
	return &s3.BlobStorage{
		Endpoint:  endpoint,
		Insecure:  insecure,
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Tenant:    tenant,
		Password:  password,
	}, nil
}

func (d *DefaultStorageFactory) getFastcache(stg config.Storage, _ string) (interfaces.BlobStorage, error) {
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
			MaxRAMSize:        ramusage,
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
	case bluge.BlugeIndex:
		bluge.InitBluge(cnfg.Properties)
	case mongodb.MongoIndex:
		mongodb.InitMongoDB(cnfg.Properties)
	}
	return nil
}

// Close colsing this default storage factory
func (d *DefaultStorageFactory) Close() error {
	d.tenantStores.Range(func(key, v any) bool {
		tDao, ok := v.(*interfaces.BlobStorage)
		if ok {
			err := (*tDao).Close()
			if err != nil {
				log.Logger.Errorf("error closing tenant storage dao: %s\r\n%v,", key, err)
			}
		}
		return true
	})
	return nil
}
