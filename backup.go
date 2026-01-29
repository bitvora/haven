package main

import (
	"archive/zip"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func backupDatabase(ctx context.Context) {
	if config.BackupProvider == "none" || config.BackupProvider == "" {
		log.Println("🚫 no backup provider set")
		return
	}

	ticker := time.NewTicker(time.Duration(config.BackupIntervalHours) * time.Hour)
	defer ticker.Stop()

	zipFileName := "db.zip"
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if err := ZipDirectory("db", zipFileName); err != nil {
				log.Println("🚫 error zipping database folder:", err)
				continue
			}
			switch config.BackupProvider {
			case "s3":
				S3Upload(ctx, zipFileName)
			case "aws":
				AwsUpload(ctx, zipFileName)
			case "gcp":
				GCPBucketUpload(ctx, zipFileName)
			default:
				log.Println("🚫 we only support AWS, GCP, and S3 at this time")
			}
		}
	}
}

// Deprecated: Use S3Upload instead
//
//goland:noinspection GoUnhandledErrorResult
func GCPBucketUpload(ctx context.Context, zipFileName string) {
	if config.GcpConfig == nil {
		log.Fatal("🚫 GCP specified as backup provider but no GCP config found. Check environment variables.")
	}

	bucket := config.GcpConfig.Bucket

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// open the zip db file.
	f, err := os.Open(zipFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	obj := client.Bucket(bucket).Object(zipFileName)

	// Upload an object with storage.Writer.
	wc := obj.NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		log.Fatal(err)
	}

	if err := wc.Close(); err != nil {
		log.Fatal(err)
	}

	log.Printf("✅ Successfully uploaded %q to %q\n", zipFileName, bucket)

	// delete the file.
	err = os.Remove(zipFileName)
	if err != nil {
		log.Fatal(err)
	}
}

// Deprecated: Use S3Upload instead
//
//goland:noinspection GoUnhandledErrorResult
func AwsUpload(ctx context.Context, zipFileName string) {
	if config.AwsConfig == nil {
		log.Fatal("🚫 AWS specified as backup provider but no AWS config found. Check environment variables.")
	}

	s3UploadShared(
		ctx,
		zipFileName,
		config.AwsConfig.AccessKeyID,
		config.AwsConfig.SecretAccessKey,
		"s3.amazonaws.com",
		config.AwsConfig.Region,
		config.AwsConfig.Bucket,
		true,
	)
}

func S3Upload(ctx context.Context, zipFileName string) {
	if config.S3Config == nil {
		log.Fatal("🚫 S3 specified as backup provider but no S3 config found. Check environment variables.")
	}

	s3UploadShared(
		ctx,
		zipFileName,
		config.S3Config.AccessKeyID,
		config.S3Config.SecretKey,
		config.S3Config.Endpoint,
		config.S3Config.Region,
		config.S3Config.BucketName,
		true,
	)
}

func s3UploadShared(
	ctx context.Context,
	zipFileName string,
	accessKey string,
	secret string,
	endpoint string,
	region string,
	bucketName string,
	secure bool,
) {
	log.Println("🚀 uploading to S3 Bucket...")

	// Create MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secret, ""),
		Region: region,
		Secure: secure,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Upload the file to the S3 bucket
	file, err := os.Open(zipFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			log.Println("🚫 error closing db zip file:", err)
		}
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.PutObject(
		ctx,
		bucketName,
		zipFileName,
		file,
		fileInfo.Size(),
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("✅ Successfully uploaded %q to %q\n", zipFileName, bucketName)

	// delete the file
	err = os.Remove(zipFileName)
	if err != nil {
		log.Fatal(err)
	}
}

//goland:noinspection GoUnhandledErrorResult
func ZipDirectory(sourceDir, zipFileName string) error {
	log.Println("📦 zipping up the database")
	file, err := os.Create(zipFileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	w := zip.NewWriter(file)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	err = filepath.Walk(sourceDir, walker)
	if err != nil {
		//panic(err)
	}

	log.Println("📦 database zipped up!")
	return nil
}
