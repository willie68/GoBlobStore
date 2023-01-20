// Package main this is the entry point into the service
package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httptracer"
	"github.com/go-chi/render"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/apiv1"
	"github.com/willie68/GoBlobStore/internal/auth"
	"github.com/willie68/GoBlobStore/internal/config"
	"github.com/willie68/GoBlobStore/internal/crypt"
	"github.com/willie68/GoBlobStore/internal/dao"
	"github.com/willie68/GoBlobStore/internal/health"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/serror"
	"github.com/willie68/GoBlobStore/internal/utils/httputils"

	flag "github.com/spf13/pflag"
)

var port int
var sslport int
var serviceURL string
var apikey string
var ssl bool
var configFile string
var serviceConfig config.Config
var tracer opentracing.Tracer
var sslsrv *http.Server
var srv *http.Server

func init() {
	// variables for parameter override
	ssl = false
	log.Logger.Info("init service")
	flag.IntVarP(&port, "port", "p", 0, "port of the http server.")
	flag.IntVarP(&sslport, "sslport", "t", 0, "port of the https server.")
	flag.StringVarP(&configFile, "config", "c", "", "this is the path and filename to the config file")
	flag.StringVarP(&serviceURL, "serviceURL", "u", "", "service url from outside")
}

func apiRoutes() (*chi.Mux, error) {
	log.Logger.Infof("baseurl : %s", apiv1.BaseURL)
	router := chi.NewRouter()
	setDefaultHandler(router)

	if serviceConfig.Apikey {
		setApikeyHandler(router)
	}

	// jwt is activated, register the Authenticator and Validator
	if strings.EqualFold(serviceConfig.Auth.Type, "jwt") {
		jwtConfig, err := auth.ParseJWTConfig(serviceConfig.Auth)
		if err != nil {
			return router, err
		}
		log.Logger.Infof("jwt config: %v", jwtConfig)

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
		r.Mount(apiv1.BlobRoutes())
		r.Mount(apiv1.SearchRoutes())
		r.Mount(apiv1.ConfigRoutes())
		r.Mount(apiv1.AdminRoutes())
		r.Mount(apiv1.StoresRoutes())
		r.Mount(apiv1.TenantStoresRoutes())
		r.Mount("/", health.Routes())
		if serviceConfig.Metrics.Enable {
			r.Mount("/metrics", promhttp.Handler())
		}
	})

	return router, nil
}

func setApikeyHandler(router *chi.Router) {
	router.Use(
		api.SysAPIHandler(api.SysAPIConfig{
			Apikey:           apikey,
			HeaderKeyMapping: serviceConfig.HeaderMapping,
			SkipFunc: func(r *http.Request) bool {
				path := strings.TrimSuffix(r.URL.Path, "/")
				if strings.HasSuffix(path, "/livez") {
					return true
				}
				if strings.HasSuffix(path, "/readyz") {
					return true
				}
				if strings.HasSuffix(path, "/metrics") {
					return true
				}
				return false
			},
		}),
	)
}

func setDefaultHandler(router *chi.Router) {
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		//middleware.DefaultCompress,
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
		httptracer.Tracer(tracer, httptracer.Config{
			ServiceName:    config.Servicename,
			ServiceVersion: "V" + apiv1.APIVersion,
			SampleRate:     1,
			SkipFunc: func(r *http.Request) bool {
				return false
				//return r.URL.Path == "/livez"
			},
			Tags: map[string]any{
				"_dd.measured": 1, // datadog, turn on metrics for http.request stats
				// "_dd1.sr.eausr": 1, // datadog, event sample rate
			},
		}),
	)
	if serviceConfig.Metrics.Enable {
		router.Use(
			api.MetricsHandler(api.MetricsConfig{
				SkipFunc: func(r *http.Request) bool {
					return false
				},
			}),
		)
	}
}

