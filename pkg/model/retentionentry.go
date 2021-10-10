package model

type RetentionEntry struct {
	Filename      string `yaml:"filename" json:"filename"`
	TenantID      string `yaml:"tenantID" json:"tenantID"`
	BlobID        string `yaml:"blobID" json:"blobID"`
	CreationDate  int    `yaml:"creationDate" json:"creationDate"`
	Retention     int64  `yaml:"retention" json:"retention"`
	RetentionBase int    `yaml:"retentionBase" json:"retentionBase"`
}
