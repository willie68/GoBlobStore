package management

import (
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils"
)

type CheckContext struct {
	TenantID string
	primary  interfaces.BlobStorageDao
	backup   interfaces.BlobStorageDao
	cancel   bool
}

func (c CheckContext) Check() error {
	c.cancel = false
	err := c.primary.GetBlobs(func(id string) bool {
		if c.cancel {
			return false
		}
		dsc, err := c.primary.GetBlobDescription(id)
		if err != nil {
			msg := "%s: can't read blobdescription"
			log.Logger.Errorf(msg, id)
		}
		hash, err := utils.BuildHash(id, c.primary)
		if err != nil {
			msg := "%s: can't read blobdescription"
			log.Logger.Errorf(msg, id)
		}
		if dsc.Hash == "" {
			dsc.Hash = hash
			err := c.primary.UpdateBlobDescription(id, dsc)
			log.Logger.Errorf("error updating blob description on primary. \r\n%v", err)
		}
		return true
	})
	return err
}
