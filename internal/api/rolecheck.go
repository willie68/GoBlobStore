package api

import (
	"context"
	"net/http"

	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/serror"
)

type Role string

type RoleChecker interface {
	CheckRole(ctx context.Context, allowedRoles []Role) bool
}

var RoleCheckerImpl RoleChecker

const (
	R_OBJECT_READER  Role = "object-reader"
	R_OBJECT_CREATOR Role = "object-creator"
	R_OBJECT_ADMIN   Role = "object-admin"
	R_TENANT_ADMIN   Role = "tenant-admin"
	R_ADMIN          Role = "admin"
)

// RoleCheck implements a simple middleware handler for adding basic http auth to a route.
func RoleCheck(allowedRoles []Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !RoleCheckerImpl.CheckRole(r.Context(), allowedRoles) {
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
