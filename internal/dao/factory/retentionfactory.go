package factory

import (
	"fmt"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	"github.com/willie68/GoBlobStore/internal/dao/retentionmanager"
)

// CreateRetentionManager creates a new Retention manager depending o nthe configuration
func CreateRetentionManager(rtnMgrStr string, tenantDao interfaces.TenantDao) (interfaces.RetentionManager, error) {
	switch rtnMgrStr {
	//This is the single node retention manager
	case retentionmanager.SingleRetentionManagerName:
		rtnMgr := &retentionmanager.SingleRetentionManager{
			TntDao:  tenantDao,
			MaxSize: 10000,
		}
		return rtnMgr, nil
	}
	return nil, fmt.Errorf("no rentention manager found for class: %s", rtnMgrStr)
}
