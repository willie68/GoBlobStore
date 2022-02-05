package management

import (
	"errors"

	"github.com/willie68/GoBlobStore/internal/dao/business"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
)

type CheckManagement struct {
	StorageFactory interfaces.StorageFactory
	cCtxs          map[string]*CheckContext
	checks         map[string]string
}

// management functions
// createStorage creating a new storage dao for the tenant depending on the configuration
func (c *CheckManagement) Init() error {
	c.cCtxs = make(map[string]*CheckContext)
	return nil
}

func (c *CheckManagement) StartCheck(tenant string) (string, error) {
	if c, ok := c.cCtxs[tenant]; ok {
		if c.Running {
			return "", errors.New("check already running for tenant")
		}
	}

	return id, nil
}

func (c *CheckManagement) GetCheckDao(tenant string) (*management.CheckContext, error) {
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
	cCtx := CheckContext{
		TenantID: tenant,
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
