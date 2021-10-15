package dao

import "github.com/willie68/GoBlobStore/pkg/model"

type RetentionManager interface {
	Init() error

	GetAllRetentions(tenant string, callback func(r model.RetentionEntry) bool) error
	AddRetention(tenant string, r *model.RetentionEntry) error
	DeleteRetention(tenant string, id string) error
	ResetRetention(tenant string, id string) error

	Close() error
}
