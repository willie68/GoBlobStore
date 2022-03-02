package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/willie68/GoBlobStore/internal/api"
	"github.com/willie68/GoBlobStore/internal/config"
)

type JWTAuthConfig struct {
	Active      bool
	Validate    bool
	TenantClaim string
	Strict      bool
	RoleClaim   string
	RoleMapping map[string]string
}

type JWT struct {
	Token     string
	Header    map[string]interface{}
	Payload   map[string]interface{}
	Signature string
	IsValid   bool
}

type JWTAuth struct {
	Config JWTAuthConfig
}

// JWT config for the service
var JWTConfig JWTAuthConfig = JWTAuthConfig{
	Active: false,
}

func InitJWT(cnfg JWTAuthConfig) JWTAuth {
	JWTConfig = cnfg
	return JWTAuth{
		Config: cnfg,
	}
}

func ParseJWTConfig(cfg config.Authentication) (JWTAuthConfig, error) {
	jwtcfg := JWTAuthConfig{
		Active: true,
	}
	var err error
	jwtcfg.Validate, err = config.GetConfigValueAsBool(cfg.Properties, "validate")
	if err != nil {
		return jwtcfg, err
	}
	jwtcfg.Strict, err = config.GetConfigValueAsBool(cfg.Properties, "strict")
	if err != nil {
		return jwtcfg, err
	}
	jwtcfg.TenantClaim, err = config.GetConfigValueAsString(cfg.Properties, "tenantClaim")
	if err != nil {
		return jwtcfg, err
	}
	jwtcfg.RoleClaim, err = config.GetConfigValueAsString(cfg.Properties, "roleClaim")
	if err != nil {
		return jwtcfg, err
	}
	jwtcfg.RoleMapping = make(map[string]string)
	jwtcfg.RoleMapping[string(api.R_OBJECT_READER)] = "object-reader"
	jwtcfg.RoleMapping[string(api.R_OBJECT_CREATOR)] = "object-creator"
	jwtcfg.RoleMapping[string(api.R_OBJECT_ADMIN)] = "object-admin"
	jwtcfg.RoleMapping[string(api.R_TENANT_ADMIN)] = "tenant-admin"
	jwtcfg.RoleMapping[string(api.R_ADMIN)] = "admin"

	vm, ok := cfg.Properties["rolemapping"].(map[string]interface{})
	if ok {
		v, err := config.GetConfigValueAsString(vm, "object-reader")
		if err == nil && v != "" {
			jwtcfg.RoleMapping[string(api.R_OBJECT_READER)] = v
		}
		v, err = config.GetConfigValueAsString(vm, "object-creator")
		if err == nil && v != "" {
			jwtcfg.RoleMapping[string(api.R_OBJECT_CREATOR)] = v
		}
		v, err = config.GetConfigValueAsString(vm, "object-admin")
		if err == nil && v != "" {
			jwtcfg.RoleMapping[string(api.R_OBJECT_ADMIN)] = v
		}
		v, err = config.GetConfigValueAsString(vm, "tenant-admin")
		if err == nil && v != "" {
			jwtcfg.RoleMapping[string(api.R_TENANT_ADMIN)] = v
		}
		v, err = config.GetConfigValueAsString(vm, "admin")
		if err == nil && v != "" {
			jwtcfg.RoleMapping[string(api.R_ADMIN)] = v
		}
	}
	return jwtcfg, nil
}

func DecodeJWT(token string) (JWT, error) {
	jwt := JWT{
		Token:   token,
		IsValid: false,
	}

	if token == "" {
		return JWT{}, errors.New("missing token string")
	}

	if len(token) > 7 && strings.ToUpper(token[0:6]) == "BEARER" {
		token = token[7:]
	}

	// decode JWT token without verifying the signature
	jwtParts := strings.Split(token, ".")
	if len(jwtParts) < 2 {
		err := errors.New("token missing payload part")
		return jwt, err
	}
	var err error

	jwt.Header, err = jwtDecodePart(jwtParts[0])
	if err != nil {
		err = fmt.Errorf("token header parse error, %v", err)
		return jwt, err
	}

	jwt.Payload, err = jwtDecodePart(jwtParts[1])
	if err != nil {
		err = fmt.Errorf("token payload parse error, %v", err)
		return jwt, err
	}
	if len(jwtParts) > 2 {
		jwt.Signature = jwtParts[2]
	}
	jwt.IsValid = true
	return jwt, nil
}

func jwtDecodePart(payload string) (map[string]interface{}, error) {
	var result map[string]interface{}
	payloadData, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(payload)
	if err != nil {
		err = fmt.Errorf("token payload can't be decoded: %v", err)
		return nil, err
	}
	err = json.Unmarshal(payloadData, &result)
	if err != nil {
		err = fmt.Errorf("token payload parse error, %v", err)
		return nil, err
	}
	return result, nil
}

func (j *JWT) Validate(cfg JWTAuthConfig) error {
	//TODO here should be the implementation of the validation of the token
	return nil
}
