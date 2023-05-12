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
	//t.SkipNow()
	ctx := context.Background()
	endpoint := "s3.tms.proactcloud.de:443"
	accessKeyID := "J6UTM424GSEQUF28OC9N"
	secretAccessKey := "qn0hJj5F3XEVrayBLPM5wbHZMQ5WDMtqlKO1AzAo"
	useSSL := true

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
		t.Fatal(err)
	}

	log.Printf("%#v\n", minioClient) // minioClient is now set up

	log.Printf("%#v\n", minioClient.IsOnline())
	// Make a new bucket called mymusic.
	bucketName := "dev-test"
	//location := "us-east-1"

	//	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: location})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			t.Fatal(err)
		}
	} else {
		log.Printf("Successfully created %s\n", bucketName)
	}

	cctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	objectCh := minioClient.ListObjects(cctx, bucketName, minio.ListObjectsOptions{
		Prefix:    "",
		Recursive: true,
	})
	for object := range objectCh {
		if object.Err != nil {
			cancel()
			break
		}
		id := object.Key
		log.Printf("obj %s: %v", id, object)
	}

	// Upload the zip file
	objectName := "pdf.pdf"
	filePath := "../testdata/pdf.pdf"
	contentType := "text/markdown"

	// Upload the zip file with FPutObject
	info, err := minioClient.FPutObject(ctx, bucketName, objectName, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		t.Fatal(err)
	}

	log.Printf("Successfully uploaded %s of size %d\n", objectName, info.Size)
}
