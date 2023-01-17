package model

// RetentionEntry antry for the retention of a blob
type RetentionEntry struct {
	Filename      string `yaml:"filename" json:"filename"`
	TenantID      string `yaml:"tenantID" json:"tenantID"`
	BlobID        string `yaml:"blobID" json:"blobID"`
	CreationDate  int64  `yaml:"creationDate" json:"creationDate"`
	Retention     int64  `yaml:"retention" json:"retention"`
	RetentionBase int64  `yaml:"retentionBase" json:"retentionBase"`
}

// GetRetentionTimestampMS getting the time stamp in ms
func (r *RetentionEntry) GetRetentionTimestampMS() int64 {
	if r.RetentionBase > 0 {
		return r.RetentionBase + r.Retention*60*1000
	}
	return r.CreationDate + r.Retention*60*1000
}

// RetentionEntryFromBlobDescription building a retention entry from a blobdescription
func RetentionEntryFromBlobDescription(b BlobDescription) RetentionEntry {
	return RetentionEntry{
		BlobID:        b.BlobID,
		CreationDate:  b.CreationDate,
		Filename:      b.Filename,
		Retention:     b.Retention,
		RetentionBase: 0,
		TenantID:      b.TenantID,
	}
}
