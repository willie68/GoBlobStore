package opscoor

import (
	"errors"

	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

// SingleNodeOpsCoor this coordinator is used in a single node. It coordinates different operation on the same id
// the following operation are defined
type SingleNodeOpsCoor struct {
}

var _ interfaces.OpsCoor = &SingleNodeOpsCoor{}

// defining different errors for this service
var (
	ErrNotImplemented = errors.New("not implemented")
)

// Prepare the operation on the id, this is a factory
func (s *SingleNodeOpsCoor) Prepare(op interfaces.OperationType, id string) (interfaces.Operation, error) {

	return nil, ErrNotImplemented
}
