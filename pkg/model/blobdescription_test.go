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
		Hash:          "sha-256:fbbab289f7f94b25736c58be46a994c441fd02552cc6022352e3d86d2fab7c83",
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

func TestCheck(t *testing.T) {
	ast := assert.New(t)
	now := time.Now()
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
		Hash:          "sha-256:fbbab289f7f94b25736c58be46a994c441fd02552cc6022352e3d86d2fab7c83",
		Check: &Check{
			Store: &CheckInfo{
				LastCheck: &now,
				Healthy:   true,
				Message:   "checked",
			},
			Backup: &CheckInfo{
				LastCheck: &now,
				Healthy:   true,
				Message:   "checked",
			},
			Healthy: true,
			Message: "checked",
		},
	}
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

	ast.Equal(blobDescription.BlobID, blobInfo.BlobID)

	ast.True(now.Equal(*blobInfo.Check.Store.LastCheck))
	ast.True(now.Equal(*blobInfo.Check.Backup.LastCheck))

	ast.True(blobInfo.Check.Healthy)
	ast.True(blobInfo.Check.Store.Healthy)
	ast.True(blobInfo.Check.Backup.Healthy)
}
