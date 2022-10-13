package model

import (
	"encoding/json"

	"github.com/willie68/GoBlobStore/internal/config"
)

type GetConfigResponse struct {
	TenantID string `json:"tenantid"`
	// to be compatible
	Created    bool           `json:"created"`
	Backup     config.Storage `json:"backup"`
	Properties map[string]any `json:"properties"`
}

func (r GetConfigResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       string         `json:"type"`
		TenantID   string         `json:"tenantid"`
		Created    bool           `json:"created"`
		Backup     config.Storage `json:"backup"`
		Properties map[string]any `json:"properties"`
	}{
		Type:       "configResponse",
		TenantID:   r.TenantID,
		Created:    r.Created,
		Backup:     r.Backup,
		Properties: r.Properties,
	})
}

type CreateResponse struct {
	TenantID string `json:"tenantid"`
}

func (r CreateResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type     string `json:"type"`
		TenantID string `json:"tenantid"`
	}{
		Type:     "createResponse",
		TenantID: r.TenantID,
	})
}

type DeleteResponse struct {
	TenantID  string `json:"tenantid"`
	ProcessID string `json:"processid"`
}

func (r DeleteResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type      string `json:"type"`
		TenantID  string `json:"tenantid"`
		ProcessID string `json:"processid"`
	}{
		Type:      "deleteResponse",
		TenantID:  r.TenantID,
		ProcessID: r.ProcessID,
	})
}

type SizeResponse struct {
	TenantID string `json:"tenantid"`
	Size     int64  `json:"size"`
}

func (r SizeResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type     string `json:"type"`
		TenantID string `json:"tenantid"`
		Size     int64  `json:"size"`
	}{
		Type:     "sizeResponse",
		TenantID: r.TenantID,
		Size:     r.Size,
	})
}
