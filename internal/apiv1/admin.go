package apiv1

import (
	"net/http"

	log "github.com/willie68/GoBlobStore/internal/logging"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/dao"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const AdminSubpath = "/admin"

/*
AdminRoutes getting all routes for the admin endpoint
*/
func AdminRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/check", GetCheckEndpoint)
	router.Get("/processes", GetProcessesEndpoint)
	router.Get("/processes/{id}", GetProcessEndpoint)
	router.Delete("/processes/{id}", DeleteProcessEndpoint)
	return router
}

// GetProcessesEndpoint getting all processes active for this tenant
// @Summary getting all processes active for this tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} GetResponse "response with the id of the tenant and the created flag as bool as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /admin/processes [get]
func GetProcessesEndpoint(response http.ResponseWriter, request *http.Request) {
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

// GetCheckEndpoint starting a new check for this tenant
// @Summary starting a new check for this tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} CreateResponse "response with the id of the tenant as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /admin/command [post]
func GetCheckEndpoint(response http.ResponseWriter, request *http.Request) {
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

	

	render.Status(request, http.StatusCreated)
	render.JSON(response, request, rsp)
}

// GetProcessEndpoint getting a single process for a tenant
// @Summary getting a single process for a tenant
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} SizeResponse "response with the size as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /admin/processes/{id} [get]
func GetProcessEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	idStr := chi.URLParam(request, "id")
	if idStr == "" {
		msg := "process id missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-id", msg))
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

// DeleteProcessEndpoint deleting a single process for a tenant, if the process is finished
// @Summary deleting a single process for a tenant, if the process is finished
// @Tags configs
// @Accept  json
// @Produce  json
// @Security api_key
// @Param tenant header string true "Tenant"
// @Success 200 {array} DeleteResponse "response with the id of the started process for deletion as json"
// @Failure 400 {object} serror.Serr "client error information as json"
// @Failure 500 {object} serror.Serr "server error information as json"
// @Router /admin/processes/{id} [delete]
func DeleteProcessEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	idStr := chi.URLParam(request, "id")
	if idStr == "" {
		msg := "process id missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-id", msg))
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
