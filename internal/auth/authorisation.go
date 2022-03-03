package auth

import (
	"context"
	"strings"

	"github.com/willie68/GoBlobStore/internal/api"
)

type JWTRoleChecker struct {
	Config JWTAuthConfig
}

type JWTTntChecker struct {
	Config JWTAuthConfig
}

func (j JWTRoleChecker) CheckRole(ctx context.Context, allowedRoles []api.Role) bool {
	if !j.Config.Active || !j.Config.RoleActive {
		return true
	}
	_, claims, _ := FromContext(ctx)
	if claims != nil {
		userroles, ok := claims[j.Config.RoleClaim].([]interface{})
		if !ok {
			return false
		}
		for _, uri := range userroles {
			ur := uri.(string)
			for _, r := range allowedRoles {

				if strings.EqualFold(ur, string(r)) {
					return true
				}
			}
		}
	}
	return false
}

func (j JWTTntChecker) CheckTenant(ctx context.Context, tenant string) bool {
	if !j.Config.Active {
		return true
	}
	if j.Config.TenantClaim == "" {
		return true
	}
	_, claims, _ := FromContext(ctx)
	if claims != nil {
		jwtTenant, ok := claims[j.Config.TenantClaim].(string)
		if ok {
			return strings.EqualFold(tenant, jwtTenant)
		} else {
			if j.Config.Strict {
				return false
			}
		}
	}
	return false
}
