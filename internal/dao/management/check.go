package management

import (
	"fmt"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
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
		ok := true
		msgs := make([]string, 0)
		msgs = append(msgs, fmt.Sprintf("%s:", id))

		dsc, err := c.primary.GetBlobDescription(id)
		if err != nil {
			msg := "%s: can't read blobdescription"
			log.Logger.Errorf(msg, id)
			msgs = append(msgs, "can't read blobdescription")
			ok = false
		}

		msgs, ok = c.checkPrimary(*dsc, msgs)
		if c.backup != nil {
			msgs, ok = c.checkBackup(*dsc, msgs)

		}
		if !ok {
			log.Logger.Errorf("errors: %v", msgs)
		}
		return true
	})
	return err
}

func (c CheckContext) checkPrimary(dsc model.BlobDescription, msgs []string) ([]string, bool) {
	ok := true
	id := dsc.BlobID
	hash, err := utils.BuildHash(id, c.primary)
	if err != nil {
		msg := "%s: error building hash, %v"
		log.Logger.Errorf(msg, id, err)
		msgs = append(msgs, fmt.Sprintf("error building hash: %v", err))
		ok = false
	}
	if dsc.Hash == "" {
		msgs = append(msgs, "missing hash on primary")
		dsc.Hash = hash
		err := c.primary.UpdateBlobDescription(id, &dsc)
		if err != nil {
			log.Logger.Errorf("error updating blob description on primary. \r\n%v", err)
			msgs = append(msgs, fmt.Sprintf("error updating blob description on primary: %v", err))
			ok = false
		}
	}
	if dsc.Hash != hash {
		msg := "%s: wrong hash"
		log.Logger.Errorf(msg, id)
		msgs = append(msgs, "wrong hash")
		ok = false
	}
	return msgs, ok
}

func (c CheckContext) checkBackup(bd model.BlobDescription, msgs []string) ([]string, bool) {
	ok := true
	id := bd.BlobID
	hash := bd.Hash

	dsc, err := c.backup.GetBlobDescription(id)
	if err != nil {
		msg := "%s: bck: can't read blobdescription"
		log.Logger.Errorf(msg, id)
		msgs = append(msgs, "bck: can't read blobdescription")
		ok = false
	}

	bckhash, err := utils.BuildHash(id, c.backup)
	if err != nil {
		msg := "%s: bck: error building hash, %v"
		log.Logger.Errorf(msg, id, err)
		msgs = append(msgs, fmt.Sprintf("bck: error building hash: %v", err))
		ok = false
	} else {
		if bckhash != hash {
			msg := "%s: bck: wrong hash"
			log.Logger.Errorf(msg, id)
			msgs = append(msgs, "bck: wrong hash")
			ok = false
		}
	}

	if dsc.Hash == "" {
		msgs = append(msgs, "missing hash on backup")
		err := c.backup.UpdateBlobDescription(id, &bd)
		if err != nil {
			log.Logger.Errorf("error updating blob description on backup. \r\n%v", err)
			msgs = append(msgs, fmt.Sprintf("error updating blob description on backup: %v", err))
			ok = false
		}
	}
	return msgs, ok
}
