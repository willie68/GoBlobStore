package dao

import (
	"errors"

	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/migration"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/factory"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
)

// test for interface compatibility

var tenantDao interfaces.TenantDao
var rtnMgr interfaces.RetentionManager
var cnfg config.Engine
var stgf interfaces.StorageFactory
var migMan *migration.MigrationManagement

// Init initialise the storage factory
func Init(storage config.Engine) error {
	cnfg = storage
	if cnfg.Storage.Storageclass == "" {
		return errors.New("no storage class given")
	}

	var bktDao interfaces.TenantDao
	tntDao, err := factory.CreateTenantDao(cnfg.Storage)
	if err != nil {
		return err
	}
	if cnfg.Backup.Storageclass != "" {
		bktDao, err = factory.CreateTenantDao(cnfg.Backup)
		if err != nil {
			return err
		}
	}
	tenantDao = &business.MainTenantDao{
		TntDao: tntDao,
		BckDao: bktDao,
	}

	if cnfg.RetentionManager == "" {
		return errors.New("no retention class given")
	}

	// this order of creation of factories is crucial, because the RetentionManager needs the StorageFactory and other way round
	stgf = &factory.DefaultStorageFactory{
		TenantDao: tenantDao,
	}

	rtnMgr, err = factory.CreateRetentionManager(cnfg.RetentionManager, tenantDao)
	if err != nil {
		return err
	}

	err = stgf.Init(storage, rtnMgr)
	if err != nil {
		return err
	}

	err = rtnMgr.Init(stgf)
	if err != nil {
		return err
	}

	migMan = &migration.MigrationManagement{
		StorageFactory: stgf,
	}
	err = migMan.Init()
	if err != nil {
		return err
	}

	// migrate backup
	err = migration.MigrateBackup(tenantDao, stgf)
	if err != nil {
		return err
	}
	return nil
}

// GetTenantDao returning the tenant for administration tenants
func GetTenantDao() (interfaces.TenantDao, error) {
	if tenantDao == nil {
		return nil, errors.New("no tenantdao present")
	}
	return tenantDao, nil
}

// GetStorageFactory returning the storage factory
func GetStorageFactory() (interfaces.StorageFactory, error) {
	if stgf == nil {
		return nil, errors.New("no storage factory present")
	}
	return stgf, nil
}

// GetMigrationManagement returning the tenant for administration tenants
func GetMigrationManagement() (*migration.MigrationManagement, error) {
	if migMan == nil {
		return nil, errors.New("no check management present")
	}
	return migMan, nil
}

// Close closing ths storage factory
func Close() {
	err := stgf.Close()
	if err != nil {
		log.Logger.Errorf("error closing storage factory:\r\n%v,", err)
	}

	err = rtnMgr.Close()
	if err != nil {
		log.Logger.Errorf("error closing retention manager:\r\n%v,", err)
	}

	err = tenantDao.Close()
	if err != nil {
		log.Logger.Errorf("error closing tenant dao:\r\n%v,", err)
	}

	err = migMan.Close()
	if err != nil {
		log.Logger.Errorf("error closing check management:\r\n%v,", err)
	}
}
