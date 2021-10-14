package dao

import "github.com/willie68/GoBlobStore/pkg/model"

type RetentionManager interface {
	Init() error

	AddRetention(tenant string, b *model.BlobDescription) error

	Close() error
}
