package s3

/*
this file contains the logic to use a S3 server als backend for the blob storage.
*/
import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	fn_stlst        = "storelist.json"
	blobDescription = "Blobdescription"
)

type S3TenantManager struct {
	Endpoint   string
	Bucket     string
	AccessKey  string
	SecretKey  string
	Password   string
	minioCient minio.Client
	secure     bool
	local      bool
	storelist  []S3StoreEntry
}

type S3BlobStorage struct {
	Endpoint   string
	Bucket     string
	AccessKey  string
	SecretKey  string
	Tenant     string
	Password   string
	minioCient minio.Client
	secure     bool
	local      bool
}

func (s *S3TenantManager) Init() error {
	u, err := url.Parse(s.Endpoint)
	if err != nil {
		return err
	}
	endpoint := u.Host + "/" + u.Path
	s.secure = u.Scheme == "https"
	host, _, _ := net.SplitHostPort(u.Host)
	var options *minio.Options
	if host == "127.0.0.1" {
		options = &minio.Options{
			Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
			Secure: s.secure,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			},
		}
		s.local = true
	} else {
		options = &minio.Options{
			Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
			Secure: s.secure,
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
	stat, err := s.minioCient.StatObject(ctx, s.Bucket, fn_stlst, minio.StatObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return nil
			}
		}
		return err
	}
	fmt.Printf("%v", stat)
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
	uploadInfo, err := s.minioCient.PutObject(ctx, s.Bucket, fn_stlst, r, -1, minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/json",
	})
	if err != nil {
		return err
	}
	fmt.Printf("%v\r\n", uploadInfo)
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
	if !s.secure || s.local {
		return nil
	}
	return encrypt.DefaultPBKDF([]byte(s.Password), []byte(s.Bucket))
}

func (s *S3TenantManager) Close() error {
	return nil
}

//S3 Blob Storage
// initialise this dao
func (s *S3BlobStorage) Init() error {
	u, err := url.Parse(s.Endpoint)
	if err != nil {
		return err
	}
	endpoint := u.Host + "/" + u.Path
	s.secure = u.Scheme == "https"
	host, _, _ := net.SplitHostPort(u.Host)
	var options *minio.Options
	if host == "127.0.0.1" {
		options = &minio.Options{
			Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
			Secure: s.secure,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
			},
		}
		s.local = true
	} else {
		options = &minio.Options{
			Creds:  credentials.NewStaticV4(s.AccessKey, s.SecretKey, ""),
			Secure: s.secure,
		}
	}
	client, err := minio.New(endpoint, options)

	if err != nil {
		return err
	}
	s.minioCient = *client
	return nil
}

// getting a list of blob from the filesystem using offset and limit
func (s *S3BlobStorage) GetBlobs(offset int, limit int) ([]string, error) {
	return nil, errors.New("not yet implemented")
}

// CRUD operation on the blob files
// storing a blob to the storage system
func (s *S3BlobStorage) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	ctx := context.Background()
	uuid := uuid.NewString()
	b.BlobID = uuid
	metadatastr, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	metadata := make(map[string]string)
	metadata[blobDescription] = string(metadatastr)

	filename := s.id2f(uuid)
	uploadInfo, err := s.minioCient.PutObject(ctx, s.Bucket, filename, f, b.ContentLength, minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/octet-stream",
		UserMetadata:         metadata,
	})
	if err != nil {
		return "", err
	}
	fmt.Printf("%v\r\n", uploadInfo)
	return uuid, nil
}

// checking, if a blob is present
func (s *S3BlobStorage) HasBlob(id string) (bool, error) {
	filename := s.id2f(id)
	ctx := context.Background()
	_, err := s.minioCient.StatObject(ctx, s.Bucket, filename, minio.StatObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// getting the description of the file
func (s *S3BlobStorage) GetBlobDescription(id string) (*model.BlobDescription, error) {
	filename := s.id2f(id)
	ctx := context.Background()
	stat, err := s.minioCient.StatObject(ctx, s.Bucket, filename, minio.StatObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return nil, os.ErrNotExist
			}
		}
		return nil, err
	}
	metadata := stat.UserMetadata
	jsonstr, ok := metadata[blobDescription]
	if ok {
		var b model.BlobDescription
		err = json.Unmarshal([]byte(jsonstr), &b)
		if err != nil {
			return nil, err
		}
		return &b, nil
	}
	return nil, os.ErrNotExist
}

