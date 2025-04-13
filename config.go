package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type AwsConfig struct {
	AccessKeyID     string `json:"access"`
	SecretAccessKey string `json:"secret"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
}

type GcpConfig struct {
	Bucket string `json:"bucket"`
}

type S3Config struct {
	AccessKeyID string `json:"access_key_id"`
	SecretKey   string `json:"secret_key"`
	Endpoint    string `json:"endpoint"`
	BucketName  string `json:"bucket_name"`
	Region      string `json:"region"`
}

type Config struct {
	OwnerNpub                            string     `json:"owner_npub"`
	DBEngine                             string     `json:"db_engine"`
	LmdbMapSize                          int64      `json:"lmdb_map_size"`
	BlossomPath                          string     `json:"blossom_path"`
	RelayURL                             string     `json:"relay_url"`
	RelayPort                            int        `json:"relay_port"`
	RelayBindAddress                     string     `json:"relay_bind_address"`
	RelaySoftware                        string     `json:"relay_software"`
	RelayVersion                         string     `json:"relay_version"`
	PrivateRelayName                     string     `json:"private_relay_name"`
	PrivateRelayNpub                     string     `json:"private_relay_npub"`
	PrivateRelayDescription              string     `json:"private_relay_description"`
	PrivateRelayIcon                     string     `json:"private_relay_icon"`
	ChatRelayName                        string     `json:"chat_relay_name"`
	ChatRelayNpub                        string     `json:"chat_relay_npub"`
	ChatRelayDescription                 string     `json:"chat_relay_description"`
	ChatRelayIcon                        string     `json:"chat_relay_icon"`
	ChatRelayWotDepth                    int        `json:"chat_relay_wot_depth"`
	ChatRelayWotRefreshIntervalHours     int        `json:"chat_relay_wot_refresh_interval_hours"`
	ChatRelayMinimumFollowers            int        `json:"chat_relay_minimum_followers"`
	OutboxRelayName                      string     `json:"outbox_relay_name"`
	OutboxRelayNpub                      string     `json:"outbox_relay_npub"`
	OutboxRelayDescription               string     `json:"outbox_relay_description"`
	OutboxRelayIcon                      string     `json:"outbox_relay_icon"`
	InboxRelayName                       string     `json:"inbox_relay_name"`
	InboxRelayNpub                       string     `json:"inbox_relay_npub"`
	InboxRelayDescription                string     `json:"inbox_relay_description"`
	InboxRelayIcon                       string     `json:"inbox_relay_icon"`
	InboxPullIntervalSeconds             int        `json:"inbox_pull_interval_seconds"`
	ImportStartDate                      string     `json:"import_start_date"`
	ImportOwnerNotesFetchTimeoutSeconds  int        `json:"import_owned_notes_fetch_timeout_seconds"`
	ImportTaggedNotesFetchTimeoutSeconds int        `json:"import_tagged_fetch_timeout_seconds"`
	ImportQueryIntervalSeconds           int        `json:"import_query_interval_seconds"`
	ImportSeedRelays                     []string   `json:"import_seed_relays"`
	BackupProvider                       string     `json:"backup_provider"`
	BackupIntervalHours                  int        `json:"backup_interval_hours"`
	WotFetchTimeoutSeconds               int        `json:"wot_fetch_timeout_seconds"`
	LogLevel                             string     `json:"log_level"`
	BlastrRelays                         []string   `json:"blastr_relays"`
	AwsConfig                            *AwsConfig `json:"aws_config"`
	S3Config                             *S3Config  `json:"s3_config"`
	GcpConfig                            *GcpConfig `json:"gcp_config"`
}

func loadConfig() Config {
	_ = godotenv.Load(".env")

	return Config{
		OwnerNpub:                            getEnv("OWNER_NPUB"),
		DBEngine:                             getEnvString("DB_ENGINE", "lmdb"),
		LmdbMapSize:                          getEnvInt64("LMDB_MAPSIZE", 0),
		BlossomPath:                          getEnvString("BLOSSOM_PATH", "blossom"),
		RelayURL:                             getEnv("RELAY_URL"),
		RelayPort:                            getEnvInt("RELAY_PORT", 3355),
		RelayBindAddress:                     getEnvString("RELAY_BIND_ADDRESS", "0.0.0.0"),
		RelaySoftware:                        "https://github.com/bitvora/haven",
		RelayVersion:                         "v1.0.5",
		PrivateRelayName:                     getEnv("PRIVATE_RELAY_NAME"),
		PrivateRelayNpub:                     getEnv("PRIVATE_RELAY_NPUB"),
		PrivateRelayDescription:              getEnv("PRIVATE_RELAY_DESCRIPTION"),
		PrivateRelayIcon:                     getEnv("PRIVATE_RELAY_ICON"),
		ChatRelayName:                        getEnv("CHAT_RELAY_NAME"),
		ChatRelayNpub:                        getEnv("CHAT_RELAY_NPUB"),
		ChatRelayDescription:                 getEnv("CHAT_RELAY_DESCRIPTION"),
		ChatRelayIcon:                        getEnv("CHAT_RELAY_ICON"),
		ChatRelayWotDepth:                    getEnvInt("CHAT_RELAY_WOT_DEPTH", 0),
		ChatRelayWotRefreshIntervalHours:     getEnvInt("CHAT_RELAY_WOT_REFRESH_INTERVAL_HOURS", 0),
		ChatRelayMinimumFollowers:            getEnvInt("CHAT_RELAY_MINIMUM_FOLLOWERS", 0),
		OutboxRelayName:                      getEnv("OUTBOX_RELAY_NAME"),
		OutboxRelayNpub:                      getEnv("OUTBOX_RELAY_NPUB"),
		OutboxRelayDescription:               getEnv("OUTBOX_RELAY_DESCRIPTION"),
		OutboxRelayIcon:                      getEnv("OUTBOX_RELAY_ICON"),
		InboxRelayName:                       getEnv("INBOX_RELAY_NAME"),
		InboxRelayNpub:                       getEnv("INBOX_RELAY_NPUB"),
		InboxRelayDescription:                getEnv("INBOX_RELAY_DESCRIPTION"),
		InboxRelayIcon:                       getEnv("INBOX_RELAY_ICON"),
		InboxPullIntervalSeconds:             getEnvInt("INBOX_PULL_INTERVAL_SECONDS", 3600),
		ImportStartDate:                      getEnv("IMPORT_START_DATE"),
		ImportOwnerNotesFetchTimeoutSeconds:  getEnvInt("IMPORT_OWNER_NOTES_FETCH_TIMEOUT_SECONDS", 60),
		ImportTaggedNotesFetchTimeoutSeconds: getEnvInt("IMPORT_TAGGED_NOTES_FETCH_TIMEOUT_SECONDS", 120),
		ImportQueryIntervalSeconds:           getEnvInt("IMPORT_QUERY_INTERVAL_SECONDS", 360000),
		ImportSeedRelays:                     getRelayListFromFile(getEnv("IMPORT_SEED_RELAYS_FILE")),
		BackupProvider:                       getEnv("BACKUP_PROVIDER"),
		BackupIntervalHours:                  getEnvInt("BACKUP_INTERVAL_HOURS", 24),
		WotFetchTimeoutSeconds:               getEnvInt("WOT_FETCH_TIMEOUT_SECONDS", 60),
		LogLevel:                             getEnvString("HAVEN_LOG_LEVEL", "INFO"),
		BlastrRelays:                         getRelayListFromFile(getEnv("BLASTR_RELAYS_FILE")),
		AwsConfig:                            getAwsConfig(),
		S3Config:                             getS3Config(),
		GcpConfig:                            getGcpConfig(),
	}
}

func getAwsConfig() *AwsConfig {
	backupProvider := getEnv("BACKUP_PROVIDER")

	if backupProvider == "aws" {
		return &AwsConfig{
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY"),
			Region:          getEnv("AWS_REGION"),
			Bucket:          getEnv("AWS_BUCKET"),
		}
	}

	return nil
}

func getS3Config() *S3Config {
	backupProvider := getEnv("BACKUP_PROVIDER")

	if backupProvider == "s3" {
		return &S3Config{
			AccessKeyID: getEnv("S3_ACCESS_KEY_ID"),
			SecretKey:   getEnv("S3_SECRET_KEY"),
			Endpoint:    getEnv("S3_ENDPOINT"),
			BucketName:  getEnv("S3_BUCKET_NAME"),
			Region:      getEnv("S3_REGION"),
		}
	}

	return nil
}

func getGcpConfig() *GcpConfig {
	backupProvider := getEnv("BACKUP_PROVIDER")

	if backupProvider == "gcp" {
		return &GcpConfig{
			Bucket: getEnv("GCP_BUCKET_NAME"),
		}
	}

	return nil
}

func getRelayListFromFile(filePath string) []string {
	file, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file: %s", err)
	}

	var relayList []string
	if err := json.Unmarshal(file, &relayList); err != nil {
		log.Fatalf("Failed to parse JSON: %s", err)
	}

	for i, relay := range relayList {
		relay = strings.TrimSpace(relay)
		if !strings.HasPrefix(relay, "wss://") && !strings.HasPrefix(relay, "ws://") {
			relay = "wss://" + relay
		}
		relayList[i] = relay
	}
	return relayList
}

func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatalf("Environment variable %s not set", key)
	}
	return value
}

func getEnvString(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}
		return intValue
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			panic(err)
		}
		return intValue
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			panic(err)
		}
		return boolValue
	}
	return defaultValue
}

var art = `
██╗  ██╗ █████╗ ██╗   ██╗███████╗███╗   ██╗
██║  ██║██╔══██╗██║   ██║██╔════╝████╗  ██║
███████║███████║██║   ██║█████╗  ██╔██╗ ██║
██╔══██║██╔══██║╚██╗ ██╔╝██╔══╝  ██║╚██╗██║
██║  ██║██║  ██║ ╚████╔╝ ███████╗██║ ╚████║
╚═╝  ╚═╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═══╝
HIGH AVAILABILITY VAULT FOR EVENTS ON NOSTR
	`
