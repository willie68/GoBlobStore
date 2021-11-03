package dao

import (
	"fmt"
)

// createRetentionManager creates a new Retention manager depending o nthe configuration
func createRetentionManager(rtnMgrStr string) error {
	switch rtnMgrStr {
	//This is the single node retention manager
	case "SingleRetention":
		rtnMgr = &SingleRetentionManager{
			tntDao:  tenantDao,
			maxSize: 10000,
		}
	default:
		return fmt.Errorf("no rentention manager found for class: %s", rtnMgrStr)
	}
	return nil
}
