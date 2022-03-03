package apiv1

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/dao"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

/*
ConfigRoutes getting all routes for the config endpoint
*/
func ConfigRoutes() (string, *chi.Mux) {
	router := chi.NewRouter()
	router.With(api.RoleCheck([]api.Role{api.R_ADMIN})).Post("/", PostCreateTenant)
	router.With(api.RoleCheck([]api.Role{api.R_TENANT_ADMIN})).Get("/", GetTenantConfig)
	router.With(api.RoleCheck([]api.Role{api.R_ADMIN})).Delete("/", DeleteTenant)
	router.With(api.RoleCheck([]api.Role{api.R_TENANT_ADMIN})).Get("/size", GetTenantSize)
	return BaseURL + configSubpath, router
}

/*
StoresRoutes getting all routes for the stores endpoint, this is part of the new api. But manly here only a new name.
*/
func StoresRoutes() (string, *chi.Mux) {
	router := chi.NewRouter()
	router.With(api.RoleCheck([]api.Role{api.R_ADMIN})).Post("/", PostCreateTenant)
	router.With(api.RoleCheck([]api.Role{api.R_TENANT_ADMIN})).Get("/", GetTenantConfig)
	router.With(api.RoleCheck([]api.Role{api.R_ADMIN})).Delete("/", DeleteTenant)
	router.With(api.RoleCheck([]api.Role{api.R_TENANT_ADMIN})).Get("/size", GetTenantSize)
	return BaseURL + configSubpath + storesSubpath, router
}

/*
GetTenantConfig
because of the automatic store creation, the value is more likely that data is stored for this tenant
*/
// GetTenantConfig getting if a store for a tenant is initialised
// @Summary getting if a store for a tenant is initialised, because of the automatic store creation, the value is more likely that data is stored for this tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} GetResponse "response with the id of the tenant and the created flag as bool as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /config [get]
func GetTenantConfig(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	dao, err := dao.GetTenantDao()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	rsp := model.GetResponse{
		TenantID: tenant,
		Created:  dao.HasTenant(tenant),
	}
	render.JSON(response, request, rsp)
}

// PostCreateTenant create a new store for a tenant because of the automatic store creation, this method will always return 201
// @Summary create a new store for a tenant because of the automatic store creation, this method will always return 201
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} CreateResponse "response with the id of the tenant as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /config [post]
func PostCreateTenant(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	log.Logger.Infof("create store for tenant %s", tenant)
	dao, err := dao.GetTenantDao()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	err = dao.AddTenant(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}

	rsp := model.CreateResponse{
		TenantID: tenant,
	}
	render.Status(request, http.StatusCreated)
	render.JSON(response, request, rsp)
}

// DeleteTenant deleting the store for a tenant, this will automatically delete all data in the store async
// @Summary deleting the store for a tenant, this will automatically delete all data in the store. On sync you will get an empty string as process, for async operations you will get an id of the deletion process
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} DeleteResponse "response with the id of the started process for deletion as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /config [delete]
func DeleteTenant(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	dao, err := dao.GetTenantDao()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if !dao.HasTenant(tenant) {
		httputils.Err(response, request, serror.NotFound("tenant", tenant, nil))
		return
	}
	process, err := dao.RemoveTenant(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	rsp := model.DeleteResponse{
		TenantID:  tenant,
		ProcessID: process,
	}
	render.JSON(response, request, rsp)
}

// GetTenantSize size of the store for a tenant
// @Summary Get the size of the store for a tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} SizeResponse "response with the size as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /config/size [get]
func GetTenantSize(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	dao, err := dao.GetTenantDao()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if !dao.HasTenant(tenant) {
		httputils.Err(response, request, serror.NotFound("tenant", tenant, nil))
		return
	}
	size := dao.GetSize(tenant)
	rsp := model.SizeResponse{
		TenantID: tenant,
		Size:     size,
	}
	render.JSON(response, request, rsp)
}
