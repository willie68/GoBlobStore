package apiv1

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/dao"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const ConfigSubpath = "/config"

/*
ConfigRoutes getting all routes for the config endpoint
*/
func ConfigRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Post("/", PostConfigEndpoint)
	router.Get("/", GetConfigEndpoint)
	router.Delete("/", DeleteConfigEndpoint)
	router.Get("/size", GetConfigSizeEndpoint)
	return router
}

/*
GetConfigEndpoint getting if a store for a tenant is initialised
because of the automatic store creation, the value is more likely that data is stored for this tenant
*/
func GetConfigEndpoint(response http.ResponseWriter, request *http.Request) {
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
	render.JSON(response, request, dao.HasTenant(tenant))
}

/*
PostConfigEndpoint create a new store for a tenant
because of the automatic store creation, this method will always return 201
*/
func PostConfigEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	log.Printf("create store for tenant %s", tenant)
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

	render.Status(request, http.StatusCreated)
	render.JSON(response, request, tenant)
}

// DeleteConfigEndpoint deleting the store for a tenant, this will automatically delete all data in the store async
// @Summary deleting the store for a tenant, this will automatically delete all data in the store. On sync you will get an empty string as process, for async operations you will get an id of the deletion process
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} DeleteResponse "response with the id of the started process for deletion as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /config [get]
func DeleteConfigEndpoint(response http.ResponseWriter, request *http.Request) {
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

// GetConfigSizeEndpoint size of the store for a tenant
// @Summary Get the size of the store for a tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} SizeResponse "response with the size as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /config [get]
func GetConfigSizeEndpoint(response http.ResponseWriter, request *http.Request) {
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
