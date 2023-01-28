package business

import (
	"errors"
	"sync"

	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/internal/utils/slicesutils"
)

// this type is doing all the stuff for managing different tenants in the system.
// It will use the underlying tenant services for the storage part.

// checking interface compatibility
var _ interfaces.TenantManager = &MainTenant{}

// MainTenant the business object for doing all tenant based operations
type MainTenant struct {
	TntSrv  interfaces.TenantManager
	BckSrv  interfaces.TenantManager
	hasBck  bool
	rmTnt   []string
	rmtSync sync.Mutex
}

// Init initialize this service
func (m *MainTenant) Init() error {
	var err error
	if m.TntSrv == nil {
		return errors.New("tenant service should not be nil")
	}
	err = m.TntSrv.Init()
	if m.BckSrv != nil {
		err = m.BckSrv.Init()
		m.hasBck = true
	}
	m.rmTnt = make([]string, 0)
	return err
}

// GetTenants walk thru all configured tenants and get the id back
func (m *MainTenant) GetTenants(callback func(tenant string) bool) error {
	return m.TntSrv.GetTenants(func(t string) bool {
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
	err := m.TntSrv.AddTenant(tenant)
	if m.hasBck {
		err = m.BckSrv.AddTenant(tenant)
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
		go m.BckSrv.RemoveTenant(tenant)
	}
	return "", nil
}

func (m *MainTenant) removeTnt(tenant string) {
	_, err := m.TntSrv.RemoveTenant(tenant)
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
	return m.TntSrv.HasTenant(tenant)
}

// SetConfig writing a new config object for the tenant
func (m *MainTenant) SetConfig(tenant string, config interfaces.TenantConfig) error {
	err := m.TntSrv.SetConfig(tenant, config)
	if m.hasBck {
		m.BckSrv.SetConfig(tenant, config)
	}
	return err
}

// GetConfig reading the config object for the tenant
func (m *MainTenant) GetConfig(tenant string) (*interfaces.TenantConfig, error) {
	cfn, err := m.TntSrv.GetConfig(tenant)
	if err != nil {
		return nil, err
	}
	if (cfn == nil) && (m.BckSrv != nil) {
		cfn, err = m.BckSrv.GetConfig(tenant)
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
	return m.TntSrv.GetSize(tenant)
}

// Close closing the service
func (m *MainTenant) Close() error {
	var err error
	if m.TntSrv != nil {
		err = m.TntSrv.Close()
	}
	if m.BckSrv != nil {
		err = m.BckSrv.Close()
	}
	return err
}
