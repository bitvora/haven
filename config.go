package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"encoding/json"
	"io/ioutil"

	"github.com/joho/godotenv"
)

type Config struct {
	OwnerNpub                        string   `json:"owner_npub"`
	RelayURL                         string   `json:"relay_url"`
	RelaySoftware                    string   `json:"relay_software"`
	RelayVersion                     string   `json:"relay_version"`
	PrivateRelayName                 string   `json:"private_relay_name"`
	PrivateRelayNpub                 string   `json:"private_relay_npub"`
	PrivateRelayDescription          string   `json:"private_relay_description"`
	PrivateRelayIcon                 string   `json:"private_relay_icon"`
	ChatRelayName                    string   `json:"chat_relay_name"`
	ChatRelayNpub                    string   `json:"chat_relay_npub"`
	ChatRelayDescription             string   `json:"chat_relay_description"`
	ChatRelayIcon                    string   `json:"chat_relay_icon"`
	ChatRelayWotDepth                int      `json:"chat_relay_wot_depth"`
	ChatRelayWotRefreshIntervalHours int      `json:"chat_relay_wot_refresh_interval_hours"`
	ChatRelayMinimumFollowers        int      `json:"chat_relay_minimum_followers"`
	OutboxRelayName                  string   `json:"outbox_relay_name"`
	OutboxRelayNpub                  string   `json:"outbox_relay_npub"`
	OutboxRelayDescription           string   `json:"outbox_relay_description"`
	OutboxRelayIcon                  string   `json:"outbox_relay_icon"`
	InboxRelayName                   string   `json:"inbox_relay_name"`
	InboxRelayNpub                   string   `json:"inbox_relay_npub"`
	InboxRelayDescription            string   `json:"inbox_relay_description"`
	InboxRelayIcon                   string   `json:"inbox_relay_icon"`
	InboxPullIntervalSeconds         int      `json:"inbox_pull_interval_seconds"`
	ImportStartDate                  string   `json:"import_start_date"`
	ImportQueryIntervalSeconds       int      `json:"import_query_interval_seconds"`
	ImportSeedRelays                 []string `json:"import_seed_relays"`
	BackupProvider                   string   `json:"backup_provider"`
	BackupIntervalHours              int      `json:"backup_interval_hours"`
	BlastrRelays                     []string `json:"blastr_relays"`
}

type AwsConfig struct {
	AccessKeyID     string `json:"access"`
	SecretAccessKey string `json:"secret"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
}

func getRelayListFromEnvOrFile(envKey, fileKey string) []string {
	envValue := getEnv(envKey)
	filePath := getEnv(fileKey)

	if filePath != "" {
		return getRelayListFromFile(filePath)
	}

	if envValue != "" {
		return getRelayList(envValue)
	}

	return []string{}
}

func loadConfig() Config {
	godotenv.Load(".env")

	return Config{
		OwnerNpub:                        getEnv("OWNER_NPUB"),
		RelayURL:                         getEnv("RELAY_URL"),
		RelaySoftware:                    "https://github.com/bitvora/haven",
		RelayVersion:                     "v0.1.0",
		PrivateRelayName:                 getEnv("PRIVATE_RELAY_NAME"),
		PrivateRelayNpub:                 getEnv("PRIVATE_RELAY_NPUB"),
		PrivateRelayDescription:          getEnv("PRIVATE_RELAY_DESCRIPTION"),
		PrivateRelayIcon:                 getEnv("PRIVATE_RELAY_ICON"),
		ChatRelayName:                    getEnv("CHAT_RELAY_NAME"),
		ChatRelayNpub:                    getEnv("CHAT_RELAY_NPUB"),
		ChatRelayDescription:             getEnv("CHAT_RELAY_DESCRIPTION"),
		ChatRelayIcon:                    getEnv("CHAT_RELAY_ICON"),
		ChatRelayWotDepth:                getEnvInt("CHAT_RELAY_WOT_DEPTH", 0),
		ChatRelayWotRefreshIntervalHours: getEnvInt("CHAT_RELAY_WOT_REFRESH_INTERVAL_HOURS", 0),
		ChatRelayMinimumFollowers:        getEnvInt("CHAT_RELAY_MINIMUM_FOLLOWERS", 0),
		OutboxRelayName:                  getEnv("OUTBOX_RELAY_NAME"),
		OutboxRelayNpub:                  getEnv("OUTBOX_RELAY_NPUB"),
		OutboxRelayDescription:           getEnv("OUTBOX_RELAY_DESCRIPTION"),
		OutboxRelayIcon:                  getEnv("OUTBOX_RELAY_ICON"),
		InboxRelayName:                   getEnv("INBOX_RELAY_NAME"),
		InboxRelayNpub:                   getEnv("INBOX_RELAY_NPUB"),
		InboxRelayDescription:            getEnv("INBOX_RELAY_DESCRIPTION"),
		InboxRelayIcon:                   getEnv("INBOX_RELAY_ICON"),
		InboxPullIntervalSeconds:         getEnvInt("INBOX_PULL_INTERVAL_SECONDS", 3600),
		ImportStartDate:                  getEnv("IMPORT_START_DATE"),
		ImportQueryIntervalSeconds:       getEnvInt("IMPORT_QUERY_INTERVAL_SECONDS", 360000),
		ImportSeedRelays:                 getRelayListFromEnvOrFile("IMPORT_SEED_RELAYS", "IMPORT_SEED_RELAYS_FILE"),
		BackupProvider:                   getEnv("BACKUP_PROVIDER"),
		BackupIntervalHours:              getEnvInt("BACKUP_INTERVAL_HOURS", 24),
		BlastrRelays:                     getRelayListFromEnvOrFile("BLASTR_RELAYS", "BLASTR_RELAYS_FILE"),
	}
}

func getRelayList(commaList string) []string {
	relayList := strings.Split(commaList, ",")
	for i, relay := range relayList {
		relayList[i] = "wss://" + strings.TrimSpace(relay)
	}
	return relayList
}

func getRelayListFromFile(filePath string) []string {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file: %s", err)
	}

	var relayList []string
	if err := json.Unmarshal(file, &relayList); err != nil {
		log.Fatalf("Failed to parse JSON: %s", err)
	}

	for i, relay := range relayList {
		relay = strings.TrimSpace(relay)
		if !strings.HasPrefix(relay, "wss://") {
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

var art = `
██╗  ██╗ █████╗ ██╗   ██╗███████╗███╗   ██╗
██║  ██║██╔══██╗██║   ██║██╔════╝████╗  ██║
███████║███████║██║   ██║█████╗  ██╔██╗ ██║
██╔══██║██╔══██║╚██╗ ██╔╝██╔══╝  ██║╚██╗██║
██║  ██║██║  ██║ ╚████╔╝ ███████╗██║ ╚████║
╚═╝  ╚═╝╚═╝  ╚═╝  ╚═══╝  ╚══════╝╚═╝  ╚═══╝
HIGH AVAILABILITY VAULT FOR EVENTS ON NOSTR
	`
