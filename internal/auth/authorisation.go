package auth

import (
	"context"
	"strings"

	"github.com/willie68/GoBlobStore/internal/api"
)

type JWTRoleChecker struct {
	Config JWTAuthConfig
}

func (j JWTRoleChecker) CheckRole(ctx context.Context, allowedRoles []api.Role) bool {
	if !j.Config.Active {
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
