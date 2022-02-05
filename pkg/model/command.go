package model

type CommandType string

const (
	CheckCmd   CommandType = "check"
	RestoreCmd             = "restore"
)

type Command struct {
	Command   CommandType            `json:"command"`
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenantid"`
	Parameter map[string]interface{} `json:"parameter"`
}
