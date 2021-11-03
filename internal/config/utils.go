package config

import (
	"fmt"
)

func GetConfigValueAsString(stgCfng Storage, key string) (string, error) {
	if _, ok := stgCfng.Properties[key]; !ok {
		return "", fmt.Errorf("missing config value for %s", key)
	}
	value, ok := stgCfng.Properties[key].(string)
	if !ok {
		return "", fmt.Errorf("config value for %s is not a string", "endpoint")
	}
	return value, nil
}

func GetConfigValueAsBool(stgCfng Storage, key string) (bool, error) {
	if _, ok := stgCfng.Properties[key]; !ok {
		return false, fmt.Errorf("missing config value for %s", key)
	}
	value, ok := stgCfng.Properties[key].(bool)
	if !ok {
		return false, fmt.Errorf("config value for %s is not a string", "endpoint")
	}
	return value, nil
}
