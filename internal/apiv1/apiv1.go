package apiv1

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httptracer"
	"github.com/go-chi/render"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/auth"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/health"
	log "github.com/willie68/GoBlobStore/internal/logging"
)

// APIVersion the actual implemented api version
const APIVersion = "1"

// BaseURL is the url all endpoints will be available under
var BaseURL = fmt.Sprintf("/api/v%s", APIVersion)

// APIKey the apikey of this service
var APIKey string

const adminSubpath = "/admin"
const storesSubpath = "/stores"
const configSubpath = "/config"
const blobsSubpath = "/blobs"
const searchSubpath = "/search"

// APIRoutes defining all api v1 routes
func APIRoutes(cfn config.Config, trc opentracing.Tracer) (*chi.Mux, error) {
	APIKey = getApikey()
	log.Root.Infof("baseurl : %s", BaseURL)
	router := chi.NewRouter()
	setDefaultHandler(router, cfn, trc)

	if cfn.Apikey {
		setApikeyHandler(cfn, router)
	}

	// jwt is activated, register the Authenticator and Validator
	if strings.EqualFold(cfn.Auth.Type, "jwt") {
		jwtConfig, err := auth.ParseJWTConfig(cfn.Auth)
		if err != nil {
			return router, err
		}
		log.Root.Infof("jwt config: %v", jwtConfig)

		auth.InitJWT(jwtConfig)

		jwtAuth := auth.JWTAuth{
			Config: jwtConfig,
		}
		router.Use(
			auth.Verifier(&jwtAuth),
			auth.Authenticator,
		)
		api.RoleCheckerImpl = &auth.JWTRoleChecker{
			Config: jwtConfig,
		}
		api.TntCheckerImpl = &auth.JWTTntChecker{
			Config: jwtConfig,
		}
	}

	// building the routes
	router.Route("/", func(r chi.Router) {
		r.Mount(BlobRoutes())
		r.Mount(SearchRoutes())
		r.Mount(ConfigRoutes())
		r.Mount(AdminRoutes())
		r.Mount(StoresRoutes())
		r.Mount(TenantStoresRoutes())
		r.Mount("/", health.Routes())
		if cfn.Metrics.Enable {
			r.Mount(api.MetricsEndpoint, promhttp.Handler())
		}
	})
	log.Root.Infof("%s api routes", config.Servicename)

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Root.Infof("api route: %s %s", method, route)
		return nil
	}

	if err := chi.Walk(router, walkFunc); err != nil {
		log.Root.Alertf("could not walk api routes. %s", err.Error())
	}

	return router, nil
}

func setApikeyHandler(cfn config.Config, router *chi.Mux) {
	router.Use(
		api.SysAPIHandler(api.SysAPIConfig{
			Apikey:           APIKey,
			HeaderKeyMapping: cfn.HeaderMapping,
			SkipFunc: func(r *http.Request) bool {
				path := strings.TrimSuffix(r.URL.Path, "/")
				if strings.HasSuffix(path, "/livez") {
					return true
				}
				if strings.HasSuffix(path, "/readyz") {
					return true
				}
				if strings.HasSuffix(path, api.MetricsEndpoint) {
					return true
				}
				return false
			},
		}),
	)
}

func setDefaultHandler(router *chi.Mux, cfn config.Config, tracer opentracing.Tracer) {
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.Recoverer,
		cors.Handler(cors.Options{
			// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
			AllowedOrigins: []string{"*"},
			// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-mcs-username", "X-mcs-password", "X-mcs-profile"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}),
	)
	if tracer != nil {
		router.Use(httptracer.Tracer(tracer, httptracer.Config{
			ServiceName:    config.Servicename,
			ServiceVersion: "V" + APIVersion,
			SampleRate:     1,
			Tags: map[string]any{
				"_dd.measured": 1, // datadog, turn on metrics for http.request stats
				// "_dd1.sr.eausr": 1, // datadog, event sample rate
			},
		}))
	}
	if cfn.Metrics.Enable {
		router.Use(
			api.MetricsHandler(api.MetricsConfig{}),
		)
	}
}

// HealthRoutes returning the health routes
func HealthRoutes(cfn config.Config) *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		middleware.Recoverer,
	)
	if cfn.Metrics.Enable {
		router.Use(
			api.MetricsHandler(api.MetricsConfig{}),
		)
	}

	router.Route("/", func(r chi.Router) {
		r.Mount("/", health.Routes())
		if cfn.Metrics.Enable {
			r.Mount(api.MetricsEndpoint, promhttp.Handler())
		}
	})

	log.Root.Info("health api routes")
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Root.Infof("health route: %s %s", method, route)
		return nil
	}
	if err := chi.Walk(router, walkFunc); err != nil {
		log.Root.Alertf("could not walk health routes. %s", err.Error())
	}

	return router
}

// getApikey generate an apikey based on the service name
func getApikey() string {
	value := fmt.Sprintf("%s_%s", config.Servicename, "default")
	apikey := fmt.Sprintf("%x", md5.Sum([]byte(value)))
	return strings.ToLower(apikey)
}
