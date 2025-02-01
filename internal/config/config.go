package config

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/bitvora/haven/internal/utils"
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

type Config struct {
	OwnerNpub                        string     `json:"owner_npub"`
	DBEngine                         string     `json:"db_engine"`
	LmdbMapSize                      int64      `json:"lmdb_map_size"`
	RelayURL                         string     `json:"relay_url"`
	RelayPort                        int        `json:"relay_port"`
	RelayBindAddress                 string     `json:"relay_bind_address"`
	RelaySoftware                    string     `json:"relay_software"`
	RelayVersion                     string     `json:"relay_version"`
	PrivateRelayName                 string     `json:"private_relay_name"`
	PrivateRelayNpub                 string     `json:"private_relay_npub"`
	PrivateRelayDescription          string     `json:"private_relay_description"`
	PrivateRelayIcon                 string     `json:"private_relay_icon"`
	ChatRelayName                    string     `json:"chat_relay_name"`
	ChatRelayNpub                    string     `json:"chat_relay_npub"`
	ChatRelayDescription             string     `json:"chat_relay_description"`
	ChatRelayIcon                    string     `json:"chat_relay_icon"`
	ChatRelayWotDepth                int        `json:"chat_relay_wot_depth"`
	ChatRelayWotRefreshIntervalHours int        `json:"chat_relay_wot_refresh_interval_hours"`
	ChatRelayMinimumFollowers        int        `json:"chat_relay_minimum_followers"`
	OutboxRelayName                  string     `json:"outbox_relay_name"`
	OutboxRelayNpub                  string     `json:"outbox_relay_npub"`
	OutboxRelayDescription           string     `json:"outbox_relay_description"`
	OutboxRelayIcon                  string     `json:"outbox_relay_icon"`
	InboxRelayName                   string     `json:"inbox_relay_name"`
	InboxRelayNpub                   string     `json:"inbox_relay_npub"`
	InboxRelayDescription            string     `json:"inbox_relay_description"`
	InboxRelayIcon                   string     `json:"inbox_relay_icon"`
	InboxPullIntervalSeconds         int        `json:"inbox_pull_interval_seconds"`
	ImportStartDate                  string     `json:"import_start_date"`
	ImportQueryIntervalSeconds       int        `json:"import_query_interval_seconds"`
	ImportSeedRelays                 []string   `json:"import_seed_relays"`
	BackupProvider                   string     `json:"backup_provider"`
	BackupIntervalHours              int        `json:"backup_interval_hours"`
	BlastrRelays                     []string   `json:"blastr_relays"`
	BlossomPath                      string     `json:"blossom_path"`
	AwsConfig                        *AwsConfig `json:"aws_config"`
	GcpConfig                        *GcpConfig `json:"gcp_config"`
}

