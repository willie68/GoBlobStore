package factory

import (
	"fmt"

	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/internal/services/retentionmanager"
)

// CreateRetentionManager creates a new Retention manager depending ot he configuration
func CreateRetentionManager(rtnMgrStr string, tntsrv interfaces.TenantManager) (interfaces.RetentionManager, error) {
	if rtnMgrStr == retentionmanager.SingleRetentionManagerName {
		//This is the single node retention manager
		rtnMgr := &retentionmanager.SingleRetentionManager{
			TntSrv:  tntsrv,
			MaxSize: 10000,
		}
		return rtnMgr, nil
	}
	return nil, fmt.Errorf("no retention manager found for class: %s", rtnMgrStr)
}
