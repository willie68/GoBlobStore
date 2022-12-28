package simplefile

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
)

type SimpleFileTenantManager struct {
	RootPath string // this is the root path for the file system storage
}

var _ interfaces.TenantDao = &SimpleFileTenantManager{}

func (s *SimpleFileTenantManager) Init() error {
	err := os.MkdirAll(s.RootPath, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (s *SimpleFileTenantManager) GetTenants(callback func(tenant string) bool) error {
	infos, err := ioutil.ReadDir(s.RootPath)
	if err != nil {
		return err
	}
	for _, i := range infos {
		if !strings.HasPrefix(i.Name(), "_") {
			ok := callback(i.Name())
			if !ok {
				return nil
			}
		}
	}
	return nil
}

func (s *SimpleFileTenantManager) AddTenant(tenant string) error {

	tenantPath := filepath.Join(s.RootPath, tenant)

	err := os.MkdirAll(tenantPath, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (s *SimpleFileTenantManager) RemoveTenant(tenant string) (string, error) {
	if !s.HasTenant(tenant) {
		return "", errors.New("tenant not exists")
	}
	tenantPath := filepath.Join(s.RootPath, tenant)
	err := os.RemoveAll(tenantPath)
	if err != nil {
		return "", err
	}
	return "", nil
}

func (s *SimpleFileTenantManager) HasTenant(tenant string) bool {
	tenantPath := filepath.Join(s.RootPath, tenant)

	if _, err := os.Stat(tenantPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// SetConfig writing a new config object for the tenant
func (s *SimpleFileTenantManager) SetConfig(tenant string, config interfaces.TenantConfig) error {
	cfnName := s.getConfigName(tenant)
	err := os.MkdirAll(filepath.Dir(cfnName), os.ModePerm)
	if err != nil {
		return err
	}
	str, err := json.Marshal(config)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(cfnName, str, os.ModePerm)
	return err
}

// GetConfig reading the config object for the tenant
func (s *SimpleFileTenantManager) GetConfig(tenant string) (*interfaces.TenantConfig, error) {
	cfnName := s.getConfigName(tenant)
	if _, err := os.Stat(cfnName); os.IsNotExist(err) {
		return nil, nil
	}
	data, err := ioutil.ReadFile(cfnName)
	if err != nil {
		return nil, err
	}

	var cfn interfaces.TenantConfig
	err = json.Unmarshal(data, &cfn)
	if err != nil {
		return nil, err
	}
	return &cfn, nil
}

func (s *SimpleFileTenantManager) getConfigName(tenant string) string {
	return filepath.Join(s.RootPath, tenant, "_config", "config.json")
}

func (s *SimpleFileTenantManager) GetSize(tenant string) int64 {
	if !s.HasTenant(tenant) {
		return -1
	}
	tenantPath := filepath.Join(s.RootPath, tenant)

	if _, err := os.Stat(tenantPath); os.IsNotExist(err) {
		return 0
	}

	var dirSize int64 = 0
	readSize := func(path string, file os.FileInfo, err error) error {
		if !file.IsDir() {
			dirSize += file.Size()
		}
		return nil
	}

	filepath.Walk(tenantPath, readSize)
	return dirSize
}

func (s *SimpleFileTenantManager) Close() error {
	return nil
}
