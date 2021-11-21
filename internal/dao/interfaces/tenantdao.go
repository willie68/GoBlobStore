package interfaces

// TenantDao is the part of the daos which will adminitrate the tenant part of a storage system
type TenantDao interface {
	Init() error // initialise this dao

	GetTenants(callback func(tenant string) bool) error

	AddTenant(tenant string) error
	RemoveTenant(tenant string) error
	HasTenant(tenant string) bool
	GetSize(tenant string) int64

	Close() error
}
