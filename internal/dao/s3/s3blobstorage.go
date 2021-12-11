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
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/willie68/GoBlobStore/internal/dao/interfaces"
	log "github.com/willie68/GoBlobStore/internal/logging"
	"github.com/willie68/GoBlobStore/internal/utils"
	"github.com/willie68/GoBlobStore/pkg/model"
)

const (
	blobDescription = "Blobdescription"
)

type S3BlobStorage struct {
	Endpoint   string
	Insecure   bool
	Bucket     string
	AccessKey  string
	SecretKey  string
	Tenant     string
	Password   string
	minioCient minio.Client
	usetls     bool
}

var _ interfaces.BlobStorageDao = &S3BlobStorage{}

//S3 Blob Storage
// initialise this dao
func (s *S3BlobStorage) Init() error {
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
	s.minioCient = *client
	// check the bucket and try to create it
	ctx := context.Background()
	ok, err := s.minioCient.BucketExists(ctx, s.Bucket)
	if err != nil {
		return err
	}
	if !ok {
		err := s.minioCient.MakeBucket(ctx, s.Bucket, minio.MakeBucketOptions{Region: "us-east-1", ObjectLocking: false})
		if err != nil {
			return err
		}
	}

	return nil
}

// GetTenant return the id of the tenant
func (s *S3BlobStorage) GetTenant() string {
	return s.Tenant
}

// getting a list of blob from the filesystem using offset and limit
func (s *S3BlobStorage) GetBlobs(callback func(id string) bool) error {
	return errors.New("not yet implemented")
}

// CRUD operation on the blob files
// storing a blob to the storage system
func (s *S3BlobStorage) StoreBlob(b *model.BlobDescription, f io.Reader) (string, error) {
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
	_, err = s.minioCient.PutObject(ctx, s.Bucket, filename, f, b.ContentLength, minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/octet-stream",
		UserMetadata:         metadata,
	})
	if err != nil {
		return "", err
	}
	return b.BlobID, nil
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

// Retentionrelated methods
//GetAllRetentions for every retention entry for this tenant we call the callback function,
// you can stop the walk by returning a false in the callback
func (s *S3BlobStorage) GetAllRetentions(callback func(r model.RetentionEntry) bool) error {
	filename := s.tntrp()
	ctx := context.Background()
	objectCh := s.minioCient.ListObjects(ctx, s.Bucket, minio.ListObjectsOptions{
		Prefix:    filename,
		Recursive: false,
	})
	for object := range objectCh {
		if object.Err != nil {
			log.Logger.Errorf("S3BlobStorage: unknown error on listfiles: %v", object.Err)
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
func (s *S3BlobStorage) AddRetention(r *model.RetentionEntry) error {
	filename := s.id2rf(r.BlobID)
	ctx := context.Background()
	jsonstr, err := json.Marshal(r)
	if err != nil {
		return err
	}
	f := bytes.NewReader(jsonstr)
	_, err = s.minioCient.PutObject(ctx, s.Bucket, filename, f, int64(len(jsonstr)), minio.PutObjectOptions{
		ServerSideEncryption: s.getEncryption(),
		ContentType:          "application/json",
	})
	if err != nil {
		return err
	}
	return nil
}

// GetRetention getting a single retention entry
func (s *S3BlobStorage) GetRetention(id string) (model.RetentionEntry, error) {
	r, err := s.getRetention(id)
	return *r, err
}

//DeleteRetention deletes the retention entry from the storaage
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

// ResetRetention resets the retention for a blob
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

// getting the retention entry for a id
func (s *S3BlobStorage) getRetention(id string) (*model.RetentionEntry, error) {
	filename := s.id2rf(id)
	return s.getRetentionByFile(filename)
}

//getRetentionByFile get a retention entry for filename
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

//getEncryption here you get the ServerSide encryption for the tenant
func (s *S3BlobStorage) getEncryption() encrypt.ServerSide {
	if !s.usetls || s.Insecure {
		return nil
	}
	return encrypt.DefaultPBKDF([]byte(s.Password), []byte(s.Bucket+s.Tenant))
}

// id2f getting the blob file path and name to the payload
func (s *S3BlobStorage) id2f(id string) string {
	return fmt.Sprintf("%s/%s.bin", s.Tenant, id)
}

// id2rf getting the retention file path and name for an id
func (s *S3BlobStorage) id2rf(id string) string {
	return fmt.Sprintf("%s/retention/%s.json", s.Tenant, id)
}

//tntrp getting the path to the retention files
func (s *S3BlobStorage) tntrp() string {
	return fmt.Sprintf("%s/retention/", s.Tenant)
}
