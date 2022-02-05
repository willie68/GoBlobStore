package apiv1

import (
	"errors"
	"net/http"

	log "github.com/willie68/GoBlobStore/internal/logging"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/willie68/GoBlobStore/internal/dao"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"
)

const AdminSubpath = "/admin"

/*
AdminRoutes getting all routes for the admin endpoint
*/
func AdminRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Get("/check", GetCheck)
	router.Post("/check", PostCheck)
	router.Post("/restore", PostRestore)
	return router
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
// @Router /admin/command [post]
func GetCheck(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	log.Logger.Infof("create store for tenant %s", tenant)
	cMan, err := dao.GetCheckManagement()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	res, err := cMan.GetCheckResult(tenant)
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
// @Router /admin/command [post]
func PostCheck(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	log.Logger.Infof("do check for tenant %s", tenant)
	cMan, err := dao.GetCheckManagement()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if cMan.IsCheckRunning(tenant) {
		httputils.Err(response, request, serror.BadRequest(errors.New("Check is already running for tenant")))
		return
	}
	_, err = cMan.StartCheck(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	res, err := cMan.GetCheckResult(tenant)
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
// @Router /admin/command [post]
func PostRestore(response http.ResponseWriter, request *http.Request) {
	tenant, err := httputils.TenantID(request)
	if err != nil {
		msg := "tenant header missing"
		httputils.Err(response, request, serror.BadRequest(nil, "missing-tenant", msg))
		return
	}
	log.Logger.Infof("do restore for tenant %s", tenant)
	cMan, err := dao.GetCheckManagement()
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	if cMan.IsCheckRunning(tenant) {
		httputils.Err(response, request, serror.BadRequest(errors.New("Check is already running for tenant")))
		return
	}
	_, err = cMan.StartCheck(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	res, err := cMan.GetCheckResult(tenant)
	if err != nil {
		httputils.Err(response, request, serror.InternalServerError(err))
		return
	}
	render.Status(request, http.StatusCreated)
	render.JSON(response, request, res)
}
