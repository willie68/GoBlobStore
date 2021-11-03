package dao

import (
	"errors"

	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/factory"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	clog "github.com/willie68/GoBlobStore/internal/logging"
)

// test for interface compatibility

var tenantDao interfaces.TenantDao
var rtnMgr interfaces.RetentionManager
var cnfg config.Engine
var StorageFactory interfaces.StorageFactory

//Init initialise the storage factory
func Init(storage config.Engine) error {
	cnfg = storage
	if cnfg.Storage.Storageclass == "" {
		return errors.New("no storage class given")
	}

	tntDao, err := createTenantDao(cnfg.Storage)
	if err != nil {
		return err
	}
	tenantDao = tntDao

	if cnfg.RetentionManager == "" {
		return errors.New("no retention class given")
	}

	err = createRetentionManager(cnfg.RetentionManager)
	if err != nil {
		return err
	}

	StorageFactory = &factory.DefaultStorageFactory{
		TenantDao: tenantDao,
		RtnMgr:    rtnMgr,
	}

	err = StorageFactory.Init(storage)
	if err != nil {
		return err
	}

	err = rtnMgr.Init()
	if err != nil {
		return err
	}

	return nil
}

func Close() {
	err := StorageFactory.Close()
	if err != nil {
		clog.Logger.Errorf("error closing storage factory:\r\n%v,", err)
	}

	err = rtnMgr.Close()
	if err != nil {
		clog.Logger.Errorf("error closing retention manager:\r\n%v,", err)
	}

	err = tenantDao.Close()
	if err != nil {
		clog.Logger.Errorf("error closing tenant dao:\r\n%v,", err)
	}
}
