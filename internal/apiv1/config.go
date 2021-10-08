package apiv1

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/dao"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"
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
	tenant := getTenant(request)
	if tenant == "" {
		msg := fmt.Sprintf("tenant header %s missing", httputils.TenantHeader)
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	dao, err := dao.GetTenantDao()
	if err != nil {
		outputError(response, err)
		return
	}
	render.JSON(response, request, dao.HasTenant(tenant))
}

/*
PostConfigEndpoint create a new store for a tenant
because of the automatic store creation, this method will always return 201
*/
func PostConfigEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant := getTenant(request)
	if tenant == "" {
		msg := fmt.Sprintf("tenant header %s missing", httputils.TenantHeader)
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	log.Printf("create store for tenant %s", tenant)
	dao, err := dao.GetTenantDao()
	if err != nil {
		outputError(response, err)
		return
	}

	err = dao.AddTenant(tenant)
	if err != nil {
		outputError(response, err)
		return
	}

	render.Status(request, http.StatusCreated)
	render.JSON(response, request, tenant)
}

/*
DeleteConfigEndpoint deleting store for a tenant, this will automatically delete all data in the store
*/
func DeleteConfigEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant := getTenant(request)
	if tenant == "" {
		msg := fmt.Sprintf("tenant header %s missing", httputils.TenantHeader)
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	dao, err := dao.GetTenantDao()
	if err != nil {
		outputError(response, err)
		return
	}
	if !dao.HasTenant(tenant) {
		httputils.Err(response, request, serror.NotFound("tenant", tenant, nil))
		return
	}
	err = dao.RemoveTenant(tenant)
	if err != nil {
		outputError(response, err)
		return
	}
	render.JSON(response, request, tenant)
}

/*
GetConfigSizeEndpoint size of the store for a tenant
*/
func GetConfigSizeEndpoint(response http.ResponseWriter, request *http.Request) {
	tenant := getTenant(request)
	if tenant == "" {
		msg := fmt.Sprintf("tenant header %s missing", httputils.TenantHeader)
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	render.JSON(response, request, tenant)
}
