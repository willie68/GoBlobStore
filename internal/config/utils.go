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
		return "", fmt.Errorf("config value for %s is not a string", key)
	}
	return value, nil
}

func GetConfigValueAsBool(stgCfng Storage, key string) (bool, error) {
	if _, ok := stgCfng.Properties[key]; !ok {
		return false, fmt.Errorf("missing config value for %s", key)
	}
	value, ok := stgCfng.Properties[key].(bool)
	if !ok {
		return false, fmt.Errorf("config value for %s is not a string", key)
	}
	return value, nil
}

func GetConfigValueAsInt(stgCfng Storage, key string) (int64, error) {
	if _, ok := stgCfng.Properties[key]; !ok {
		return 0, fmt.Errorf("missing config value for %s", key)
	}
	var value int64
	switch v := stgCfng.Properties[key].(type) {
	case int:
		value = int64(v)
	case int64:
		value = v
	default:
		return 0, fmt.Errorf("config value for %s is not a integer", key)
	}
	return value, nil
}
