package model

import "encoding/json"

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
