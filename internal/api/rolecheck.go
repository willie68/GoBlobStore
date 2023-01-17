package api

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/serror"
)

// Role definig a role
type Role string

// RoleChecker interface for checking roles
type RoleChecker interface {
	CheckRole(ctx context.Context, allowedRoles []Role) bool
}

// RoleCheckerImpl the default implementation of the role checker
var RoleCheckerImpl RoleChecker

// definition of default roles
const (
	RoleObjectReader  Role = "object-reader"
	RoleObjectCreator Role = "object-creator"
	RoleObjectAdmin   Role = "object-admin"
	RoleTenantAdmin   Role = "tenant-admin"
	RoleAdmin         Role = "admin"
)

// RoleCheck implements a simple middleware handler for adding basic http auth to a route.
func RoleCheck(allowedRoles []Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if (RoleCheckerImpl != nil) && !RoleCheckerImpl.CheckRole(r.Context(), allowedRoles) {
				msg := "not allowed"
				apierr := serror.Forbidden(nil, msg)
				render.Status(r, apierr.Code)
				render.JSON(w, r, apierr)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
