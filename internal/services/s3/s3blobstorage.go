// Package s3 this package contains all s3 related structs
package s3

// this file contains the logic to use a S3 server als backend for the blob storage.
import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/willie68/GoBlobStore/internal/services/interfaces"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	blobDescription = "Blobdescription"
)

// BlobStorage service for storing blob files into a S3 compatible storage
type BlobStorage struct {
	Endpoint    string
	Insecure    bool
	Bucket      string
	AccessKey   string
	SecretKey   string
	Tenant      string
	Password    string
	minioClient minio.Client
	usetls      bool
}

var _ interfaces.BlobStorage = &BlobStorage{}

// S3 Blob Storage

// Init initialise this service
func (s *BlobStorage) Init() error {
	if s.Tenant == "" {
		return errors.New("tenant should not be null or empty")
	}
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

// GetTenant return the id of the tenant
func (s *BlobStorage) GetTenant() string {
	return s.Tenant
}

// GetBlobs getting a list of blob from the storage
func (s *BlobStorage) GetBlobs(callback func(id string) bool) error {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	objectCh := s.minioClient.ListObjects(ctx, s.Bucket, minio.ListObjectsOptions{
		Prefix:    s.Tenant,
		Recursive: true,
	})
	for object := range objectCh {
		if object.Err != nil {
			cancel()
			return object.Err
		}
		id := object.Key
		id = strings.TrimPrefix(id, s.Tenant+"/")
		id = strings.TrimSuffix(id, ".bin")
		next := callback(id)
		if !next {
			cancel()
			break
		}
	}
	return nil
}

// CRUD operation on the blob files

// StoreBlob storing a blob to the storage system
func (s *BlobStorage) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
	ctx := context.Background()
	if b.BlobID == "" {
		uuid := utils.GenerateID()
		b.BlobID = uuid
	}
	metadatastr, err := json.Marshal(b)
	if err != nil {
		return "", err
	}
	metadata := make(map[string]string)
	metadata[blobDescription] = string(metadatastr)

	filename := s.id2f(b.BlobID)
	_, err = s.minioClient.PutObject(ctx, s.Bucket, filename, f, b.ContentLength, minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/octet-stream",
		UserMetadata:         metadata,
	})
	if err != nil {
		return "", err
	}
	return b.BlobID, nil
}

// UpdateBlobDescription updating the blob description
func (s *BlobStorage) UpdateBlobDescription(_ string, b *model.BlobDescription) error {
	metadatastr, err := json.Marshal(b)
	if err != nil {
		return err
	}
	metadata := make(map[string]string)
	metadata[blobDescription] = string(metadatastr)

	filename := s.id2f(b.BlobID)

	srcOpts := minio.CopySrcOptions{
		Bucket:     s.Bucket,
		Object:     filename,
		Encryption: s.getEncryption(),
	}

	// Destination object
	dstOpts := minio.CopyDestOptions{
		Bucket:          s.Bucket,
		Object:          filename,
		Encryption:      s.getEncryption(),
		UserMetadata:    metadata,
		ReplaceMetadata: true,
	}

	// Copy object call
	_, err = s.minioClient.CopyObject(context.Background(), dstOpts, srcOpts)
	if err != nil {
		return err
	}
	return nil
}

