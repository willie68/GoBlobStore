package retentionmanager

import (
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/pkg/model"
)

var _ interfaces.RetentionManager = &NoRetention{}

// NoRetention do nothing retention manager
type NoRetention struct{}

// Init initialize this no retention manager
func (n *NoRetention) Init(_ interfaces.StorageFactory) error {
	return nil
}

// GetAllRetentions return nothing
func (n *NoRetention) GetAllRetentions(_ string, _ func(r model.RetentionEntry) bool) error {
	return nil
}

// AddRetention no adding possible
func (n *NoRetention) AddRetention(_ string, _ *model.RetentionEntry) error {
	return nil
}

// DeleteRetention nothing to delete here
func (n *NoRetention) DeleteRetention(_ string, _ string) error {
	return nil
}

// ResetRetention no reset possible
func (n *NoRetention) ResetRetention(_ string, _ string) error {
	return nil
}

// Close nothing to close
func (n *NoRetention) Close() error {
	return nil
}
