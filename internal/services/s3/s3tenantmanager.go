package s3

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
)

const (
	fnStlst = "storelist.json"
)

// TenantManager the s3 based tenant manager
type TenantManager struct {
	Endpoint    string
	Insecure    bool // true for self signed certificates
	Bucket      string
	AccessKey   string
	SecretKey   string
	Password    string
	minioClient minio.Client
	usetls      bool
	storelist   []StoreEntry
}

// Init initialize this tenant manager
func (s *TenantManager) Init() error {
	u, err := url.Parse(s.Endpoint)
	if err != nil {
		return err
	}
	endpoint := u.Host + "/" + u.Path
	s.usetls = u.Scheme == "https"
	var options *minio.Options
	if s.Insecure {
		options = &minio.Options{
			Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
			Secure: s.usetls,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			},
		}
	} else {
		options = &minio.Options{
			Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
			Secure: s.usetls,
		}
	}
	client, err := minio.New(endpoint, options)

	if err != nil {
		return err
	}
	s.minioClient = *client
	// check the bucket and try to create it
	ctx := context.Background()
	ok, err := s.minioClient.BucketExists(ctx, s.Bucket)
	if err != nil {
		return err
	}
	if !ok {
		err := s.minioClient.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: "us-east-1", ObjectLocking: false})
		if err != nil {
			return err
		}
	}
	return nil
}

// GetTenants walk thru all tenants
func (s *TenantManager) GetTenants(callback func(tenant string) bool) error {
	if s.storelist == nil {
		err := s.readStorelist()
		if err != nil {
			return err
		}
	}
	for _, tenant := range s.storelist {
		callback(tenant.Tenant)
	}
	return nil
}

func (s *TenantManager) readStorelist() error {
	ctx := context.Background()
	_, err := s.minioClient.StatObject(ctx, s.Bucket, fnStlst, minio.StatObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return nil
			}
		}
		return err
	}
	reader, err := s.minioClient.GetObject(ctx, s.Bucket, fnStlst, minio.GetObjectOptions{
		ServerSideEncryption: nil,
	})
	if err != nil {
		return err
	}
	var storeEntries []StoreEntry
	data, err := ioutil.ReadAll(reader)
	if err == nil && data != nil {
		err = json.Unmarshal(data, &storeEntries)
	}
	if err != nil {
		return err
	}
	s.storelist = storeEntries
	return nil
}

func (s *TenantManager) writeStorelist() error {
	ctx := context.Background()

	data, err := json.Marshal(s.storelist)
	if err != nil {
		return err
	}
	r := bytes.NewReader(data)
	_, err = s.minioClient.PutObject(ctx, s.Bucket, fnStlst, r, -1, minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/json",
	})
	if err != nil {
		return err
	}
	return nil
}

// AddTenant add a new tenant to the manager
func (s *TenantManager) AddTenant(tenant string) error {
	if s.HasTenant(tenant) {
		return errors.New("tenant already exists")
	}
	s.storelist = append(s.storelist, StoreEntry{
		Tenant: strings.ToLower(tenant),
	})
	err := s.writeStorelist()
	if err != nil {
		return err
	}

	return nil
}

// RemoveTenant remove a tenant from the service, delete all related data
func (s *TenantManager) RemoveTenant(tenant string) (string, error) {
	if !s.HasTenant(tenant) {
		return "", nil
	}
	tenant = strings.ToLower(tenant)
	index := -1
	for x, store := range s.storelist {
		if strings.EqualFold(store.Tenant, tenant) {
			index = x
		}
	}
	if index > -1 {
		s.storelist[index] = s.storelist[len(s.storelist)-1]
		s.storelist = s.storelist[:len(s.storelist)-1]
	}
	err := s.writeStorelist()
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	for object := range s.minioClient.ListObjects(ctx, s.Bucket, minio.ListObjectsOptions{Prefix: tenant, Recursive: true}) {
		s.minioClient.RemoveObject(ctx, s.Bucket, object.Key, minio.RemoveObjectOptions{ForceDelete: true})
	}

	return "", nil
}

// HasTenant checking if a tenant is present
func (s *TenantManager) HasTenant(tenant string) bool {
	tenant = strings.ToLower(tenant)
	for _, store := range s.storelist {
		if strings.EqualFold(store.Tenant, tenant) {
			return true
		}
	}
	return false
}

// SetConfig writing a new config object for the tenant
func (s *TenantManager) SetConfig(tenant string, config interfaces.TenantConfig) error {
	ctx := context.Background()
	cfnName := s.getConfigName(tenant)

	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	r := bytes.NewReader(data)
	_, err = s.minioClient.PutObject(ctx, s.Bucket, cfnName, r, -1, minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/json",
	})
	if err != nil {
		return err
	}
	return nil
}

// GetConfig reading the config object for the tenant
func (s *TenantManager) GetConfig(tenant string) (*interfaces.TenantConfig, error) {
	ctx := context.Background()
	cfnName := s.getConfigName(tenant)

	_, err := s.minioClient.StatObject(ctx, s.Bucket, cfnName, minio.StatObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return nil, nil
			}
		}
		return nil, err
	}
	reader, err := s.minioClient.GetObject(ctx, s.Bucket, cfnName, minio.GetObjectOptions{
		ServerSideEncryption: nil,
	})
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(reader)
	var cfn interfaces.TenantConfig
	if err == nil && data != nil {
		err = json.Unmarshal(data, &cfn)
	}
	if err != nil {
		return nil, err
	}
	return &cfn, nil
}

func (s *TenantManager) getConfigName(tenant string) string {
	tenant = strings.ToLower(tenant)
	return fmt.Sprintf("%s/%s/%s", tenant, "_config", "config.json")
}

// GetSize getting the overall storage size for a tenant, niy
func (s *TenantManager) GetSize(_ string) int64 {
	return 0
}

// AddSize adding the blob size to the tenant size
func (s *TenantManager) AddSize(_ string, _ int64) {
	// not implemented
}

// SubSize subtract the blob size to the tenant size
func (s *TenantManager) SubSize(tenant string, size int64) {
	// not implemented
}

func (s *TenantManager) getEncryption() encrypt.ServerSide {
	if !s.usetls || s.Insecure {
		return nil
	}
	return encrypt.DefaultPBKDF([]byte(s.Password), []byte(s.Bucket))
}

// Close closing the manager
func (s *TenantManager) Close() error {
	return nil
}