func healthRoutes() *chi.Mux {
	router := chi.NewRouter()
	router.Use(
		render.SetContentType(render.ContentTypeJSON),
		middleware.Logger,
		//middleware.DefaultCompress,
		middleware.Recoverer,
		httptracer.Tracer(tracer, httptracer.Config{
			ServiceName:    config.Servicename,
			ServiceVersion: "V" + apiv1.APIVersion,
			SampleRate:     1,
			SkipFunc: func(r *http.Request) bool {
				return false
			},
			Tags: map[string]any{
				"_dd.measured": 1, // datadog, turn on metrics for http.request stats
				// "_dd1.sr.eausr": 1, // datadog, event sample rate
			},
		}),
	)
	if serviceConfig.Metrics.Enable {
		router.Use(
			api.MetricsHandler(api.MetricsConfig{
				SkipFunc: func(r *http.Request) bool {
					return false
				},
			}),
		)
	}

	router.Route("/", func(r chi.Router) {
		r.Mount("/", health.Routes())
		if serviceConfig.Metrics.Enable {
			r.Mount("/metrics", promhttp.Handler())
		}
	})
	return router
}

// @title GoBlobStore service API
// @version 1.0
// @description The GoBlobStore service is a micro services for storing and serving binary data.
// @BasePath /api/v1
// @securityDefinitions.apikey api_key
// @in header
// @name apikey
func main() {
	configFolder, err := config.GetDefaultConfigFolder()
	if err != nil {
		panic("can't get config folder")
	}

	flag.Parse()

	log.Logger.Infof("starting server, config folder: %s", configFolder)
	defer log.Logger.Close()

	serror.Service = config.Servicename
	if configFile == "" {
		configFile, err = getDefaultConfigfile()
		if err != nil {
			log.Logger.Errorf("error getting default config file: %v", err)
			panic("error getting default config file")
		}
	}

	config.File = configFile
	log.Logger.Infof("using config file: %s", configFile)

	// autorestart starts here...
	if err := config.Load(); err != nil {
		log.Logger.Alertf("can't load config file: %s", err.Error())
		panic("can't load config file")
	}

	serviceConfig = config.Get()
	initConfig()
	initLogging()

	log.Logger.Info("service is starting")

	var closer io.Closer
	tracer, closer = initJaeger(config.Servicename, serviceConfig.OpenTracing)
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	healthCheckConfig := health.CheckConfig(serviceConfig.HealthCheck)

	health.InitHealthSystem(healthCheckConfig, tracer)

	if serviceConfig.Sslport > 0 {
		ssl = true
		log.Logger.Info("ssl active")
	}

	apikey = getApikey()
	if config.Get().Apikey {
		log.Logger.Infof("apikey: %s", apikey)
	}
	log.Logger.Infof("ssl: %t", ssl)
	log.Logger.Infof("serviceURL: %s", serviceConfig.ServiceURL)
	log.Logger.Infof("%s api routes", config.Servicename)

	if err := initStorageSystem(); err != nil {
		errstr := fmt.Sprintf("could not initialise dao factory. %s", err.Error())
		log.Logger.Alertf(errstr)
		panic(errstr)
	}

	router, err := apiRoutes()
	if err != nil {
		errstr := fmt.Sprintf("could not create api routes. %s", err.Error())
		log.Logger.Alertf(errstr)
		panic(errstr)
	}
	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		log.Logger.Infof("%s %s", method, route)
		return nil
	}

	if err := chi.Walk(router, walkFunc); err != nil {
		log.Logger.Alertf("could not walk api routes. %s", err.Error())
	}
	log.Logger.Info("health api routes")
	healthRouter := healthRoutes()
	if err := chi.Walk(healthRouter, walkFunc); err != nil {
		log.Logger.Alertf("could not walk health routes. %s", err.Error())
	}

	if ssl {
		startHTTPSServer(router)
		startHTTPServer(healthRouter)
	} else {
		startHTTPServer(router)
	}

	log.Logger.Info("waiting for clients")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	if err = srv.Shutdown(ctx); err != nil {
		log.Logger.Errorf("shutdown http server error: %v", err)
	}
	if ssl {
		if err = sslsrv.Shutdown(ctx); err != nil {
			log.Logger.Errorf("shutdown https server error: %v", err)
		}
	}

	log.Logger.Info("finished")

	os.Exit(0)
}

