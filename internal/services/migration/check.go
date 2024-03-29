package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/internal/utils"
)

// CheckContext struct for the running check
type CheckContext struct {
	TenantID string
	CheckID  string
	Started  time.Time
	Finished time.Time
	Cache    interfaces.BlobStorage
	Primary  interfaces.BlobStorage
	Backup   interfaces.BlobStorage
	Running  bool
	Filename string
	BlobID   string
	cancel   bool
	Message  string
}

// checking interface compatibility
var _ interfaces.Running = &CheckContext{}

// CheckResultLine on entry for the result of the check, usually converted into one report output line
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

// CheckStorage checks the storage to find inconsistencies.
// It will write a audit file with a line for every blob in the storage, including name, hash, and state
func (c *CheckContext) CheckStorage() (string, error) {
	c.Running = true
	defer func() {
		c.Running = false
	}()
	c.cancel = false
	file, err := os.CreateTemp("", "check.*.json")
	if err != nil {
		return "", err
	}
	defer file.Close()
	c.Filename = file.Name()
	logger.Debugf("start checking tenant \"%s\", results in file: %s", c.TenantID, file.Name())
	_, _ = file.WriteString(fmt.Sprintf("{ \"Tenant\" : \"%s\"", c.TenantID))

	// checking all blobs in cache
	if c.Cache != nil {
		c.checkCache(file)
	}
	// checking all blobs in main storage
	count := 0
	logger.Debug("checking primary")
	_, _ = file.WriteString(",\r\n\"Primary\": [\r\n")
	err = c.Primary.GetBlobs(func(id string) bool {
		if count > 0 {
			_, _ = file.WriteString(",\r\n")
		}
		c.checkBlob(id, file)
		count++
		return true
	})
	_, _ = file.WriteString("]")
	_, _ = file.WriteString(fmt.Sprintf(",\r\n\"PrimaryCount\": %d", count))
	if err != nil {
		logger.Errorf("check: error checking primary. %v", err)
	}
	// checking all blobs in backup storage
	if c.Backup != nil {
		c.checkBackup(file)
	}
	_, _ = file.WriteString("\r\n}")
	return file.Name(), err
}

func (c *CheckContext) checkBackup(file *os.File) {
	logger.Debug("checking backup")
	count := 0
	first := true
	_, _ = file.WriteString(",\r\n\"Backup\": [\r\n")
	err := c.Backup.GetBlobs(func(id string) bool {
		// only check blobs that are not already checked in primary
		if ok, _ := c.Primary.HasBlob(id); !ok {
			if !first {
				_, _ = file.WriteString(",\r\n")
			}
			_, _ = file.WriteString(fmt.Sprintf("{\"ID\": \"%s\", \"HasError\": true }", id))
			first = false
		}
		count++
		return true
	})
	if err != nil {
		logger.Errorf("check: error checking backup. %v", err)
	}
	_, _ = file.WriteString("]")
	_, _ = file.WriteString(fmt.Sprintf(",\r\n\"BackupCount\": %d", count))
}

func (c *CheckContext) checkCache(file *os.File) {
	count := 0
	logger.Debug("checking cache")
	_, _ = file.WriteString(",\r\n\"Cache\": [")
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
				_, _ = file.WriteString(",\r\n")
			}
			_, _ = file.WriteString(fmt.Sprintf("{\"ID\": \"%s\", \"HasError\": %t, \"Messages\": [\"%s\"]}", id, !ip, msg))
			count++
		}
		return true
	})

	_, _ = file.WriteString("]")
	if err != nil {
		logger.Errorf("check: error checking cache. %v", err)
	}
	_, _ = file.WriteString(fmt.Sprintf(",\r\n\"CacheCount\": %d", count))
}

// IsRunning checking if this task is running
func (c *CheckContext) IsRunning() bool {
	return c.Running
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
	_, _ = file.WriteString(string(js))
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
