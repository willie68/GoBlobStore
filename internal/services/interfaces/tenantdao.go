package interfaces

import "github.com/willie68/GoBlobStore/internal/config"

// TenantConfig config for the tenant
type TenantConfig struct {
	Backup     config.Storage `yaml:"backup" json:"backup"`
	Properties map[string]any `yaml:"properties" json:"properties"`
}

// TenantManager is the part of the service which will administrate the tenant part of a storage system
type TenantManager interface {
	Init() error // initialize this service

	GetTenants(callback func(tenant string) bool) error // walk thru all configured tenants and get the id back

	AddTenant(tenant string) error              // adding a new tenant
	RemoveTenant(tenant string) (string, error) // removing a tenant, deleting all data async, return the processid for this
	HasTenant(tenant string) bool               // checking if a tenant is present

	SetConfig(tenant string, cnfg TenantConfig) error // setting a new config object
	GetConfig(tenant string) (*TenantConfig, error)   // getting the config object

	GetSize(tenant string) int64       // getting the overall storage size for this tenant, if tenant not present -1 is returned
	AddSize(tenant string, size int64) // adding the blob size to the tenant storage
	SubSize(tenant string, size int64) // adding the blob size to the tenant storage
	Close() error                      // closing the service
}
