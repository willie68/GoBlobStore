package interfaces

import "errors"

// OpsCoor interface for the operation coordinator
type OpsCoor interface {
	Prepare(op Operation, id string) (bool, error)
	Start(op Operation, id string) (bool, error)
	Stop(op Operation, id string) (bool, error)
}

// Operation Defining the different operations
type Operation struct {
	op string
}

// defining different operations
var (
	OpUnknown = Operation{op: ""}
	OpBackup  = Operation{op: "backup"}
	OpTntBck  = Operation{op: "tntbackup"}
	OpRestore = Operation{op: "restore"}
	OpCache   = Operation{op: "cache"}
)

// String convert to string
func (o *Operation) String() string {
	return o.op
}

// ErrOperationUnknown if a string can't be converted into an operation
var (
	ErrOperationUnknown = errors.New("operation is unknown")
)

// OpFromString converts a string into an operation
func OpFromString(s string) (Operation, error) {
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
