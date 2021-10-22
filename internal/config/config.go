package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config our service configuration
type Config struct {
	//port of the http server
	Port int `yaml:"port"`
	//port of the https server
	Sslport int `yaml:"sslport"`
	//this is the url how to connect to this service from outside
	ServiceURL string `yaml:"serviceURL"`

	SecretFile string `yaml:"secretfile"`

	HealthCheck HealthCheck `yaml:"healthcheck"`

	Logging LoggingConfig `yaml:"logging"`

	Storage Storage `yaml:"storage"`

	HeaderMapping map[string]string `yaml:"headermapping"`
}

type Storage struct {
	Storageclass     string                 `yaml:"storageclass"`
	Properties       map[string]interface{} `yaml:"properties"`
	RetentionManager string                 `yaml:"retentionManager"`
	Tenantautoadd    bool                   `yaml:"tenantautoadd"`
}

// HealthCheck configuration for the health check system
type HealthCheck struct {
	Period int `yaml:"period"`
}

type LoggingConfig struct {
	Level    string `yaml:"level"`
	Filename string `yaml:"filename"`
}

var defaultHeaderMapping = map[string]string{"tenant": "X-tenant", "retention": "X-retention", "apikey": "X-apikey", "filename": "X-filename", "headerprefix": "x-"}

var DefaultConfig = Config{
	Port:       8000,
	Sslport:    8443,
	ServiceURL: "https://127.0.0.1:8443",
	SecretFile: "",
	HealthCheck: HealthCheck{
		Period: 30,
	},
	Logging: LoggingConfig{
		Level:    "INFO",
		Filename: "${configdir}/logging.log",
	},
	Storage: Storage{
		Storageclass: "SimpleFile",
		Properties: map[string]interface{}{
			"rootpath": "./blbstg",
		},
		RetentionManager: "SingleRetention",
		Tenantautoadd:    true,
	},
	HeaderMapping: defaultHeaderMapping,
}

// GetDefaultConfigFolder returning the default configuration folder of the system
func GetDefaultConfigFolder() (string, error) {
	home, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	configFolder := fmt.Sprintf("%s/GoBlob", home)
	err = os.MkdirAll(configFolder, os.ModePerm)
	if err != nil {
		return "", err
	}
	return configFolder, nil
}

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

var config = Config{
	Port:       0,
	Sslport:    0,
	ServiceURL: "http://127.0.0.1",
	HealthCheck: HealthCheck{
		Period: 30,
	},
	HeaderMapping: defaultHeaderMapping,
}

// File the config file
var File = "config/service.yaml"

// Get returns loaded config
func Get() Config {
	return config
}

// Load loads the config
func Load() error {
	_, err := os.Stat(File)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(File)
	if err != nil {
		return fmt.Errorf("can't load config file: %s", err.Error())
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("can't unmarshal config file: %s", err.Error())
	}

	for k, v := range defaultHeaderMapping {
		if _, ok := config.HeaderMapping[k]; !ok {
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
		var secretConfig Secret = Secret{}
		err = yaml.Unmarshal(data, &secretConfig)
		if err != nil {
			return fmt.Errorf("can't unmarshal secret file: %s", err.Error())
		}
		mergeSecret(secretConfig)
	}
	return nil
}

func mergeSecret(secret Secret) {
	//	config.MongoDB.Username = secret.MongoDB.Username
	//	config.MongoDB.Password = secret.MongoDB.Password
}
