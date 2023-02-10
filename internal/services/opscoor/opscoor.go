package opscoor

import (
	"errors"

	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

// SingleNodeOpsCoor this coordinator is used in a single node. It coordinates different operation on the same id
// the following operation are defined
type SingleNodeOpsCoor struct {
}

type DefaultOperation struct {
	coor interfaces.OpsCoor
	opt  interfaces.OperationType
	id   string
}

type BackupOperation struct {
	DefaultOperation
}

type TntBackupOperation struct {
	DefaultOperation
}

type RestoreOperation struct {
	DefaultOperation
}

type CacheOperation struct {
	DefaultOperation
}

var _ interfaces.Operation = &DefaultOperation{}
var _ interfaces.Operation = &BackupOperation{}
var _ interfaces.OpsCoor = &SingleNodeOpsCoor{}

// defining different errors for this service
var (
	ErrNotImplemented = errors.New("not implemented")
)

// Prepare the operation on the id, this is a factory
func (s *SingleNodeOpsCoor) Prepare(opt interfaces.OperationType, id string) (interfaces.Operation, error) {
	var op interfaces.Operation
	switch opt {
	case interfaces.OpBackup:
		op = &BackupOperation{}
	case interfaces.OpTntBck:
		op = &TntBackupOperation{}
	case interfaces.OpRestore:
		op = &RestoreOperation{}
	case interfaces.OpCache:
		op = &CacheOperation{}
	default:
		op = &DefaultOperation{}
	}
	err := op.Init(s, id)
	if err != nil {
		return nil, err
	}
	return op, nil
}

// Init initialize the default operation
func (o *DefaultOperation) Init(coor interfaces.OpsCoor, id string) error {
	o.opt = interfaces.OpUnknown
	o.id = id
	o.coor = coor
	return nil
}

// Start initialize the default operation
func (o *DefaultOperation) Start() (bool, error) {
	return true, ErrNotImplemented
}

// Stop initialize the default operation
func (o *DefaultOperation) Stop() (bool, error) {
	return true, ErrNotImplemented
}

// Init initialize the backup operation
func (o *BackupOperation) Init(coor interfaces.OpsCoor, id string) error {
	err := o.DefaultOperation.Init(coor, id)
	if err != nil {
		return err
	}
	o.opt = interfaces.OpBackup
	return nil
}

// Init initialize the backup operation
func (o *TntBackupOperation) Init(coor interfaces.OpsCoor, id string) error {
	err := o.DefaultOperation.Init(coor, id)
	if err != nil {
		return err
	}
	o.opt = interfaces.OpTntBck
	return nil
}

// Init initialize the backup operation
func (o *RestoreOperation) Init(coor interfaces.OpsCoor, id string) error {
	err := o.DefaultOperation.Init(coor, id)
	if err != nil {
		return err
	}
	o.opt = interfaces.OpRestore
	return nil
}

// Init initialize the backup operation
func (o *CacheOperation) Init(coor interfaces.OpsCoor, id string) error {
	err := o.DefaultOperation.Init(coor, id)
	if err != nil {
		return err
	}
	o.opt = interfaces.OpCache
	return nil
}
