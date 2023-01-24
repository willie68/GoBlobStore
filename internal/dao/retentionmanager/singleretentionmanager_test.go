package retentionmanager

import (
	"path/filepath"
	"testing"

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

type NoStorage struct{}

func TestInit(t *testing.T) {
	ast := assert.New(t)
	tntMgr := &simplefile.TenantManager{
		RootPath: rootFilePrefix,
	}
	err := tntMgr.Init()
	ast.Nil(err)

	srm := SingleRetentionManager{
		TntDao: tntMgr,
	}
	ast.NotNil(srm)
	stgf := &NoStorage{}

	err = srm.Init(stgf)
	ast.Nil(err)

}
