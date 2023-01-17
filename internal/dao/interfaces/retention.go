package interfaces

import "github.com/willie68/GoBlobStore/pkg/model"

// RetentionManager interface for retention manager
type RetentionManager interface {
	Init(stgf StorageFactory) error

	GetAllRetentions(tenant string, callback func(r model.RetentionEntry) bool) error
	AddRetention(tenant string, r *model.RetentionEntry) error
	DeleteRetention(tenant string, id string) error
	ResetRetention(tenant string, id string) error

	Close() error
}