// HasBlob checking, if a blob is present
func (s *BlobStorage) HasBlob(id string) (bool, error) {
	filename := s.id2f(id)
	ctx := context.Background()
	_, err := s.minioClient.StatObject(ctx, s.Bucket, filename, minio.StatObjectOptions{})
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

// GetBlobDescription getting the description of the file
func (s *BlobStorage) GetBlobDescription(id string) (*model.BlobDescription, error) {
	filename := s.id2f(id)
	ctx := context.Background()
	stat, err := s.minioClient.StatObject(ctx, s.Bucket, filename, minio.StatObjectOptions{})
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

// RetrieveBlob retrieving the binary data from the storage system
func (s *BlobStorage) RetrieveBlob(id string, w io.Writer) error {
	filename := s.id2f(id)
	ctx := context.Background()
	r, err := s.minioClient.GetObject(ctx, s.Bucket, filename, minio.GetObjectOptions{})
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

// DeleteBlob removing a blob from the storage system
func (s *BlobStorage) DeleteBlob(id string) error {
	filename := s.id2f(id)
	ctx := context.Background()
	err := s.minioClient.RemoveObject(ctx, s.Bucket, filename, minio.RemoveObjectOptions{})
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

// CheckBlob checking a single blob from the storage system
func (s *BlobStorage) CheckBlob(id string) (*model.CheckInfo, error) {
	return utils.CheckBlob(id, s)
}

// SearchBlobs quering a single blob, niy
func (s *BlobStorage) SearchBlobs(_ string, _ func(id string) bool) error {
	return errors.New("not implemented yet")
}

// Retentionrelated methods

// GetAllRetentions for every retention entry for this tenant we call the callback function,
// you can stop the walk by returning a false in the callback
func (s *BlobStorage) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	filename := s.tntrp()
	ctx := context.Background()
	objectCh := s.minioClient.ListObjects(ctx, s.Bucket, minio.ListObjectsOptions{
		Prefix:    filename,
		Recursive: false,
	})
	for object := range objectCh {
		if object.Err != nil {
			logger.Errorf("S3BlobStorage: unknown error on listfiles: %v", object.Err)
			return object.Err
		}
		r, err := s.getRetentionByFile(object.Key)
		if err == nil {
			proceed := callback(*r)
			if !proceed {
				break
			}
		}
	}
	return nil
}

// AddRetention adding a retention entry to the storage
func (s *BlobStorage) AddRetention(r *model.RetentionEntry) error {
	filename := s.id2rf(r.BlobID)
	ctx := context.Background()
	jsonstr, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f := bytes.NewReader(jsonstr)
	_, err = s.minioClient.PutObject(ctx, s.Bucket, filename, f, int64(len(jsonstr)), minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/json",
	})
	if err != nil {
		return err
	}
	return nil
}

// GetRetention getting a single retention entry
func (s *BlobStorage) GetRetention(id string) (model.RetentionEntry, error) {
	r, err := s.getRetention(id)
	return *r, err
}

// DeleteRetention deletes the retention entry from the storage
func (s *BlobStorage) DeleteRetention(id string) error {
	filename := s.id2rf(id)
	ctx := context.Background()
	err := s.minioClient.RemoveObject(ctx, s.Bucket, filename, minio.RemoveObjectOptions{})
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

// ResetRetention resets the retention for a blob
func (s *BlobStorage) ResetRetention(id string) error {
	r, err := s.getRetention(id)
	if err != nil {
		return err
	}
	r.RetentionBase = time.Now().UnixMilli()
	return s.AddRetention(r)
}

// GetLastError returning the last error (niy)
func (s *BlobStorage) GetLastError() error {
	return nil
}

// Close closing the storage
func (s *BlobStorage) Close() error {
	return nil
}

// getting the retention entry for a id
func (s *BlobStorage) getRetention(id string) (*model.RetentionEntry, error) {
	filename := s.id2rf(id)
	return s.getRetentionByFile(filename)
}

// getRetentionByFile get a retention entry for filename
func (s *BlobStorage) getRetentionByFile(filename string) (*model.RetentionEntry, error) {
	ctx := context.Background()
	r, err := s.minioClient.GetObject(ctx, s.Bucket, filename, minio.GetObjectOptions{
		ServerSideEncryption: s.getEncryption(),
	})
	if err != nil {
		return nil, err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	re := model.RetentionEntry{}
	err = json.Unmarshal(buf.Bytes(), &re)
	if err != nil {
		return nil, err
	}
	return &re, nil
}

// getEncryption here you get the ServerSide encryption for the tenant
func (s *BlobStorage) getEncryption() encrypt.ServerSide {
	if !s.usetls || s.Insecure {
		return nil
	}
	ss := encrypt.DefaultPBKDF([]byte(s.Password), []byte(s.Bucket+s.Tenant))
	return ss
}

// id2f getting the blob file path and name to the payload
func (s *BlobStorage) id2f(id string) string {
	return fmt.Sprintf("%s/%s.bin", s.Tenant, id)
}

// id2rf getting the retention file path and name for an id
func (s *BlobStorage) id2rf(id string) string {
	return fmt.Sprintf("%s/retention/%s.json", s.Tenant, id)
}

// tntrp getting the path to the retention files
func (s *BlobStorage) tntrp() string {
	return fmt.Sprintf("%s/retention/", s.Tenant)
}
