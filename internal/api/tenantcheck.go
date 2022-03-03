package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/serror"
)

type TenantChecker interface {
	CheckTenant(ctx context.Context, tenant string) bool
}

var TntCheckerImpl TenantChecker

// TenantCheck implements a simple middleware handler for adding basic http auth to a route.
func TenantCheck() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if TntCheckerImpl != nil {
				tenant := chi.URLParam(r, URL_PARAM_TENANT_ID)
				if tenant != "" {
					if !TntCheckerImpl.CheckTenant(r.Context(), tenant) {
						msg := "not allowed"
						apierr := serror.Forbidden(nil, msg)
						render.Status(r, apierr.Code)
						render.JSON(w, r, apierr)
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
