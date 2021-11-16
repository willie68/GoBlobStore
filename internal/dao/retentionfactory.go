package dao

import (
	"fmt"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/retentionmanager"
)

// createRetentionManager creates a new Retention manager depending o nthe configuration
func createRetentionManager(rtnMgrStr string) (interfaces.RetentionManager, error) {
	switch rtnMgrStr {
	//This is the single node retention manager
	case retentionmanager.SingleRetentionManagerName:
		rtnMgr := &retentionmanager.SingleRetentionManager{
			TntDao:  tenantDao,
			MaxSize: 10000,
		}
		return rtnMgr, nil
	default:
		return nil, fmt.Errorf("no rentention manager found for class: %s", rtnMgrStr)
	}
	return nil, fmt.Errorf("no rentention manager found for class: %s", rtnMgrStr)
}
