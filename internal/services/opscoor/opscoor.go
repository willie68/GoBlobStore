package opscoor

import (
	"errors"
	"sync"

	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

// SingleNodeOpsCoor this coordinator is used in a single node. It coordinates different operation on the same id
// the following operation are defined
type SingleNodeOpsCoor struct {
	midx     sync.Mutex
	idx      int64
	entities sync.Map
}

type operations struct {
	sync.RWMutex
	items []DefaultOperation
}

type state struct {
	s string
}

var (
	sprep = state{s: "prepare"}
	srun  = state{s: "running"}
	sfin  = state{s: "finished"}
)

// DefaultOperation this is the default operation abstract class
type DefaultOperation struct {
	idx int64
	opt interfaces.OperationType
	id  string
	f   interfaces.Callback
	st  state
}

// interface checks
var (
	_ interfaces.Operation = &DefaultOperation{}
)

// defining different errors for this service
var (
	ErrNotImplemented = errors.New("not implemented")
)

func newOperations() *operations {
	cs := &operations{
		items: make([]DefaultOperation, 0),
	}
	return cs
}

// Append adds an item to the concurrent slice
func (cs *operations) Append(item DefaultOperation) {
	cs.Lock()
	defer cs.Unlock()

	cs.items = append(cs.items, item)
}

func (cs *operations) Len() int {
	cs.Lock()
	defer cs.Unlock()
	return len(cs.items)
}

func (cs *operations) Remove(op *DefaultOperation) bool {
	cs.Lock()
	defer cs.Unlock()
	i := -1
	for x, en := range cs.items {
		if en.idx == op.idx {
			i = x
			break
		}
	}
	if i > -1 {
		cs.items[i] = cs.items[len(cs.items)-1]
		cs.items = cs.items[:len(cs.items)-1]
		return true
	}
	return false
}

// NewSingleNodeOpsCoor return a OpsCoor for a single node system
func NewSingleNodeOpsCoor() *SingleNodeOpsCoor {
	snoc := &SingleNodeOpsCoor{}
	snoc.init()
	return snoc
}

func (s *SingleNodeOpsCoor) init() {
	s.midx = sync.Mutex{}
}

// Prepare the operation on the id, this is a factory
func (s *SingleNodeOpsCoor) Prepare(opt interfaces.OperationType, id string, f interfaces.Callback) (interfaces.Operation, bool, error) {
	var ops *operations
	ok := opt != interfaces.OpUnknown
	s.midx.Lock()
	defer s.midx.Unlock()
	s.idx++
	op := &DefaultOperation{
		idx: s.idx,
		opt: opt,
		id:  id,
		f:   f,
		st:  sprep,
	}
	entries, lok := s.entities.Load(id)
	if !lok {
		ops = newOperations()
		s.entities.Store(id, ops)
	} else {
		ops, _ = entries.(*operations)
	}
	ops.Append(*op)
	go func() {
		op.st = srun
		if op.f != nil {
			op.f(op)
		}
		ops.Remove(op)
		op.st = sfin
	}()
	return op, ok, nil
}

// Count operations of an id
func (s *SingleNodeOpsCoor) Count(id string) int {
	entries, ok := s.entities.Load(id)
	if !ok {
		return 0
	}
	ops, _ := entries.(*operations)
	return ops.Len()
}

// ID return the id of this operation
func (o *DefaultOperation) ID() string {
	return o.id
}

// Type return the id of this operation
func (o *DefaultOperation) Type() interfaces.OperationType {
	return o.opt
}

// Started return the id of this operation
func (o *DefaultOperation) Started() bool {
	return false
}

// Active return the id of this operation
func (o *DefaultOperation) Active() bool {
	return false
}

// Finished return the id of this operation
func (o *DefaultOperation) Finished() bool {
	return false
}
