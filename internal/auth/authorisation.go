package auth

import (
	"context"
	"strings"

	"github.com/willie68/GoBlobStore/internal/api"
)

// JWTRoleChecker checking a user role against the configuration
type JWTRoleChecker struct {
	Config JWTAuthConfig
}

// JWTTntChecker checking a user tenant against the configuration
type JWTTntChecker struct {
	Config JWTAuthConfig
}

// CheckRole checking the user role against the given in the REST Api route
func (j JWTRoleChecker) CheckRole(ctx context.Context, allowedRoles []api.Role) bool {
	if !j.Config.Active || !j.Config.RoleActive {
		return true
	}
	_, claims, _ := FromContext(ctx)
	if claims != nil {
		userroles, ok := claims[j.Config.RoleClaim].([]any)
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

// CheckTenant checking the user tenant against the given in the REST Api route
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
		}
		if j.Config.Strict {
			return false
		}
	}
	return false
}
