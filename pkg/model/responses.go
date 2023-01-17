package model

import (
	"encoding/json"

	"github.com/willie68/GoBlobStore/internal/config"
)

// GetConfigResponse REST response for config request
type GetConfigResponse struct {
	TenantID string `json:"tenantid"`
	// to be compatible
	Created    bool           `json:"created"`
	Backup     config.Storage `json:"backup"`
	LastError  error          `json:"lastError"`
	Properties map[string]any `json:"properties"`
}

// MarshalJSON marshall this to JSON
func (r GetConfigResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type       string         `json:"type"`
		TenantID   string         `json:"tenantid"`
		Created    bool           `json:"created"`
		Backup     config.Storage `json:"backup"`
		LastError  error          `json:"lastError"`
		Properties map[string]any `json:"properties"`
	}{
		Type:       "configResponse",
		TenantID:   r.TenantID,
		Created:    r.Created,
		Backup:     r.Backup,
		LastError:  r.LastError,
		Properties: r.Properties,
	})
}

// CreateResponse REST response for create response
type CreateResponse struct {
	TenantID string `json:"tenantid"`
	Backup   string `json:"backup"`
}

// MarshalJSON marshall this to JSON
func (r CreateResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type     string `json:"type"`
		TenantID string `json:"tenantid"`
		Backup   string `json:"backup"`
	}{
		Type:     "createResponse",
		TenantID: r.TenantID,
		Backup:   r.Backup,
	})
}

// DeleteResponse REST response for delete response
type DeleteResponse struct {
	TenantID  string `json:"tenantid"`
	ProcessID string `json:"processid"`
}

// MarshalJSON marshall this to JSON
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

// SizeResponse REST response for size response
type SizeResponse struct {
	TenantID string `json:"tenantid"`
	Size     int64  `json:"size"`
}

// MarshalJSON marshall this to JSON
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
