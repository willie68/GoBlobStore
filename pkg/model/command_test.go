package model

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/willie68/GoBlobStore/internal/utils/jsonutils"

	"github.com/stretchr/testify/assert"
)

func TestCommandSerialisation(t *testing.T) {
	ast := assert.New(t)
	cmd := Command{
		Command:   CheckCmd,
		TenantID:  "mcs",
		Parameter: map[string]interface{}{"check": true, "text": "text", "int": int64(42)},
	}
	ast.NotNil(cmd)

	jsonStr, err := json.Marshal(cmd)
	ast.Nil(err)
	ast.NotNil(jsonStr)

	fmt.Printf("json: %s", jsonStr)

	var newCmd Command
	err = jsonutils.DecodeBytes(jsonStr, &newCmd)
	newCmd.Parameter = jsonutils.ConvertJson2Map(newCmd.Parameter)
	ast.Nil(err)

	ast.Equal(cmd.Command, newCmd.Command)
	ast.Equal(cmd.TenantID, newCmd.TenantID)
	ast.Equal(len(cmd.Parameter), len(newCmd.Parameter))
	for k, v := range cmd.Parameter {
		value, ok := newCmd.Parameter[k]
		ast.True(ok)
		ast.Equal(v, value)
	}
}
