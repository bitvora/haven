package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func runBackup(ctx context.Context) {
	backupCmd := flag.NewFlagSet("backup", flag.ExitOnError)
	relay := backupCmd.String("relay", "", "Relay name (use then the file parameter ends in jsonl)")
	relayShort := backupCmd.String("r", "", "Relay name (shorthand)")
	output := backupCmd.String("output", "", "Output file (shorthand)")
	outputShort := backupCmd.String("o", "", "Output file (shorthand)")

	args := os.Args[2:]
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if it's a flag that takes a value
			// In our case, all flags (relay, r, output, o) take values.
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positionals = append(positionals, arg)
		}
	}

	err := backupCmd.Parse(append(flags, positionals...))

	if err != nil {
		log.Fatal("🚫 failed to parse backup command:", err)
		return
	}

	targetRelay := *relay
	if targetRelay == "" {
		targetRelay = *relayShort
	}

	parsedArgs := backupCmd.Args()
	fileName := "haven_backup.zip"
	if len(parsedArgs) > 0 {
		fileName = parsedArgs[0]
	}

	targetOutput := *output
	if targetOutput == "" {
		targetOutput = *outputShort
	}
	if targetOutput != "" {
		fileName = targetOutput
	}

	if strings.HasSuffix(fileName, ".jsonl") {
		if targetRelay == "" {
			log.Fatal("🚫 --relay parameter is required when exporting to .jsonl")
		}
		if err := exportToJSONL(ctx, targetRelay, fileName); err != nil {
			log.Fatal("🚫 export failed:", err)
		}
	} else {
		if err := exportToZip(ctx, fileName); err != nil {
			log.Fatal("🚫 backup failed:", err)
		}
	}
}

func runRestore(ctx context.Context) {
	restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
	relay := restoreCmd.String("relay", "", "Relay name (use then the file parameter ends in jsonl)")
	relayShort := restoreCmd.String("r", "", "Relay name (shorthand)")
	input := restoreCmd.String("input", "", "Input file (shorthand)")
	inputShort := restoreCmd.String("i", "", "Input file (shorthand)")

	args := os.Args[2:]
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positionals = append(positionals, arg)
		}
	}

	err := restoreCmd.Parse(append(flags, positionals...))

	if err != nil {
		log.Fatal("🚫 failed to parse restore command:", err)
		return
	}

	targetRelay := *relay
	if targetRelay == "" {
		targetRelay = *relayShort
	}

	parsedArgs := restoreCmd.Args()
	if len(parsedArgs) == 0 && *input == "" && *inputShort == "" {
		log.Fatal("🚫 usage: haven restore <file> or haven restore -i <file>")
	}

	fileName := ""
	if len(parsedArgs) > 0 {
		fileName = parsedArgs[0]
	}
	targetInput := *input
	if targetInput == "" {
		targetInput = *inputShort
	}
	if targetInput != "" {
		fileName = targetInput
	}

	if strings.HasSuffix(fileName, ".jsonl") {
		if targetRelay == "" {
			log.Fatal("🚫 --relay parameter is required when restoring from .jsonl")
		}
		if err := importFromJSONL(ctx, targetRelay, fileName); err != nil {
			log.Fatal("🚫 restore failed:", err)
		}
	} else {
		if err := importFromZip(ctx, fileName); err != nil {
			log.Fatal("🚫 restore failed:", err)
		}
	}
}

// startPeriodicCloudBackups periodically backs up the database to a cloud provider.
// Supported providers are S3, AWS (deprecated), and GCP (deprecated).
// The backup interval is defined by the BACKUP_INTERVAL_HOURS environment variable.
// For more details on configuration, see docs/backup.md#periodic-cloud-backups.
func startPeriodicCloudBackups(ctx context.Context) {
	if config.BackupProvider == "none" || config.BackupProvider == "" {
		log.Println("🚫 no backup provider set")
		return
	}

	ticker := time.NewTicker(time.Duration(config.BackupIntervalHours) * time.Hour)
	defer ticker.Stop()

	zipFileName := "haven_backup.zip"
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			log.Println("⏰ starting periodic backup...")
			if err := exportToZip(ctx, zipFileName); err != nil {
				log.Println("🚫 error exporting to zip:", err)
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
	log.Println("🆙 uploading to S3 Bucket...")

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