// retrieving the binary data from the storage system
func (s *S3BlobStorage) RetrieveBlob(id string, w io.Writer) error {
	filename := s.id2f(id)
	ctx := context.Background()
	r, err := s.minioCient.GetObject(ctx, s.Bucket, filename, minio.GetObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return os.ErrNotExist
			}
		}
		return err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}
	return nil
}

// removing a blob from the storage system
func (s *S3BlobStorage) DeleteBlob(id string) error {
	filename := s.id2f(id)
	ctx := context.Background()
	err := s.minioCient.RemoveObject(ctx, s.Bucket, filename, minio.RemoveObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return os.ErrNotExist
			}
		}
		return err
	}
	return nil
}

//Retentionrelated methods
// for every retention entry for this tenant we call this this function, you can stop the listing by returnong a false
func (s *S3BlobStorage) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	filename := s.id2rp()
	ctx := context.Background()
	objectCh := s.minioCient.ListObjects(ctx, s.Bucket, minio.ListObjectsOptions{
		Prefix:    filename,
		Recursive: false,
	})
	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return object.Err
		}
		r, err := s.getRetentionByFile(object.Key)
		if err == nil {
			callback(*r)
		}
	}
	return nil
}

func (s *S3BlobStorage) AddRetention(r *model.RetentionEntry) error {
	filename := s.id2rf(r.BlobID)
	ctx := context.Background()
	jsonstr, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f := bytes.NewReader(jsonstr)
	uploadInfo, err := s.minioCient.PutObject(ctx, s.Bucket, filename, f, int64(len(jsonstr)), minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/json",
	})
	if err != nil {
		return err
	}
	fmt.Printf("%v\r\n", uploadInfo)
	return nil
}

func (s *S3BlobStorage) DeleteRetention(id string) error {
	filename := s.id2rf(id)
	ctx := context.Background()
	err := s.minioCient.RemoveObject(ctx, s.Bucket, filename, minio.RemoveObjectOptions{})
	if err != nil {
		if errResp, ok := err.(minio.ErrorResponse); ok {
			if errResp.StatusCode == 404 {
				return os.ErrNotExist
			}
		}
		return err
	}
	return nil
}

func (s *S3BlobStorage) ResetRetention(id string) error {
	r, err := s.getRetention(id)
	if err != nil {
		return err
	}
	r.RetentionBase = int(time.Now().UnixNano() / 1000000)
	return s.AddRetention(r)
}

// closing the storage
func (s *S3BlobStorage) Close() error {
	return nil
}
func (s *S3BlobStorage) getRetention(id string) (*model.RetentionEntry, error) {
	filename := s.id2rf(id)
	return s.getRetentionByFile(filename)
}

func (s *S3BlobStorage) getRetentionByFile(filename string) (*model.RetentionEntry, error) {
	ctx := context.Background()
	r, err := s.minioCient.GetObject(ctx, s.Bucket, filename, minio.GetObjectOptions{
		ServerSideEncryption: s.getEncryption(),
	})
	if err != nil {
		return nil, err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)

	re := model.RetentionEntry{}
	err = json.Unmarshal(buf.Bytes(), &re)
	if err != nil {
		return nil, err
	}
	return &re, nil
}

//getEncryption here you get the ServerSide encryption for the service itself
func (s *S3BlobStorage) getEncryption() encrypt.ServerSide {
	if !s.secure || s.local {
		return nil
	}
	return encrypt.DefaultPBKDF([]byte(s.Password), []byte(s.Bucket+s.Tenant))
}

func (s *S3BlobStorage) id2f(id string) string {
	return fmt.Sprintf("%s/%s.bin", s.Tenant, id)
}

func (s *S3BlobStorage) id2rf(id string) string {
	return fmt.Sprintf("%s/retention/%s.json", s.Tenant, id)
}

func (s *S3BlobStorage) id2rp() string {
	return fmt.Sprintf("%s/retention/", s.Tenant)
}
