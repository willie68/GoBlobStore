package s3

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

const (
	fn_stlst = "storelist.json"
)

type S3TenantManager struct {
	Endpoint   string
	Insecure   bool
	Bucket     string
	AccessKey  string
	SecretKey  string
	Password   string
	minioCient minio.Client
	usetls     bool
	storelist  []S3StoreEntry
}

func (s *S3TenantManager) Init() error {
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
	s.minioCient = *client
	return nil
}

func (s *S3TenantManager) GetTenants(callback func(tenant string) bool) error {
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

func (s *S3TenantManager) readStorelist() error {
	ctx := context.Background()
	_, err := s.minioCient.StatObject(ctx, s.Bucket, fn_stlst, minio.StatObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return nil
			}
		}
		return err
	}
	reader, err := s.minioCient.GetObject(ctx, s.Bucket, fn_stlst, minio.GetObjectOptions{
		ServerSideEncryption: nil,
	})
	if err != nil {
		return err
	}
	var storeEntries []S3StoreEntry
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

func (s *S3TenantManager) writeStorelist() error {
	ctx := context.Background()

	data, err := json.Marshal(s.storelist)
	if err != nil {
		return err
	}
	r := bytes.NewReader(data)
	_, err = s.minioCient.PutObject(ctx, s.Bucket, fn_stlst, r, -1, minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/json",
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *S3TenantManager) AddTenant(tenant string) error {
	if s.HasTenant(tenant) {
		return errors.New("tenant already exists")
	}
	s.storelist = append(s.storelist, S3StoreEntry{
		Tenant: strings.ToLower(tenant),
	})
	err := s.writeStorelist()
	if err != nil {
		return err
	}

	return nil
}

func (s *S3TenantManager) RemoveTenant(tenant string) error {
	if !s.HasTenant(tenant) {
		return nil
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
		return err
	}
	return nil
}

func (s *S3TenantManager) HasTenant(tenant string) bool {
	for _, store := range s.storelist {
		if strings.EqualFold(store.Tenant, tenant) {
			return true
		}
	}
	return false
}

func (s *S3TenantManager) GetSize(tenant string) int64 {
	return 0
}

func (s *S3TenantManager) getEncryption() encrypt.ServerSide {
	if !s.usetls || s.Insecure {
		return nil
	}
	return encrypt.DefaultPBKDF([]byte(s.Password), []byte(s.Bucket))
}

func (s *S3TenantManager) Close() error {
	return nil
}
