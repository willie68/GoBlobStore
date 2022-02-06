package migration

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

type MigrationManagement struct {
	StorageFactory interfaces.StorageFactory
	cCtxs          map[string]interface{}
}

type MigrationResult struct {
	ID        string
	Startet   time.Time
	Finnished time.Time
	Running   bool
	BlobID    string
	Command   string
}

// management functions
// createStorage creating a new storage dao for the tenant depending on the configuration
func (c *MigrationManagement) Init() error {
	c.cCtxs = make(map[string]interface{})
	return nil
}

func (c *MigrationManagement) IsRunning(tenant string) bool {
	if it, ok := c.cCtxs[tenant]; ok {
		if r, ok := it.(interfaces.Running); ok {
			if r.IsRunning() {
				return true
			}
		}
	}
	return false
}

func (c *MigrationManagement) GetResult(tenant string) (MigrationResult, error) {
	if i, ok := c.cCtxs[tenant]; ok {
		switch v := i.(type) {
		case *CheckContext:
			res := MigrationResult{
				ID:        v.CheckID,
				Running:   v.Running,
				Startet:   v.Started,
				Finnished: v.Finnished,
				Command:   "Check",
			}
			return res, nil
		case *RestoreContext:
			res := MigrationResult{
				ID:        v.ID,
				Running:   v.Running,
				Startet:   v.Started,
				Finnished: v.Finnished,
				Command:   "Restore",
			}
			return res, nil
		}
	}
	return MigrationResult{}, errors.New("no process running for tenant")
}

func (c *MigrationManagement) StartRestore(tenant string) (string, error) {
	if c.IsRunning(tenant) {
		return "", errors.New("process already running for tenant")
	}

	cCtx, err := c.getRestoreDao(tenant)
	if err != nil {
		return "", err
	}
	c.cCtxs[tenant] = cCtx
	cCtx.Running = true
	go c.doRestore(cCtx)
	return cCtx.ID, nil
}

func (c *MigrationManagement) doRestore(cCtx *RestoreContext) {
	cCtx.Started = time.Now()
	defer func() { cCtx.Finnished = time.Now() }()
	cCtx.Restore()
}

func (c *MigrationManagement) getRestoreDao(tenant string) (*RestoreContext, error) {
	d, err := c.StorageFactory.GetStorageDao(tenant)
	if err != nil {
		return nil, err
	}
	main, ok := d.(*business.MainStorageDao)
	if !ok {
		return nil, errors.New("wrong storage class for restore")
	}
	uuid := utils.GenerateID()
	cCtx := RestoreContext{
		TenantID: tenant,
		ID:       uuid,
		Primary:  main.StgDao,
		Backup:   main.BckDao,
		Running:  false,
	}
	return &cCtx, nil
}

func (c *MigrationManagement) Close() error {
	return nil
}

// management functions

func (m *MigrationManagement) StartCheck(tenant string) (string, error) {
	if m.IsRunning(tenant) {
		return "", errors.New("process already running for tenant")
	}
	cCtx, err := m.getCheckDao(tenant)
	if err != nil {
		return "", err
	}
	m.cCtxs[tenant] = cCtx
	cCtx.Running = true
	go m.doCheck(cCtx)
	return cCtx.CheckID, nil
}

func (m *MigrationManagement) doCheck(cCtx *CheckContext) {
	cCtx.Started = time.Now()
	defer func() { cCtx.Finnished = time.Now() }()
	file, err := cCtx.CheckStorage()
	if err != nil {
		cCtx.Message = fmt.Sprintf("error checking tenant %s: %v", cCtx.TenantID, err)
		return
	}
	d, err := m.StorageFactory.GetStorageDao(cCtx.TenantID)
	if err != nil {
		cCtx.Message = fmt.Sprintf("error getting storage for tenant %s: %v", cCtx.TenantID, err)
		return
	}
	s, err := os.Stat(file)
	if err != nil {
		cCtx.Message = fmt.Sprintf("error getting check file for tenant %s, %s: %v", cCtx.TenantID, file, err)
		return
	}
	b := model.BlobDescription{
		ContentType:   "application/json",
		ContentLength: s.Size(),
		Filename:      file,
	}
	r, err := os.Open(file)
	if err != nil {
		cCtx.Message = fmt.Sprintf("error getting check file for tenant %s, %s: %v", cCtx.TenantID, file, err)
		return
	}
	defer r.Close()
	id, err := d.StoreBlob(&b, r)
	if err != nil {
		cCtx.Message = fmt.Sprintf("couldn't store check file for tenant %s, %s: %v", cCtx.TenantID, file, err)
		return
	}
	cCtx.BlobID = id
}

func (c *MigrationManagement) getCheckDao(tenant string) (*CheckContext, error) {
	d, err := c.StorageFactory.GetStorageDao(tenant)
	if err != nil {
		return nil, err
	}
	main, ok := d.(*business.MainStorageDao)
	if !ok {
		return nil, errors.New("wrong storage class for check")
	}
	uuid := utils.GenerateID()
	cCtx := CheckContext{
		TenantID: tenant,
		CheckID:  uuid,
		Cache:    main.CchDao,
		Primary:  main.StgDao,
		Backup:   main.BckDao,
		Running:  false,
	}
	return &cCtx, nil
}
