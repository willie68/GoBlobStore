package services

import (
	"errors"

	"github.com/samber/do"
	"github.com/willie68/GoBlobStore/internal/services/business"
	"github.com/willie68/GoBlobStore/internal/services/migration"

	"github.com/willie68/GoBlobStore/internal/config"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/services/factory"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

const (
	DoTntSrv = "tntsrv"
	DoRtnMgr = "rtnmgr"
	DoStgf   = "stgf"
	DoMigMgr = "migmgr"
)

var tntsrv interfaces.TenantManager
var rtnMgr interfaces.RetentionManager
var cnfg config.Engine
var stgf interfaces.StorageFactory
var migMan *migration.Management

// Init initialize the storage factory
func Init(storage config.Engine) error {
	cnfg = storage
	if cnfg.Storage.Storageclass == "" {
		return errors.New("no storage class given")
	}

	var bktsrv interfaces.TenantManager
	tntMgr, err := factory.CreateTenantManager(cnfg.Storage)
	if err != nil {
		return err
	}

	if cnfg.Backup.Storageclass != "" {
		bktsrv, err = factory.CreateTenantManager(cnfg.Backup)
		if err != nil {
			return err
		}
	}

	tntsrv = &business.MainTenant{
		TntSrv: tntMgr,
		BckSrv: bktsrv,
	}

	do.ProvideNamedValue[interfaces.TenantManager](nil, DoTntSrv, tntsrv)

	if cnfg.RetentionManager == "" {
		return errors.New("no retention class given")
	}

	// this order of creation of factories is crucial, because the RetentionManager needs the StorageFactory and other way round
	stgf = &factory.DefaultStorageFactory{
		TenantMgr: tntsrv,
	}

	rtnMgr, err = factory.CreateRetentionManager(cnfg.RetentionManager, tntsrv)
	if err != nil {
		return err
	}

	do.ProvideNamedValue[interfaces.RetentionManager](nil, DoRtnMgr, rtnMgr)

	err = stgf.Init(storage, rtnMgr)
	if err != nil {
		return err
	}

	do.ProvideNamedValue[interfaces.StorageFactory](nil, DoStgf, stgf)

	err = rtnMgr.Init(stgf)
	if err != nil {
		return err
	}

	migMan = &migration.Management{
		StorageFactory: stgf,
	}
	err = migMan.Init()
	if err != nil {
		return err
	}

	// migrate backup
	err = migration.MigrateBackup(tntsrv, stgf)
	if err != nil {
		return err
	}

	do.ProvideNamedValue[*migration.Management](nil, DoMigMgr, migMan)

	return nil
}

// GetTenantSrv returning the tenant for administration tenants
func GetTenantSrv() (interfaces.TenantManager, error) {
	if tntsrv == nil {
		return nil, errors.New("no tenant service present")
	}
	return tntsrv, nil
}

// GetStorageFactory returning the storage factory
func GetStorageFactory() (interfaces.StorageFactory, error) {
	if stgf == nil {
		return nil, errors.New("no storage factory present")
	}
	return stgf, nil
}

// GetMigrationManagement returning the tenant for administration tenants
func GetMigrationManagement() (*migration.Management, error) {
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

	err = tntsrv.Close()
	if err != nil {
		log.Logger.Errorf("error closing tenant service:\r\n%v,", err)
	}

	err = migMan.Close()
	if err != nil {
		log.Logger.Errorf("error closing check management:\r\n%v,", err)
	}
}
