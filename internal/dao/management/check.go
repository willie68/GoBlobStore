package management

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils"
)

type CheckContext struct {
	TenantID string
	Cache    interfaces.BlobStorageDao
	Primary  interfaces.BlobStorageDao
	Backup   interfaces.BlobStorageDao
	cancel   bool
}

type CheckResultLine struct {
	ID            string
	Filename      string
	InCache       bool
	InBackup      bool
	PrimaryHashOK bool
	BackupHashOK  bool
	HasError      bool
	Messages      []string
}

// Admin functions

//CheckStorage checks the storage to find inconsistencies.
// It will write a audit file with a line for every blob in the storage, including name, hash, and state
func (c *CheckContext) CheckStorage() (string, error) {
	c.cancel = false
	file, err := ioutil.TempFile("", "check.*.json")
	if err != nil {
		return "", err
	}
	defer file.Close()
	log.Logger.Debugf("start checking tenant \"%s\", results in file: %s", c.TenantID, file.Name())
	file.WriteString(fmt.Sprintf("{ \"Tenant\" : \"%s\"", c.TenantID))

	// checking all blobs in cache
	if c.Cache != nil {
		count := 0
		log.Logger.Debug("checking cache")
		file.WriteString(",\r\n\"Cache\": [")
		err := c.Cache.GetBlobs(func(id string) bool {
			// checking if the blob belongs to the tenant
			b, err := c.Cache.GetBlobDescription(id)
			if (err == nil) && (b.TenantID == c.TenantID) {
				msg := "ok"
				ip, _ := c.Primary.HasBlob(id)
				if !ip {
					msg = "cache inconsistent"
				}
				if count > 0 {
					file.WriteString(",\r\n")
				}
				file.WriteString(fmt.Sprintf("{\"ID\": \"%s\", \"HasError\": %t, \"Messages\": [\"%s\"]}", id, !ip, msg))
				count++
			}
			return true
		})

		file.WriteString("]")
		if err != nil {
			log.Logger.Errorf("check: error checking cache. %v", err)
		}
		file.WriteString(fmt.Sprintf(",\r\n\"CacheCount\": %d", count))
	}
	// checking all blobs in main storage
	count := 0
	log.Logger.Debug("checking primary")
	file.WriteString(",\r\n\"Primary\": [\r\n")
	err = c.Primary.GetBlobs(func(id string) bool {
		if count > 0 {
			file.WriteString(",\r\n")
		}
		c.checkBlob(id, file)
		count++
		return true
	})
	file.WriteString("]")
	file.WriteString(fmt.Sprintf(",\r\n\"PrimaryCount\": %d", count))
	if err != nil {
		log.Logger.Errorf("check: error checking primary. %v", err)
	}
	// checking all blobs in backup storage
	if c.Backup != nil {
		log.Logger.Debug("checking backup")
		count := 0
		first := true
		file.WriteString(",\r\n\"Backup\": [\r\n")
		err := c.Backup.GetBlobs(func(id string) bool {
			// only check blobs that are not already checked in primary
			if ok, _ := c.Primary.HasBlob(id); !ok {
				if !first {
					file.WriteString(",\r\n")
				}
				file.WriteString(fmt.Sprintf("{\"ID\": \"%s\", \"HasError\": true }", id))
				first = false
			}
			count++
			return true
		})
		if err != nil {
			log.Logger.Errorf("check: error checking backup. %v", err)
		}
		file.WriteString("]")
		file.WriteString(fmt.Sprintf(",\r\n\"BackupCount\": %d", count))
	}
	file.WriteString("\r\n}")
	return file.Name(), err
}

func (c *CheckContext) checkBlob(id string, file *os.File) {
	r := newResult()
	r.ID = id
	bd, err := c.Primary.GetBlobDescription(id)
	if err != nil {
		r.Messages = append(r.Messages, err.Error())
		r.HasError = true
	}
	// getting the filename
	r.Filename = bd.Filename
	// checking if this blob is chached
	if c.Cache != nil {
		r.InCache, _ = c.Cache.HasBlob(id)
	}
	// checking if this blob is backuped
	if c.Backup != nil {
		r.InBackup, _ = c.Backup.HasBlob(id)
		if !r.InBackup {
			r.Messages = append(r.Messages, "missing blob in backup")
			r.HasError = true
		}
	}
	// checking the hash of the primary blob
	hash, err := utils.BuildHash(id, c.Primary)
	if err != nil {
		r.Messages = append(r.Messages, fmt.Sprintf("error building hash: %v", err))
	}
	if hash != bd.Hash {
		r.PrimaryHashOK = false
		r.Messages = append(r.Messages, "primary hash not correct")
		r.HasError = true
	}
	// checking the hash of the backup blob
	if r.InBackup {
		hash, err = utils.BuildHash(id, c.Backup)
		if err != nil {
			r.Messages = append(r.Messages, fmt.Sprintf("error building hash on backup: %v", err))
		}
		if hash != bd.Hash {
			r.BackupHashOK = false
			r.Messages = append(r.Messages, "backup hash not correct")
			r.HasError = true
		}
	}

	if len(r.Messages) == 0 {
		r.Messages = append(r.Messages, "ok")
	}
	// writing a line of check results
	js, _ := json.Marshal(r)
	file.WriteString(string(js))
}

func newResult() CheckResultLine {
	return CheckResultLine{
		HasError:      false,
		InCache:       false,
		InBackup:      false,
		PrimaryHashOK: true,
		BackupHashOK:  true,
		Messages:      make([]string, 0),
	}
}
