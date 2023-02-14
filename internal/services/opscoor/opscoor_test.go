package opscoor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

var snoc interfaces.OpsCoor

func initTest(t *testing.T) {
	ast := assert.New(t)

	snoc = NewSingleNodeOpsCoor()
	ast.NotNil(snoc)
}

func TestInstancingSN(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, ok, err := snoc.Prepare(interfaces.OpUnknown, "12345678", nil)
	ast.Nil(err)
	ast.NotNil(op)
	ast.False(ok)
	ast.Equal("12345678", op.ID())

	dop, ok := op.(*DefaultOperation)
	ast.True(ok)
	ast.NotNil(dop)
	ast.Equal("12345678", dop.id)
	ast.Equal(interfaces.OpUnknown, dop.opt)
}

func TestBackupOp(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, ok, err := snoc.Prepare(interfaces.OpBackup, "12345678", nil)
	ast.Nil(err)
	ast.NotNil(op)
	ast.True(ok)
	ast.Equal("12345678", op.ID())
	ast.Equal(interfaces.OpBackup, op.Type())
}

func TestTntBackupOp(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, ok, err := snoc.Prepare(interfaces.OpTntBck, "12345678", nil)
	ast.Nil(err)
	ast.NotNil(op)
	ast.True(ok)
	ast.Equal("12345678", op.ID())
	ast.Equal(interfaces.OpTntBck, op.Type())
}

func TestRestoreOp(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, ok, err := snoc.Prepare(interfaces.OpRestore, "12345678", nil)
	ast.Nil(err)
	ast.NotNil(op)
	ast.True(ok)
	ast.Equal("12345678", op.ID())
	ast.Equal(interfaces.OpRestore, op.Type())
}

func TestCacheOp(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	op, ok, err := snoc.Prepare(interfaces.OpCache, "12345678", nil)
	ast.Nil(err)
	ast.NotNil(op)
	ast.True(ok)
	ast.Equal("12345678", op.ID())
	ast.Equal(interfaces.OpCache, op.Type())
}

func TestBackupSimple(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	var counter = 0
	op1, ok, err := snoc.Prepare(interfaces.OpBackup, "12345678", func(op interfaces.Operation) bool {
		t.Log("Hello")
		counter++
		return true
	})
	ast.Nil(err)
	ast.NotNil(op1)
	ast.True(ok)

	time.Sleep(1 * time.Second)
	ast.Equal(1, counter)
}

func TestCounter(t *testing.T) {
	ast := assert.New(t)
	initTest(t)
	var counter = 0
	op, ok, err := snoc.Prepare(interfaces.OpBackup, "12345678", func(op interfaces.Operation) bool {
		time.Sleep(1 * time.Second)
		t.Log("1")
		counter++
		return true
	})
	ast.Nil(err)
	ast.NotNil(op)
	ast.True(ok)

	op, ok, err = snoc.Prepare(interfaces.OpBackup, "12345678", func(op interfaces.Operation) bool {
		time.Sleep(1 * time.Second)
		t.Log("2")
		counter++
		return true
	})
	ast.Nil(err)
	ast.NotNil(op)
	ast.True(ok)

	ast.Equal(2, snoc.Count("12345678"))
	time.Sleep(2 * time.Second)

	ast.Equal(0, snoc.Count("12345678"))
	ast.Equal(2, counter)

}
