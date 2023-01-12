package ideas

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestMinio(t *testing.T) {
	t.SkipNow()
	ctx := context.Background()
	endpoint := "127.0.0.1:9002"
	accessKeyID := "D9Q2D6JQGW1MVCC98LQL"
	secretAccessKey := "LDX7QHY/IsNiA9DbdycGMuOP0M4khr0+06DKrFAr"
	useSSL := false

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("%#v\n", minioClient) // minioClient is now set up

	// Make a new bucket called mymusic.
	bucketName := "mymusic"
	location := "us-east-1"

	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			log.Fatalln(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	// Upload the zip file
	objectName := "pdf.pdf"
	filePath := "../testdata/pdf.pdf"
	contentType := "text/markdown"

	// Upload the zip file with FPutObject
	info, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Successfully uploaded %s of size %d\n", objectName, info.Size)
}
