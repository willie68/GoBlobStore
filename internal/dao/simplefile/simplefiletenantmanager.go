package simplefile

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
)

// TenantManager the tenant manager based on a simple file storage system
type TenantManager struct {
	RootPath    string // this is the root path for the file system storage
	TenantInfos sync.Map
	calcRunning bool
}

// TenantInfo entry for tenant list
type TenantInfo struct {
	ID   string
	Size int64
}

// checking interface compatibility
var _ interfaces.TenantManager = &TenantManager{}

// Init intialise this tenant manager
func (s *TenantManager) Init() error {
	// checking the file system
	err := os.MkdirAll(s.RootPath, os.ModePerm)
	if err != nil {
		return err
	}
	s.calcRunning = false
	// background task for calculating the storage size of every tenant
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			if !s.calcRunning {
				go s.calculateAllStorageSizes()
			}
		}
	}()
	return nil
}

// GetTenants walk thru all tenants
func (s *TenantManager) GetTenants(callback func(tenant string) bool) error {
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

// AddTenant add a new tenant to the manager
func (s *TenantManager) AddTenant(tenant string) error {

	tenantPath := filepath.Join(s.RootPath, tenant)

	err := os.MkdirAll(tenantPath, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// RemoveTenant remove a tenant from the service, delete all related data
func (s *TenantManager) RemoveTenant(tenant string) (string, error) {
	if !s.HasTenant(tenant) {
		return "", errors.New("tenant not exists")
	}
	tenantPath := filepath.Join(s.RootPath, tenant)
	err := os.RemoveAll(tenantPath)
	if err != nil {
		return "", err
	}
	s.TenantInfos.Delete(tenant)
	return "", nil
}

// HasTenant checking is a tenant is created
func (s *TenantManager) HasTenant(tenant string) bool {
	tenantPath := filepath.Join(s.RootPath, tenant)

	if _, err := os.Stat(tenantPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// SetConfig writing a new config object for the tenant
func (s *TenantManager) SetConfig(tenant string, config interfaces.TenantConfig) error {
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
func (s *TenantManager) GetConfig(tenant string) (*interfaces.TenantConfig, error) {
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

func (s *TenantManager) getConfigName(tenant string) string {
	return filepath.Join(s.RootPath, tenant, "_config", "config.json")
}

// GetSize getting the overall storage size for a tenant
func (s *TenantManager) GetSize(tenant string) int64 {
	if !s.HasTenant(tenant) {
		return -1
	}
	info, ok := s.TenantInfos.Load(tenant)
	if !ok {
		return -1
	}
	tinfo, ok := info.(TenantInfo)
	if !ok {
		return -1
	}
	return tinfo.Size
}

func (s *TenantManager) calculateAllStorageSizes() {
	log.Logger.Debug("calculating storage sizes of all tenants")
	s.calcRunning = true
	defer func() { s.calcRunning = false }()
	s.GetTenants(func(tenant string) bool {
		var tinfo TenantInfo
		size := s.calculateStorageSize(tenant)
		tinfo = TenantInfo{
			ID:   tenant,
			Size: size,
		}
		s.TenantInfos.Store(tenant, tinfo)
		return true
	})
}

func (s *TenantManager) calculateStorageSize(tenant string) int64 {
	if !s.HasTenant(tenant) {
		return -1
	}
	tenantPath := filepath.Join(s.RootPath, tenant)

	if _, err := os.Stat(tenantPath); os.IsNotExist(err) {
		return 0
	}

	var dirSize int64
	readSize := func(path string, file os.FileInfo, err error) error {
		if !file.IsDir() {
			dirSize += file.Size()
		}
		return nil
	}

	filepath.Walk(tenantPath, readSize)
	return dirSize
}

// Close closing this dao
func (s *TenantManager) Close() error {
	return nil
}
