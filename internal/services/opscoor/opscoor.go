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

// Prepare the operation on the id
func (s *SingleNodeOpsCoor) Prepare(op interfaces.Operation, id string) (bool, error) {
	return false, ErrNotImplemented
}

// Start the operation
func (s *SingleNodeOpsCoor) Start(op interfaces.Operation, id string) (bool, error) {
	return false, ErrNotImplemented
}

// Stop the operation, remove this from the operation stack
func (s *SingleNodeOpsCoor) Stop(op interfaces.Operation, id string) (bool, error) {
	return false, ErrNotImplemented
}
