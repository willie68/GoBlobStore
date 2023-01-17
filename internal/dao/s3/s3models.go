package s3

// S3StoreEntry entry in a list for tenant management
type S3StoreEntry struct {
	Tenant string `yaml:"tenant" json:"tenant"`
}
