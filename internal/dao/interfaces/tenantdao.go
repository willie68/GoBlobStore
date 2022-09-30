package interfaces

import "github.com/willie68/GoBlobStore/internal/config"

type TenantConfig struct {
	Backup     config.Storage `yaml:"backup" json:"backup"`
	Properties map[string]any `yaml:"properties" json:"properties"`
}

// TenantDao is the part of the daos which will adminitrate the tenant part of a storage system
type TenantDao interface {
	Init() error // initialise this dao

	GetTenants(callback func(tenant string) bool) error // walk thru all configured tenants and get the id back

	AddTenant(tenant string) error              // adding a new tenant
	RemoveTenant(tenant string) (string, error) // removing a tenant, deleting all data async, return the processid for this
	HasTenant(tenant string) bool               // checking if a tenant is present

	SetConfig(tenant string, config TenantConfig) error // setting a new config object
	GetConfig(tenant string) (*TenantConfig, error)     // getting the config object

	GetSize(tenant string) int64 // getting the overall storage size for this tenant, if tenant not present -1 is returned

	Close() error // closing the dao
}
