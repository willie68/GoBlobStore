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

// Management this dao takes control over several async migration parts, as backup, checks ...
type Management struct {
	StorageFactory interfaces.StorageFactory
	cCtxs          map[string]any
}

// Result is the result of a async migration task
type Result struct {
	ID        string
	Startet   time.Time
	Finnished time.Time
	Running   bool
	BlobID    string
	Command   string
}

// management functions

// Init creates a new migration service
func (m *Management) Init() error {
	m.cCtxs = make(map[string]any)
	return nil
}

// IsRunning checking if a migration task is running for a tenant
func (m *Management) IsRunning(tenant string) bool {
	if it, ok := m.cCtxs[tenant]; ok {
		if r, ok := it.(interfaces.Running); ok {
			if r.IsRunning() {
				return true
			}
		}
	}
	return false
}

// GetResult getting the result of the last migration task
func (m *Management) GetResult(tenant string) (Result, error) {
	if i, ok := m.cCtxs[tenant]; ok {
		switch v := i.(type) {
		case *CheckContext:
			res := Result{
				ID:        v.CheckID,
				Running:   v.Running,
				Startet:   v.Started,
				Finnished: v.Finished,
				Command:   "Check",
			}
			return res, nil
		case *RestoreContext:
			res := Result{
				ID:        v.ID,
				Running:   v.Running,
				Startet:   v.Started,
				Finnished: v.Finnished,
				Command:   "Restore",
			}
			return res, nil
		}
	}
	return Result{}, errors.New("no process running for tenant")
}

// StartRestore starting a restore task for a tenant
func (m *Management) StartRestore(tenant string) (string, error) {
	if m.IsRunning(tenant) {
		return "", errors.New("process already running for tenant")
	}

	cCtx, err := m.getRestoreDao(tenant)
	if err != nil {
		return "", err
	}
	m.cCtxs[tenant] = cCtx
	cCtx.Running = true
	go m.doRestore(cCtx)
	return cCtx.ID, nil
}

func (m *Management) doRestore(cCtx *RestoreContext) {
	cCtx.Started = time.Now()
	defer func() {
		cCtx.Finnished = time.Now()
	}()
	cCtx.Restore()
}

func (m *Management) getRestoreDao(tenant string) (*RestoreContext, error) {
	d, err := m.StorageFactory.GetStorage(tenant)
	if err != nil {
		return nil, err
	}
	main, ok := d.(*business.MainStorage)
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

// Close closing this service
func (m *Management) Close() error {
	return nil
}

// management functions

// StartCheck starting a check of all blob for a tenant
func (m *Management) StartCheck(tenant string) (string, error) {
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

func (m *Management) doCheck(cCtx *CheckContext) {
	cCtx.Started = time.Now()
	defer func() {
		cCtx.Finished = time.Now()
	}()
	file, err := cCtx.CheckStorage()
	if err != nil {
		cCtx.Message = fmt.Sprintf("error checking tenant %s: %v", cCtx.TenantID, err)
		return
	}
	d, err := m.StorageFactory.GetStorage(cCtx.TenantID)
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

func (m *Management) getCheckDao(tenant string) (*CheckContext, error) {
	d, err := m.StorageFactory.GetStorage(tenant)
	if err != nil {
		return nil, err
	}
	main, ok := d.(*business.MainStorage)
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
