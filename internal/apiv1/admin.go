package apiv1

import (
	"errors"
	"net/http"

	"github.com/willie68/GoBlobStore/internal/api"
	log "github.com/willie68/GoBlobStore/internal/logging"
	services "github.com/willie68/GoBlobStore/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"
)

// AdminRoutes getting all routes for the admin endpoint
func AdminRoutes() (string, *chi.Mux) {
	router := chi.NewRouter()
	router.With(api.RoleCheck([]api.Role{api.RoleTenantAdmin})).Get("/check", GetCheck)
	router.With(api.RoleCheck([]api.Role{api.RoleTenantAdmin})).Post("/check", PostCheck)
	router.With(api.RoleCheck([]api.Role{api.RoleTenantAdmin})).Get("/restore", GetRestore)
	router.With(api.RoleCheck([]api.Role{api.RoleTenantAdmin})).Post("/restore", PostRestore)
	return BaseURL + adminSubpath, router
}

// GetCheck starting a new check for this tenant
// @Summary starting a new check for this tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} CreateResponse "response with the id of the tenant as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /api/v1/admin/check [get]
func GetCheck(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	cMan, err := services.GetMigrationManagement()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	res, err := cMan.GetResult(tenant)
	if err != nil {
		httputils.Err(response, request, serror.BadRequest(err))
		return
	}
	render.Status(request, http.StatusCreated)
	render.JSON(response, request, res)
}

// PostCheck starting a new check for this tenant
// @Summary starting a new check for this tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} CreateResponse "response with the id of the tenant as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /api/v1/admin/check [post]
func PostCheck(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	log.Logger.Infof("do check for tenant %s", tenant)
	cMan, err := services.GetMigrationManagement()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if cMan.IsRunning(tenant) {
		httputils.Err(response, request, serror.BadRequest(errors.New("Check is already running for tenant")))
		return
	}
	_, err = cMan.StartCheck(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	res, err := cMan.GetResult(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.Status(request, http.StatusCreated)
	render.JSON(response, request, res)
}

// PostRestore starting a new check for this tenant
// @Summary starting a new check for this tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} CreateResponse "response with the id of the tenant as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /api/v1/admin/restore [post]
func PostRestore(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	log.Logger.Infof("do restore for tenant %s", tenant)
	rMan, err := services.GetMigrationManagement()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if rMan.IsRunning(tenant) {
		httputils.Err(response, request, serror.BadRequest(errors.New("restore is already running for tenant")))
		return
	}
	_, err = rMan.StartRestore(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	res, err := rMan.GetResult(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.Status(request, http.StatusCreated)
	render.JSON(response, request, res)
}

// GetRestore starting a new check for this tenant
// @Summary starting a new check for this tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} CreateResponse "response with the id of the tenant as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /api/v1/admin/restore [get]
func GetRestore(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	rMan, err := services.GetMigrationManagement()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	res, err := rMan.GetResult(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.Status(request, http.StatusCreated)
	render.JSON(response, request, res)
}
