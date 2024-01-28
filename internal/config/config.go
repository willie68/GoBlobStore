package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"github.com/drone/envsubst"
	"github.com/pkg/errors"
	"github.com/samber/do"
	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/services/health"
	"gopkg.in/yaml.v3"
)

// Servicename the name of this service
const Servicename = "goblob-service"

// DoServiceConfig the name of the injected config
const DoServiceConfig = "service_config"

// Config our service configuration
type Config struct {
	// all secrets will be stored in this file, same structure as the main config file
	SecretFile string `yaml:"secretfile"`

	Apikey bool `yaml:"apikey"`
	// all configuration of internal services can be stored here
	Service Service `yaml:"service"`

	// configure logging to gelf logging system
	Logging logging.LoggingConfig `yaml:"logging"`

	Auth Authentication `yaml:"auth"`

	Engine Engine `yaml:"engine"`

	HeaderMapping map[string]string `yaml:"headermapping"`

	OpenTracing OpenTracing `yaml:"opentracing"`

	Metrics Metrics `yaml:"metrics"`

	Profiling Profiling `yaml:"profiling"`
}

// Service the configuration of services inside this ms
type Service struct {
	HTTP HTTP `yaml:"http"`
	// special config for health checks
	HealthSystem health.Config `yaml:"healthcheck"`
	// CA service will be used, microvault
	CA CAService `yaml:"ca"`
}

// HTTP configuration of the http service
type HTTP struct {
	// port of the http server
	Port int `yaml:"port"`
	// port of the https server
	Sslport int `yaml:"sslport"`
	// this is the url how to connect to this service from outside
	ServiceURL string `yaml:"serviceURL"`
	// other dns names (used for certificate)
	DNSNames []string `yaml:"dnss"`
	// other ips (used for certificate)
	IPAddresses []string `yaml:"ips"`
}

// CAService the micro-vault ca service config
type CAService struct {
	UseCA     bool   `yaml:"useca"`
	URL       string `yaml:"url"`
	AccessKey string `yaml:"accesskey"`
	Secret    string `yaml:"secret"`
}

// Authentication configuration
type Authentication struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

// Engine configuration
type Engine struct {
	RetentionManager string    `yaml:"retentionManager"`
	Tenantautoadd    bool      `yaml:"tenantautoadd"`
	BackupSyncmode   bool      `yaml:"backupsyncmode"`
	AllowTntBackup   bool      `yaml:"allowtntbackup"`
	Storage          Storage   `yaml:"storage"`
	Backup           Storage   `yaml:"backup"`
	Cache            Storage   `yaml:"cache"`
	Index            Storage   `yaml:"index"`
	Extractor        Extractor `yaml:"extractor"`
}

// Extractor defining config for full text extraction services
type Extractor struct {
	Service    string         `yaml:"service"`
	Properties map[string]any `yaml:"properties"`
}

// Storage configuration
type Storage struct {
	Storageclass string         `yaml:"storageclass"`
	Properties   map[string]any `yaml:"properties"`
}

// OpenTracing configuration
type OpenTracing struct {
	Host     string `yaml:"host"`
	Endpoint string `yaml:"endpoint"`
}

// Metrics configuration
type Metrics struct {
	Enable bool `yaml:"enable"`
}

// Metrics configuration
type Profiling struct {
	Enable bool `yaml:"enable"`
}

var defaultHeaderMapping = map[string]string{api.TenantHeaderKey: "X-tenant", api.RetentionHeaderKey: "X-retention", api.APIKeyHeaderKey: "X-apikey", api.FilenameKey: "X-filename", api.BlobIDHeaderKey: "X-blobid", api.HeaderPrefixKey: "X-"}

// DefaultConfig default configuration
var DefaultConfig = Config{
	Service: Service{
		HTTP: HTTP{
			Port:       8000,
			Sslport:    8443,
			ServiceURL: "https://127.0.0.1:8443",
		},
		HealthSystem: health.Config{
			Period:     30,
			StartDelay: 3,
		},
	},
	Apikey:     true,
	SecretFile: "",
	Logging: logging.LoggingConfig{
		Level:    "INFO",
		Filename: "${configdir}/logging.log",
	},
	Engine: Engine{
		RetentionManager: "SingleRetention",
		Tenantautoadd:    true,
		BackupSyncmode:   false,
		Storage: Storage{
			Storageclass: "SimpleFile",
			Properties: map[string]any{
				"rootpath": "./blbstg",
			},
		},
	},
	HeaderMapping: defaultHeaderMapping,
}

// GetDefaultConfigFolder returning the default configuration folder of the system
func GetDefaultConfigFolder() (string, error) {
	home, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	configFolder := filepath.Join(home, Servicename)
	err = os.MkdirAll(configFolder, os.ModePerm)
	if err != nil {
		return "", err
	}
	return configFolder, nil
}

// GetDefaultConfigfile getting the default config file
func GetDefaultConfigfile() (string, error) {
	configFolder, err := GetDefaultConfigFolder()
	if err != nil {
		return "", errors.Wrap(err, "can't load config file")
	}
	configFolder = filepath.Join(configFolder, "service")
	err = os.MkdirAll(configFolder, os.ModePerm)
	if err != nil {
		return "", errors.Wrap(err, "can't load config file")
	}
	return filepath.Join(configFolder, "service.yaml"), nil
}

// ReplaceConfigdir replace the configdir macro
func ReplaceConfigdir(s string) (string, error) {
	if strings.Contains(s, "${configdir}") {
		configFolder, err := GetDefaultConfigFolder()
		if err != nil {
			return "", err
		}
		return strings.Replace(s, "${configdir}", configFolder, -1), nil
	}
	return s, nil
}

var config = Config{}

// File the config file
var File = "${configdir}/service.yaml"

func init() {
	config = DefaultConfig
}

// Provide provide the config to the dependency injection
func (c *Config) Provide() {
	do.ProvideNamedValue[Config](nil, DoServiceConfig, *c)
}

// Get returns loaded config
func Get() Config {
	return config
}

// Load loads the config
func Load() error {
	myFile, err := ReplaceConfigdir(File)
	if err != nil {
		return fmt.Errorf("can't get default config folder: %s", err.Error())
	}
	File = myFile
	_, err = os.Stat(myFile)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(File)
	if err != nil {
		return fmt.Errorf("can't load config file: %s", err.Error())
	}
	dataStr, err := envsubst.EvalEnv(string(data))
	if err != nil {
		return fmt.Errorf("can't substitute config file: %s", err.Error())
	}

	err = yaml.Unmarshal([]byte(dataStr), &config)
	if err != nil {
		return fmt.Errorf("can't unmarshal config file: %s", err.Error())
	}

	for k, v := range defaultHeaderMapping {
		mp, ok := config.HeaderMapping[k]
		if !ok {
			config.HeaderMapping[k] = v
		}
		if ok && mp == "" {
			config.HeaderMapping[k] = v
		}
	}

	return readSecret()
}

func readSecret() error {
	secretFile := config.SecretFile
	if secretFile != "" {
		data, err := os.ReadFile(secretFile)
		if err != nil {
			return fmt.Errorf("can't load secret file: %s", err.Error())
		}
		var secretConfig Config
		err = yaml.Unmarshal(data, &secretConfig)
		if err != nil {
			return fmt.Errorf("can't unmarshal secret file: %s", err.Error())
		}
		// merge secret
		if err := mergo.Map(&config, secretConfig, mergo.WithOverride); err != nil {
			return fmt.Errorf("can't merge secret file: %s", err.Error())
		}
	}
	return nil
}
