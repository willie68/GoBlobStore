package model

import (
	"encoding/json"
)

type BlobDescription struct {
	StoreID       string `yaml:"storeid" json:"storeid"`
	ContentLength int64  `yaml:"contentLength" json:"contentLength"`
	ContentType   string `yaml:"contentType" json:"contentType"`
	CreationDate  int    `yaml:"creationDate" json:"creationDate"`
	Filename      string `yaml:"filename" json:"filename"`
	TenantID      string `yaml:"tenantID" json:"tenantID"`
	BlobID        string `yaml:"blobID" json:"blobID"`
	LastAccess    int    `yaml:"lastAccess" json:"lastAccess"`
	Retention     int64  `yaml:"retention" json:"retention"`
	BlobURL       string `yaml:"blobUrl" json:"blobUrl"`
	Hash          string `yaml:"hash" json:"hash"`
	Properties    map[string]interface{}
}

func (b BlobDescription) MarshalJSON() ([]byte, error) {
	mymap := make(map[string]interface{})
	mymap["storeid"] = b.StoreID
	mymap["contentLength"] = b.ContentLength
	mymap["contentType"] = b.ContentType
	mymap["creationDate"] = b.CreationDate
	mymap["filename"] = b.Filename
	mymap["tenantID"] = b.TenantID
	mymap["blobID"] = b.BlobID
	mymap["lastAccess"] = b.LastAccess
	mymap["retention"] = b.Retention
	mymap["hash"] = b.Hash
	for k, v := range b.Properties {
		mymap[k] = v
	}
	return json.Marshal(mymap)
}

func (b *BlobDescription) UnmarshalJSON(data []byte) error {
	blob := struct {
		StoreID       string `yaml:"storeid" json:"storeid"`
		ContentLength int64  `yaml:"contentLength" json:"contentLength"`
		ContentType   string `yaml:"contentType" json:"contentType"`
		CreationDate  int    `yaml:"creationDate" json:"creationDate"`
		Filename      string `yaml:"filename" json:"filename"`
		TenantID      string `yaml:"tenantID" json:"tenantID"`
		BlobID        string `yaml:"blobID" json:"blobID"`
		LastAccess    int    `yaml:"lastAccess" json:"lastAccess"`
		Retention     int64  `yaml:"retention" json:"retention"`
		Hash          string `yaml:"hash" json:"hash"`
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
	delete(mymap, "lastAccess")
	delete(mymap, "retention")
	delete(mymap, "hash")

	b.BlobID = blob.BlobID
	b.ContentLength = blob.ContentLength
	b.ContentType = blob.ContentType
	b.CreationDate = blob.CreationDate
	b.Filename = blob.Filename
	b.LastAccess = blob.LastAccess
	b.Retention = blob.Retention
	b.StoreID = blob.StoreID
	b.TenantID = blob.TenantID
	b.Hash = blob.Hash
	b.Properties = mymap
	return nil
}
