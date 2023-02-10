package opscoor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

var snoc interfaces.OpsCoor

func initTest(t *testing.T) {
	ast := assert.New(t)

	snoc = &SingleNodeOpsCoor{}
	ast.NotNil(snoc)
}

func TestInstancingSN(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, err := snoc.Prepare(interfaces.OpUnknown, "12345678")
	ast.Nil(err)
	ast.NotNil(op)

	dop, ok := op.(*DefaultOperation)
	ast.True(ok)
	ast.NotNil(dop)
	ast.Equal("12345678", dop.id)
	ast.Equal(interfaces.OpUnknown, dop.opt)
}

func TestBackupOp(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, err := snoc.Prepare(interfaces.OpBackup, "12345678")
	ast.Nil(err)
	ast.NotNil(op)

	dop, ok := op.(*BackupOperation)
	ast.True(ok)
	ast.NotNil(dop)
	ast.Equal("12345678", dop.id)
	ast.Equal(interfaces.OpBackup, dop.opt)
}

func TestTntBackupOp(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, err := snoc.Prepare(interfaces.OpTntBck, "12345678")
	ast.Nil(err)
	ast.NotNil(op)

	dop, ok := op.(*TntBackupOperation)
	ast.True(ok)
	ast.NotNil(dop)
	ast.Equal("12345678", dop.id)
	ast.Equal(interfaces.OpTntBck, dop.opt)
}

func TestRestoreOp(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, err := snoc.Prepare(interfaces.OpRestore, "12345678")
	ast.Nil(err)
	ast.NotNil(op)

	dop, ok := op.(*RestoreOperation)
	ast.True(ok)
	ast.NotNil(dop)
	ast.Equal("12345678", dop.id)
	ast.Equal(interfaces.OpRestore, dop.opt)
}

func TestCacheOp(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, err := snoc.Prepare(interfaces.OpCache, "12345678")
	ast.Nil(err)
	ast.NotNil(op)

	dop, ok := op.(*CacheOperation)
	ast.True(ok)
	ast.NotNil(dop)
	ast.Equal("12345678", dop.id)
	ast.Equal(interfaces.OpCache, dop.opt)
}
