package interfaces

import "errors"

// OpsCoor interface for the operation coordinator
type OpsCoor interface {
	Prepare(op OperationType, id string) (Operation, error)
}

// Operation this is the interface for a single operation
type Operation interface {
	Init(id string) error
	Start() (bool, error)
	Stop() (bool, error)
}

// OperationType Defining the different operations
type OperationType struct {
	op string
}

// defining different operations
var (
	OpUnknown = OperationType{op: ""}
	OpBackup  = OperationType{op: "backup"}
	OpTntBck  = OperationType{op: "tntbackup"}
	OpRestore = OperationType{op: "restore"}
	OpCache   = OperationType{op: "cache"}
)

// String convert to string
func (o *OperationType) String() string {
	return o.op
}

// ErrOperationUnknown if a string can't be converted into an operation
var (
	ErrOperationUnknown = errors.New("operation is unknown")
)

// OpFromString converts a string into an operation
func OpFromString(s string) (OperationType, error) {
	switch s {
	case OpBackup.op:
		return OpBackup, nil
	case OpTntBck.op:
		return OpTntBck, nil
	case OpRestore.op:
		return OpRestore, nil
	}
	return OpUnknown, ErrOperationUnknown
}
