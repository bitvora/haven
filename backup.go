package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bitvora/haven/internal/cloud"
)

func runBackup(ctx context.Context) {
	backupCmd := flag.NewFlagSet("backup", flag.ExitOnError)
	relay := backupCmd.String("relay", "", "Relay name (use then the file parameter ends in jsonl)")
	relayShort := backupCmd.String("r", "", "Relay name (shorthand)")
	output := backupCmd.String("output", "", "Output file (shorthand)")
	outputShort := backupCmd.String("o", "", "Output file (shorthand)")
	toCloud := backupCmd.Bool("to-cloud", false, "Upload backup to cloud storage")

	args := os.Args[2:]
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			// Check if it's a flag that takes a value
			// In our case, all flags (relay, r, output, o) take values, but to-cloud does not.
			if arg == "--to-cloud" {
				continue
			}
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

	if *toCloud {
		cloudProvider, err := getCloudUploader()
		if err != nil {
			log.Fatal("🚫 ", err)
		}
		if err := uploadBackupToCloud(ctx, cloudProvider, fileName); err != nil {
			log.Fatal("🚫 ", err)
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

func getCloudUploader() (cloud.Uploader, error) {
	if config.BackupProvider == "none" || config.BackupProvider == "" {
		return nil, fmt.Errorf("no backup provider set")
	} else if config.BackupProvider != "s3" {
		return nil, fmt.Errorf("backup provider %q not supported", config.BackupProvider)
	}

	cloudProvider, err := cloud.NewGenericS3Provider(
		config.S3Config.Endpoint,
		config.S3Config.AccessKeyID,
		config.S3Config.SecretKey,
		config.S3Config.Region,
	)
	if err != nil {
		return nil, err
	}
	return cloudProvider, nil
}

// startPeriodicCloudBackups periodically backs up the database to a cloud provider.
// Supported providers are S3, AWS (deprecated), and GCP (deprecated).
// The backup interval is defined by the BACKUP_INTERVAL_HOURS environment variable.
// For more details on configuration, see docs/backup.md#periodic-cloud-backups.
func startPeriodicCloudBackups(ctx context.Context) {
	cloudProvider, err := getCloudUploader()
	if err != nil {
		log.Printf("⚠️ Cloud backup disabled: %v", err)
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
			if err := uploadBackupToCloud(ctx, cloudProvider, zipFileName); err != nil {
				log.Println("🚫 error uploading to cloud:", err)
				continue
			}
			// delete the file
			err = os.Remove(zipFileName)
			if err != nil {
				log.Println("🚫 error deleting local backup file:", err)
			}
		}
	}
}

func uploadBackupToCloud(ctx context.Context, uploader cloud.Uploader, fileName string) error {
	log.Println("🆙 uploading backup to S3 Bucket...")

	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			log.Println("🚫 error closing db zip file:", err)
		}
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", fileName, err)
	}

	err = uploader.Upload(ctx, config.S3Config.BucketName, fileName, file, fileInfo.Size(), getContentType(fileName))
	if err != nil {
		return fmt.Errorf("failed to upload %s to %s: %w", fileName, config.S3Config.BucketName, err)
	}

	log.Printf("✅ Successfully uploaded %q to %q\n", fileName, config.S3Config.BucketName)

	return nil
}

func getContentType(objectName string) string {
	if strings.HasSuffix(objectName, ".zip") {
		return "application/zip"
	} else if strings.HasSuffix(objectName, ".jsonl") {
		return "application/jsonl"
	}
	return ""
}