func LoadConfig() Config {
	_ = godotenv.Load(".env")

	return Config{
		OwnerNpub:                        utils.GetEnv("OWNER_NPUB"),
		DBEngine:                         utils.GetEnvString("DB_ENGINE", "lmdb"),
		LmdbMapSize:                      utils.GetEnvInt64("LMDB_MAPSIZE", 0),
		BlossomPath:                      utils.GetEnvString("BLOSSOM_PATH", "blossom"),
		RelayURL:                         utils.GetEnv("RELAY_URL"),
		RelayPort:                        utils.GetEnvInt("RELAY_PORT", 3355),
		RelayBindAddress:                 utils.GetEnvString("RELAY_BIND_ADDRESS", "0.0.0.0"),
		RelaySoftware:                    "https://github.com/bitvora/haven",
		RelayVersion:                     "v1.0.4",
		PrivateRelayName:                 utils.GetEnv("PRIVATE_RELAY_NAME"),
		PrivateRelayNpub:                 utils.GetEnv("PRIVATE_RELAY_NPUB"),
		PrivateRelayDescription:          utils.GetEnv("PRIVATE_RELAY_DESCRIPTION"),
		PrivateRelayIcon:                 utils.GetEnv("PRIVATE_RELAY_ICON"),
		ChatRelayName:                    utils.GetEnv("CHAT_RELAY_NAME"),
		ChatRelayNpub:                    utils.GetEnv("CHAT_RELAY_NPUB"),
		ChatRelayDescription:             utils.GetEnv("CHAT_RELAY_DESCRIPTION"),
		ChatRelayIcon:                    utils.GetEnv("CHAT_RELAY_ICON"),
		ChatRelayWotDepth:                utils.GetEnvInt("CHAT_RELAY_WOT_DEPTH", 0),
		ChatRelayWotRefreshIntervalHours: utils.GetEnvInt("CHAT_RELAY_WOT_REFRESH_INTERVAL_HOURS", 0),
		ChatRelayMinimumFollowers:        utils.GetEnvInt("CHAT_RELAY_MINIMUM_FOLLOWERS", 0),
		OutboxRelayName:                  utils.GetEnv("OUTBOX_RELAY_NAME"),
		OutboxRelayNpub:                  utils.GetEnv("OUTBOX_RELAY_NPUB"),
		OutboxRelayDescription:           utils.GetEnv("OUTBOX_RELAY_DESCRIPTION"),
		OutboxRelayIcon:                  utils.GetEnv("OUTBOX_RELAY_ICON"),
		InboxRelayName:                   utils.GetEnv("INBOX_RELAY_NAME"),
		InboxRelayNpub:                   utils.GetEnv("INBOX_RELAY_NPUB"),
		InboxRelayDescription:            utils.GetEnv("INBOX_RELAY_DESCRIPTION"),
		InboxRelayIcon:                   utils.GetEnv("INBOX_RELAY_ICON"),
		InboxPullIntervalSeconds:         utils.GetEnvInt("INBOX_PULL_INTERVAL_SECONDS", 3600),
		ImportStartDate:                  utils.GetEnv("IMPORT_START_DATE"),
		ImportQueryIntervalSeconds:       utils.GetEnvInt("IMPORT_QUERY_INTERVAL_SECONDS", 360000),
		ImportSeedRelays:                 getRelayListFromFile(utils.GetEnv("IMPORT_SEED_RELAYS_FILE")),
		BackupProvider:                   utils.GetEnv("BACKUP_PROVIDER"),
		BackupIntervalHours:              utils.GetEnvInt("BACKUP_INTERVAL_HOURS", 24),
		BlastrRelays:                     getRelayListFromFile(utils.GetEnv("BLASTR_RELAYS_FILE")),
		AwsConfig:                        getAwsConfig(),
		GcpConfig:                        getGcpConfig(),
	}
}

func getGcpConfig() *GcpConfig {
	backupProvider := utils.GetEnv("BACKUP_PROVIDER")

	if backupProvider == "" || backupProvider == "none" {
		return nil
	}

	if backupProvider == "gcp" {
		return &GcpConfig{
			Bucket: utils.GetEnv("GCP_BUCKET_NAME"),
		}
	}

	return nil
}

func getAwsConfig() *AwsConfig {
	backupProvider := utils.GetEnv("BACKUP_PROVIDER")

	if backupProvider == "" || backupProvider == "none" {
		return nil
	}

	if backupProvider == "aws" {
		return &AwsConfig{
			AccessKeyID:     utils.GetEnv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: utils.GetEnv("AWS_SECRET_ACCESS_KEY"),
			Region:          utils.GetEnv("AWS_REGION"),
			Bucket:          utils.GetEnv("AWS_BUCKET_NAME"),
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

func Art() string {
	return `
██╗  ██╗ █████╗ ██╗   ██╗███████╗███╗   ██╗
██║  ██║██╔══██╗██║   ██║██╔════╝████╗  ██║
███████║███████║██║   ██║█████╗  ██╔██╗ ██║
██╔══██║██╔══██║╚██╗ ██╔╝██╔══╝  ██║╚██╗██║
██║  ██║██║  ██║ ╚████╔╝ ███████╗██║ ╚████║
╚═╝  ╚═╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═══╝
HIGH AVAILABILITY VAULT FOR EVENTS ON NOSTR
	`
}
