package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/drone/envsubst"
	"github.com/imdario/mergo"
	"github.com/willie68/GoBlobStore/internal/api"
	"gopkg.in/yaml.v3"
)

// Servicename Name of the service
const Servicename = "goblob-service"

// Config our service configuration
type Config struct {
	//port of the http server
	Port int `yaml:"port"`
	//port of the https server
	Sslport int `yaml:"sslport"`
	//this is the url how to connect to this service from outside
	ServiceURL string `yaml:"serviceURL"`

	SecretFile string `yaml:"secretfile"`

	Apikey bool `yaml:"apikey"`

	Logging LoggingConfig `yaml:"logging"`

	HealthCheck HealthCheck `yaml:"healthcheck"`

	Auth Authentication `yaml:"auth"`

	Engine Engine `yaml:"engine"`

	HeaderMapping map[string]string `yaml:"headermapping"`

	OpenTracing OpenTracing `yaml:"opentracing"`

	Metrics Metrics `yaml:"metrics"`
}

// Authentication configuration
type Authentication struct {
	Type       string         `yaml:"type"`
	Properties map[string]any `yaml:"properties"`
}

// Engine configuration
type Engine struct {
	RetentionManager string  `yaml:"retentionManager"`
	Tenantautoadd    bool    `yaml:"tenantautoadd"`
	BackupSyncmode   bool    `yaml:"backupsyncmode"`
	AllowTntBackup   bool    `yaml:"allowtntbackup"`
	Storage          Storage `yaml:"storage"`
	Backup           Storage `yaml:"backup"`
	Cache            Storage `yaml:"cache"`
	Index            Storage `yaml:"index"`
}

// Storage configuration
type Storage struct {
	Storageclass string         `yaml:"storageclass"`
	Properties   map[string]any `yaml:"properties"`
}

// HealthCheck configuration for the health check system
type HealthCheck struct {
	Period int `yaml:"period"`
}

// LoggingConfig configuration for the gelf logging
type LoggingConfig struct {
	Level    string `yaml:"level"`
	Filename string `yaml:"filename"`

	Gelfurl  string `yaml:"gelf-url"`
	Gelfport int    `yaml:"gelf-port"`
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

var defaultHeaderMapping = map[string]string{api.TenantHeaderKey: "X-tenant", api.RetentionHeaderKey: "X-retention", api.APIKeyHeaderKey: "X-apikey", api.FilenameKey: "X-filename", api.BlobIDHeaderKey: "X-blobid", api.HeaderPrefixKey: "X-"}

// DefaultConfig default configuration
var DefaultConfig = Config{
	Port:       8000,
	Sslport:    8443,
	ServiceURL: "https://127.0.0.1:8443",
	SecretFile: "",
	Apikey:     true,
	HealthCheck: HealthCheck{
		Period: 30,
	},
	Logging: LoggingConfig{
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
	data, err := ioutil.ReadFile(File)
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
		data, err := ioutil.ReadFile(secretFile)
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
