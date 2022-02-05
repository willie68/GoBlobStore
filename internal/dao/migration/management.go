package migration

import (
	"errors"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/utils"
)

type RestoreManagement struct {
	StorageFactory interfaces.StorageFactory
	cCtxs          map[string]*RestoreContext
}

type RestoreResult struct {
	ID        string
	Startet   time.Time
	Finnished time.Time
	Running   bool
	BlobID    string
}

// management functions
// createStorage creating a new storage dao for the tenant depending on the configuration
func (c *RestoreManagement) Init() error {
	c.cCtxs = make(map[string]*RestoreContext)
	return nil
}

func (c *RestoreManagement) IsRunning(tenant string) bool {
	if c, ok := c.cCtxs[tenant]; ok {
		if c.Running {
			return true
		}
	}
	return false
}

func (c *RestoreManagement) GetResult(tenant string) (RestoreResult, error) {
	if c, ok := c.cCtxs[tenant]; ok {
		res := RestoreResult{
			ID:        c.ID,
			Running:   c.Running,
			Startet:   c.Started,
			Finnished: c.Finnished,
		}
		return res, nil
	}
	return RestoreResult{}, errors.New("no restore running for tenant")
}

func (c *RestoreManagement) Start(tenant string) (string, error) {
	if c, ok := c.cCtxs[tenant]; ok {
		if c.Running {
			return "", errors.New("restore already running for tenant")
		}
	}
	cCtx, err := c.getRestoreDao(tenant)
	if err != nil {
		return "", err
	}
	c.cCtxs[tenant] = cCtx
	go c.doRestore(cCtx)
	return cCtx.ID, nil
}

func (c *RestoreManagement) doRestore(cCtx *RestoreContext) {
	cCtx.Started = time.Now()
	defer func() { cCtx.Finnished = time.Now() }()
	cCtx.Restore()
	return
}

func (c *RestoreManagement) getRestoreDao(tenant string) (*RestoreContext, error) {
	if c, ok := c.cCtxs[tenant]; ok {
		return c, nil
	}
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

func (c *RestoreManagement) Close() error {
	return nil
}
