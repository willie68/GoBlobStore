package model

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParsing(t *testing.T) {
	blobDescription := BlobDescription{
		StoreID:       "StoreID",
		ContentLength: 12345,
		ContentType:   "application/pdf",
		CreationDate:  int(time.Now().UnixNano() / 1000000),
		Filename:      "file.pdf",
		TenantID:      "MCS",
		BlobID:        "1234567890",
		LastAccess:    int(time.Now().UnixNano() / 1000000),
		Retention:     0,
		Properties:    make(map[string]interface{}),
	}
	blobDescription.Properties["X-es-user"] = []string{"Hallo", "Hallo2"}
	blobDescription.Properties["X-es-retention"] = []int{123456}
	blobDescription.Properties["X-es-tenant"] = "MCS"
	jsonStr, err := json.MarshalIndent(blobDescription, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(jsonStr))

	var blobInfo *BlobDescription
	err = json.Unmarshal(jsonStr, &blobInfo)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, blobDescription.BlobID, blobInfo.BlobID)

	haloStr, ok := blobInfo.Properties["X-es-user"]
	if !ok {
		t.Fatal("header not found")
	}
	halloList := haloStr.([]interface{})
	assert.Equal(t, "Hallo", halloList[0].(string))

	retention, ok := blobInfo.Properties["X-es-retention"]
	if !ok {
		t.Fatal("header not found")
	}
	retList := retention.([]interface{})
	assert.Equal(t, 123456., retList[0])

	tenant, ok := blobInfo.Properties["X-es-tenant"]
	if !ok {
		t.Fatal("header not found")
	}
	assert.Equal(t, "MCS", tenant.(string))
}
