package management

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

type CheckManagement struct {
	StorageFactory interfaces.StorageFactory
	cCtxs          map[string]*CheckContext
}

type CheckResult struct {
	CheckID   string
	Startet   time.Time
	Finnished time.Time
	Running   bool
	BlobID    string
}

// management functions
// createStorage creating a new storage dao for the tenant depending on the configuration
func (c *CheckManagement) Init() error {
	c.cCtxs = make(map[string]*CheckContext)
	return nil
}

func (c *CheckManagement) IsRunning(tenant string) bool {
	if c, ok := c.cCtxs[tenant]; ok {
		if c.Running {
			return true
		}
	}
	return false
}

func (c *CheckManagement) GetResult(tenant string) (CheckResult, error) {
	if c, ok := c.cCtxs[tenant]; ok {
		res := CheckResult{
			CheckID:   c.CheckID,
			Running:   c.Running,
			Startet:   c.Started,
			Finnished: c.Finnished,
		}
		if !c.Running {
			res.BlobID = c.BlobID
		}
		return res, nil
	}
	return CheckResult{}, errors.New("no check running for tenant")
}

func (c *CheckManagement) Start(tenant string) (string, error) {
	if c, ok := c.cCtxs[tenant]; ok {
		if c.Running {
			return "", errors.New("check already running for tenant")
		}
	}
	cCtx, err := c.getCheckDao(tenant)
	if err != nil {
		return "", err
	}
	c.cCtxs[tenant] = cCtx
	go c.doCheck(cCtx)
	return cCtx.CheckID, nil
}

func (c *CheckManagement) doCheck(cCtx *CheckContext) {
	cCtx.Started = time.Now()
	defer func() { cCtx.Finnished = time.Now() }()
	file, err := cCtx.CheckStorage()
	if err != nil {
		cCtx.Message = fmt.Sprintf("error checking tenant %s: %v", cCtx.TenantID, err)
		return
	}
	d, err := c.StorageFactory.GetStorageDao(cCtx.TenantID)
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
	cCtx.BlobID = id
	return
}

func (c *CheckManagement) getCheckDao(tenant string) (*CheckContext, error) {
	if c, ok := c.cCtxs[tenant]; ok {
		return c, nil
	}
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

func (c *CheckManagement) Close() error {
	return nil
}
