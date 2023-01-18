package s3

// StoreEntry entry in a list for tenant management
type StoreEntry struct {
	Tenant string `yaml:"tenant" json:"tenant"`
}
