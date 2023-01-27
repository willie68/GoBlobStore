package factory

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/noindex"
	"github.com/willie68/GoBlobStore/internal/dao/retentionmanager"
	"github.com/willie68/GoBlobStore/internal/dao/simplefile"
)

const (
	rootFilePrefix = "../../../testdata/dsf"
	tenant         = "test"
	blbcount       = 1000
)

var (
	blbPath = filepath.Join(rootFilePrefix, "blbstg")
	cchPath = filepath.Join(rootFilePrefix, "blbcch")
	bckPath = filepath.Join(rootFilePrefix, "bckstg")
	tntPath = filepath.Join(rootFilePrefix, "tntstg")
)

func TestTenantBckStg(t *testing.T) {
	// create a new default storage factory
	ast := assert.New(t)
	tntMgr := &simplefile.TenantManager{
		RootPath: rootFilePrefix,
	}
	err := tntMgr.Init()
	ast.Nil(err)

	stgf := &DefaultStorageFactory{
		TenantMgr: tntMgr,
	}

	rtnMgr := &retentionmanager.NoRetention{}
	err = rtnMgr.Init(stgf)
	ast.Nil(err)

	storage := config.Engine{
		RetentionManager: retentionmanager.SingleRetentionManagerName,
		Tenantautoadd:    true,
		BackupSyncmode:   false,
		AllowTntBackup:   true,
		Storage: config.Storage{
			Storageclass: STGClassSimpleFile,
			Properties: map[string]any{
				"rootpath": blbPath,
			},
		},
		Backup: config.Storage{
			Storageclass: STGClassSimpleFile,
			Properties: map[string]any{
				"rootpath": bckPath,
			},
		},
		Cache: config.Storage{
			Storageclass: STGClassFastcache,
			Properties: map[string]any{
				"rootpath":    bckPath,
				"maxcount":    1000,
				"maxramusage": 1024,
			},
		},
		Index: config.Storage{
			Storageclass: noindex.NoIndexName,
		},
	}
	err = stgf.Init(storage, rtnMgr)
	ast.Nil(err)

	err = tntMgr.AddTenant(tenant)
	ast.Nil(err)
	tntCfg := interfaces.TenantConfig{
		Backup: config.Storage{
			Storageclass: STGClassSimpleFile,
			Properties: map[string]any{
				"rootpath": tntPath,
			},
		},
	}
	tntMgr.SetConfig(tenant, tntCfg)

	bs, err := stgf.GetStorage(tenant)
	ast.Nil(err)
	ast.NotNil(bs)

	_, err = stgf.GetStorage("tnt2")
	ast.Nil(err)

	err = stgf.RemoveStorage("tnt2")
	ast.Nil(err)

	err = stgf.Close()
	ast.Nil(err)
}
