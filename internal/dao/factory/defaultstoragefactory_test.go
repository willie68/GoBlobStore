package factory

import (
	"path/filepath"
	"testing"

	"bitbucket.easy.de/dm/service-blobstorage-go/internal/config"
	"bitbucket.easy.de/dm/service-blobstorage-go/internal/dao/interfaces"
	"bitbucket.easy.de/dm/service-blobstorage-go/internal/dao/noindex"
	"bitbucket.easy.de/dm/service-blobstorage-go/internal/dao/retentionmanager"
	"bitbucket.easy.de/dm/service-blobstorage-go/internal/dao/simplefile"
	"github.com/stretchr/testify/assert"
)

const (
	rootFilePrefix = "../../../testdata/fac"
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

	err = stgf.Close()
	ast.Nil(err)
}