func getDefaultConfigfile() (string, error) {
	configFolder, err := config.GetDefaultConfigFolder()
	if err != nil {
		return "", errors.Wrap(err, "can't load config file")
	}
	configFolder = fmt.Sprintf("%s/service/", configFolder)
	err = os.MkdirAll(configFolder, os.ModePerm)
	if err != nil {
		return "", errors.Wrap(err, "can't load config file")
	}
	return configFolder + "/service.yaml", nil
}

func startHTTPSServer(router *chi.Router) {
	gc := crypt.GenerateCertificate{
		Organization: "MCS",
		Host:         "127.0.0.1",
		ValidFor:     10 * 365 * 24 * time.Hour,
		IsCA:         false,
		EcdsaCurve:   "P384",
		Ed25519Key:   false,
	}
	tlsConfig, err := gc.GenerateTLSConfig()
	if err != nil {
		log.Logger.Alertf("could not create tls config. %s", err.Error())
	}
	sslsrv = &http.Server{
		Addr:         "0.0.0.0:" + strconv.Itoa(serviceConfig.Sslport),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
		TLSConfig:    tlsConfig,
	}
	go func() {
		log.Logger.Infof("starting https server on address: %s", sslsrv.Addr)
		if err := sslsrv.ListenAndServeTLS("", ""); err != nil {
			log.Logger.Alertf("error starting server: %s", err.Error())
		}
	}()
}

func startHTTPServer(router *chi.Router) {
	// own http server for the healthchecks
	srv = &http.Server{
		Addr:         "0.0.0.0:" + strconv.Itoa(serviceConfig.Port),
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}
	go func() {
		log.Logger.Infof("starting http server on address: %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Logger.Alertf("error starting server: %s", err.Error())
		}
	}()
}

func initLogging() {
	log.Logger.SetLevel(serviceConfig.Logging.Level)
	var err error
	serviceConfig.Logging.Filename, err = config.ReplaceConfigdir(serviceConfig.Logging.Filename)
	if err != nil {
		log.Logger.Errorf("error on config dir: %v", err)
	}
	log.Logger.GelfURL = serviceConfig.Logging.Gelfurl
	log.Logger.GelfPort = serviceConfig.Logging.Gelfport
	log.Logger.Init()
}

func initConfig() {
	if port > 0 {
		serviceConfig.Port = port
	}
	if sslport > 0 {
		serviceConfig.Sslport = sslport
	}
	if serviceURL != "" {
		serviceConfig.ServiceURL = serviceURL
	}

	httputils.TenantClaim = "Tenant"

	if strings.EqualFold(serviceConfig.Auth.Type, "jwt") {
		v, ok := serviceConfig.Auth.Properties["strict"]
		if ok {
			val, ok := v.(bool)
			if ok {
				httputils.Strict = val
			}
		}
		v, ok = serviceConfig.Auth.Properties["tenantClaim"]
		if ok {
			tc, ok := v.(string)
			if ok {
				httputils.TenantClaim = tc
			}
		}
	}
}

func initJaeger(servicename string, cnfg config.OpenTracing) (opentracing.Tracer, io.Closer) {
	cfg := jaegercfg.Configuration{
		ServiceName: servicename,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: cnfg.Host,
			CollectorEndpoint:  cnfg.Endpoint,
		},
	}
	if (cnfg.Endpoint == "") && (cnfg.Host == "") {
		cfg.Disabled = true
	}
	tracer, closer, err := cfg.NewTracer(jaegercfg.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func getApikey() string {
	value := fmt.Sprintf("%s_%s", config.Servicename, "default")
	apikey := fmt.Sprintf("%x", md5.Sum([]byte(value)))
	return strings.ToLower(apikey)
}

func initStorageSystem() error {
	return dao.Init(serviceConfig.Engine)
}
