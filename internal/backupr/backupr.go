package backupr

import (
	"archive/zip"
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bitvora/haven/internal/config"
)

type Backupr struct {
	cfg config.Config
}

func NewBackupr(cfg config.Config) *Backupr {
	return &Backupr{
		cfg: cfg,
	}
}

func (b *Backupr) BackupDatabase() {
	if b.cfg.BackupProvider == "none" || b.cfg.BackupProvider == "" {
		log.Println("🚫 no backup provider set")
		return
	}

	ticker := time.NewTicker(time.Duration(b.cfg.BackupIntervalHours) * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		b.ZipDirectory()
		switch b.cfg.BackupProvider {
		case "aws":
			b.S3Upload()
		case "gcp":
			b.GCPBucketUpload()
		default:
			log.Println("🚫 we only support AWS at this time")
		}
	}
}

func (b *Backupr) GCPBucketUpload() {
	bucket := b.cfg.GcpConfig.Bucket

	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// open the zip db file.
	f, err := os.Open("db.zip")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	obj := client.Bucket(bucket).Object("db.zip")

	// Upload an object with storage.Writer.
	wc := obj.NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		log.Fatal(err)
	}

	if err := wc.Close(); err != nil {
		log.Fatal(err)
	}

	log.Printf("✅ Successfully uploaded %q to %q\n", "db.zip", bucket)

	// delete the file.
	err = os.Remove("db.zip")
	if err != nil {
		log.Fatal(err)
	}
}

func (b *Backupr) S3Upload() {
	bucket := b.cfg.AwsConfig.Bucket
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)

	// Upload the file to S3
	file, err := os.Open("db.zip")
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("db.zip"),
		Body:   file,
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("✅ Successfully uploaded %q to %q\n", "db.zip", bucket)

	// delete the file
	err = os.Remove("db.zip")
	if err != nil {
		log.Fatal(err)
	}
}

func (b *Backupr) ZipDirectory() error {
	log.Println("📦 zipping up the database")
	file, err := os.Create("db.zip")
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
	err = filepath.Walk("db", walker)
	if err != nil {
		//panic(err)
	}

	log.Println("📦 database zipped up!")
	return nil
}
