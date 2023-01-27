package business

import (
	"errors"
	"sync"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils/slicesutils"
)

// this type is doing all the stuff for managing different tenants in the system.
// It will use the underlying tenant services for the storage part.

// checking interface compatibility
var _ interfaces.TenantManager = &MainTenant{}

// MainTenant the business object for doing all tenant based operations
type MainTenant struct {
	TntDao  interfaces.TenantManager
	BckDao  interfaces.TenantManager
	hasBck  bool
	rmTnt   []string
	rmtSync sync.Mutex
}

// Init initialize this dao
func (m *MainTenant) Init() error {
	var err error
	if m.TntDao == nil {
		return errors.New("tenant dao should not be nil")
	}
	err = m.TntDao.Init()
	if m.BckDao != nil {
		err = m.BckDao.Init()
		m.hasBck = true
	}
	m.rmTnt = make([]string, 0)
	return err
}

// GetTenants walk thru all configured tenants and get the id back
func (m *MainTenant) GetTenants(callback func(tenant string) bool) error {
	return m.TntDao.GetTenants(func(t string) bool {
		if !slicesutils.Contains(m.rmTnt, t) {
			callback(t)
		}
		return true
	})
}

// AddTenant adding a new tenant
func (m *MainTenant) AddTenant(tenant string) error {
	if slicesutils.Contains(m.rmTnt, tenant) {
		return errors.New("can't add tenant, it's in removal state")
	}
	err := m.TntDao.AddTenant(tenant)
	if m.hasBck {
		err = m.BckDao.AddTenant(tenant)
	}
	return err
}

// RemoveTenant removing a tenant, deleting all data async, return the process id for this
func (m *MainTenant) RemoveTenant(tenant string) (string, error) {
	if slicesutils.Contains(m.rmTnt, tenant) {
		return "", errors.New("tenant is already in removal state")
	}
	if !m.HasTenant(tenant) {
		return "", errors.New("tenant not exists")
	}
	m.rmtSync.Lock()
	m.rmTnt = append(m.rmTnt, tenant)
	m.rmtSync.Unlock()
	go m.removeTnt(tenant)
	if m.hasBck {
		go m.BckDao.RemoveTenant(tenant)
	}
	return "", nil
}

func (m *MainTenant) removeTnt(tenant string) {
	_, err := m.TntDao.RemoveTenant(tenant)
	if err != nil {
		log.Logger.Errorf("error removing tenant %s: %v", tenant, err)
	}
	m.rmtSync.Lock()
	m.rmTnt = slicesutils.RemoveString(m.rmTnt, tenant)
	m.rmtSync.Unlock()
}

// HasTenant checking if a tenant is present
func (m *MainTenant) HasTenant(tenant string) bool {
	if slicesutils.Contains(m.rmTnt, tenant) {
		return false
	}
	return m.TntDao.HasTenant(tenant)
}

// SetConfig writing a new config object for the tenant
func (m *MainTenant) SetConfig(tenant string, config interfaces.TenantConfig) error {
	err := m.TntDao.SetConfig(tenant, config)
	if m.hasBck {
		m.BckDao.SetConfig(tenant, config)
	}
	return err
}

// GetConfig reading the config object for the tenant
func (m *MainTenant) GetConfig(tenant string) (*interfaces.TenantConfig, error) {
	cfn, err := m.TntDao.GetConfig(tenant)
	if err != nil {
		return nil, err
	}
	if (cfn == nil) && (m.BckDao != nil) {
		cfn, err = m.BckDao.GetConfig(tenant)
		if err != nil {
			log.Logger.Errorf("error reading config for tenant %s from backup. %v", tenant, err)
		}
	}
	return cfn, nil
}

// GetSize getting the overall storage size for this tenant
func (m *MainTenant) GetSize(tenant string) int64 {
	if slicesutils.Contains(m.rmTnt, tenant) {
		return -1
	}
	return m.TntDao.GetSize(tenant)
}

// Close closing the dao
func (m *MainTenant) Close() error {
	var err error
	if m.TntDao != nil {
		err = m.TntDao.Close()
	}
	if m.BckDao != nil {
		err = m.BckDao.Close()
	}
	return err
}
