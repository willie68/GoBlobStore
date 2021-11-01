package dao

import "fmt"

func getConfigValueAsString(key string) (string, error) {
	if _, ok := cnfg.Properties[key]; !ok {
		return "", fmt.Errorf("missing config value for %s", key)
	}
	value, ok := cnfg.Properties[key].(string)
	if !ok {
		return "", fmt.Errorf("config value for %s is not a string", "endpoint")
	}
	return value, nil
}

func getConfigValueAsBool(key string) (bool, error) {
	if _, ok := cnfg.Properties[key]; !ok {
		return false, fmt.Errorf("missing config value for %s", key)
	}
	value, ok := cnfg.Properties[key].(bool)
	if !ok {
		return false, fmt.Errorf("config value for %s is not a string", "endpoint")
	}
	return value, nil
}
