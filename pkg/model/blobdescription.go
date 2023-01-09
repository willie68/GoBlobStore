package model

import (
	"encoding/json"
	"time"
)

type BlobDescription struct {
	StoreID       string `yaml:"storeid" json:"storeid"`
	ContentLength int64  `yaml:"contentLength" json:"contentLength"`
	ContentType   string `yaml:"contentType" json:"contentType"`
	CreationDate  int64  `yaml:"creationDate" json:"creationDate"`
	Filename      string `yaml:"filename" json:"filename"`
	TenantID      string `yaml:"tenantID" json:"tenantID"`
	BlobID        string `yaml:"blobID" json:"blobID"`
	LastAccess    int64  `yaml:"lastAccess" json:"lastAccess"`
	Retention     int64  `yaml:"retention" json:"retention"`
	BlobURL       string `yaml:"blobUrl" json:"blobUrl"`
	Hash          string `yaml:"hash" json:"hash"`
	Check         *Check `yaml:"check,omitempty" json:"check,omitempty"`
	Properties    map[string]interface{}
}

type Check struct {
	Store   *CheckInfo `yaml:"store,omitempty" json:"store,omitempty"`
	Backup  *CheckInfo `yaml:"backup,omitempty" json:"backup,omitempty"`
	Healthy bool       `yaml:"healthy,omitempty" json:"healthy,omitempty"`
	Message string     `yaml:"message,omitempty" json:"message,omitempty"`
}

type CheckInfo struct {
	LastCheck *time.Time `yaml:"lastCheck,omitempty" json:"lastCheck,omitempty"`
	Healthy   bool       `yaml:"healthy,omitempty" json:"healthy,omitempty"`
	Message   string     `yaml:"message,omitempty" json:"message,omitempty"`
}

func (b BlobDescription) Map() map[string]interface{} {
	mymap := make(map[string]interface{})
	mymap["storeid"] = b.StoreID
	mymap["contentLength"] = b.ContentLength
	mymap["contentType"] = b.ContentType
	mymap["creationDate"] = b.CreationDate
	mymap["filename"] = b.Filename
	mymap["tenantID"] = b.TenantID
	mymap["blobID"] = b.BlobID
	mymap["blobUrl"] = b.BlobURL
	mymap["lastAccess"] = b.LastAccess
	mymap["retention"] = b.Retention
	mymap["hash"] = b.Hash
	if b.Check != nil {
		mymap["check"] = b.Check
	}
	for k, v := range b.Properties {
		mymap[k] = v
	}
	return mymap
}

func (b BlobDescription) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.Map())
}

func (b *BlobDescription) UnmarshalJSON(data []byte) error {
	blob := struct {
		StoreID       string `yaml:"storeid" json:"storeid"`
		ContentLength int64  `yaml:"contentLength" json:"contentLength"`
		ContentType   string `yaml:"contentType" json:"contentType"`
		CreationDate  int64  `yaml:"creationDate" json:"creationDate"`
		Filename      string `yaml:"filename" json:"filename"`
		TenantID      string `yaml:"tenantID" json:"tenantID"`
		BlobID        string `yaml:"blobID" json:"blobID"`
		BlobURL       string `yaml:"blobUrl" json:"blobUrl"`
		LastAccess    int64  `yaml:"lastAccess" json:"lastAccess"`
		Retention     int64  `yaml:"retention" json:"retention"`
		Hash          string `yaml:"hash" json:"hash"`
		Check         *Check `yaml:"check,omitempty" json:"check,omitempty"`
	}{}
	err := json.Unmarshal(data, &blob)
	if err != nil {
		return err
	}

	mymap := make(map[string]interface{})
	err = json.Unmarshal(data, &mymap)
	if err != nil {
		return err
	}
	delete(mymap, "storeid")
	delete(mymap, "contentLength")
	delete(mymap, "contentType")
	delete(mymap, "creationDate")
	delete(mymap, "filename")
	delete(mymap, "tenantID")
	delete(mymap, "blobID")
	delete(mymap, "blobUrl")
	delete(mymap, "lastAccess")
	delete(mymap, "retention")
	delete(mymap, "hash")
	delete(mymap, "check")

	b.BlobID = blob.BlobID
	b.BlobURL = blob.BlobURL
	b.ContentLength = blob.ContentLength
	b.ContentType = blob.ContentType
	b.CreationDate = blob.CreationDate
	b.Filename = blob.Filename
	b.LastAccess = blob.LastAccess
	b.Retention = blob.Retention
	b.StoreID = blob.StoreID
	b.TenantID = blob.TenantID
	b.Hash = blob.Hash
	if blob.Check != nil {
		b.Check = blob.Check
	}
	b.Properties = mymap
	return nil
}
