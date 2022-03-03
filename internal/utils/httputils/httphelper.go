package httputils

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/auth"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/serror"
)

// Validate validator
var val *validator.Validate
var TenantClaim string
var Strict bool

// TenantID gets the tenant-id of the given request
func TenantID(r *http.Request) (string, error) {
	tntID := chi.URLParam(r, api.URL_PARAM_TENANT_ID)
	if tntID != "" {
		return strings.ToLower(tntID), nil
	}
	var id string
	_, claims, _ := auth.FromContext(r.Context())
	if claims != nil {
		tenant, ok := claims[TenantClaim].(string)
		if ok {
			return strings.ToLower(tenant), nil
		} else {
			if Strict {
				return "", serror.BadRequest(nil, "missing-tenant", "no tenant claim in jwt token")
			}
		}
	}
	tenantHeader, ok := config.Get().HeaderMapping[api.TenantHeaderKey]
	if ok {
		id = r.Header.Get(tenantHeader)
	}
	if id == "" {
		msg := fmt.Sprintf("tenant header %s missing", tenantHeader)
		return "", serror.BadRequest(nil, "missing-tenant", msg)
	}
	return strings.ToLower(id), nil
}

// Decode decodes and validates an object
func Decode(r *http.Request, v interface{}) error {
	err := render.DefaultDecoder(r, v)
	if err != nil {
		serror.BadRequest(err, "decode-body", "could not decode body")
	}
	if err := val.Struct(v); err != nil {
		serror.BadRequest(err, "validate-body", "body invalid")
	}
	return nil
}

// Param gets the url param of the given request
func Param(r *http.Request, name string) (string, error) {
	cid := chi.URLParam(r, name)
	if cid == "" {
		msg := fmt.Sprintf("missing %s in path", name)
		return "", serror.BadRequest(nil, "missing-param", msg)
	}
	return cid, nil
}

// Created object created
func Created(w http.ResponseWriter, r *http.Request, id string, v interface{}) {
	// TODO add relative path to location
	w.Header().Add("Location", fmt.Sprintf("%s", id))
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, v)
}

// Err writes an error response
func Err(w http.ResponseWriter, r *http.Request, err error) {
	apierr := serror.Wrap(err, "unexpected-error")
	render.Status(r, apierr.Code)
	render.JSON(w, r, apierr)
}

func init() {
	val = validator.New()
}
