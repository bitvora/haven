package main

import (
	"encoding/json"
	"log"
)

var (
	privateRelayLimits PrivateRelayLimits
	chatRelayLimits    ChatRelayLimits
	inboxRelayLimits   InboxRelayLimits
	outboxRelayLimits  OutboxRelayLimits
)

type PrivateRelayLimits struct {
	EventIPLimiterTokensPerInterval        int
	EventIPLimiterInterval                 int
	EventIPLimiterMaxTokens                int
	AllowEmptyFilters                      bool
	AllowComplexFilters                    bool
	ConnectionRateLimiterTokensPerInterval int
	ConnectionRateLimiterInterval          int
	ConnectionRateLimiterMaxTokens         int
}

type ChatRelayLimits struct {
	EventIPLimiterTokensPerInterval        int
	EventIPLimiterInterval                 int
	EventIPLimiterMaxTokens                int
	AllowEmptyFilters                      bool
	AllowComplexFilters                    bool
	ConnectionRateLimiterTokensPerInterval int
	ConnectionRateLimiterInterval          int
	ConnectionRateLimiterMaxTokens         int
}

type InboxRelayLimits struct {
	EventIPLimiterTokensPerInterval        int
	EventIPLimiterInterval                 int
	EventIPLimiterMaxTokens                int
	AllowEmptyFilters                      bool
	AllowComplexFilters                    bool
	ConnectionRateLimiterTokensPerInterval int
	ConnectionRateLimiterInterval          int
	ConnectionRateLimiterMaxTokens         int
}

type OutboxRelayLimits struct {
	EventIPLimiterTokensPerInterval        int
	EventIPLimiterInterval                 int
	EventIPLimiterMaxTokens                int
	AllowEmptyFilters                      bool
	AllowComplexFilters                    bool
	ConnectionRateLimiterTokensPerInterval int
	ConnectionRateLimiterInterval          int
	ConnectionRateLimiterMaxTokens         int
}

func initRelayLimits() {
	privateRelayLimits = PrivateRelayLimits{
		EventIPLimiterTokensPerInterval:        getEnvInt("PRIVATE_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL", 50),
		EventIPLimiterInterval:                 getEnvInt("PRIVATE_RELAY_EVENT_IP_LIMITER_INTERVAL", 1),
		EventIPLimiterMaxTokens:                getEnvInt("PRIVATE_RELAY_EVENT_IP_LIMITER_MAX_TOKENS", 100),
		AllowEmptyFilters:                      getEnvBool("PRIVATE_RELAY_ALLOW_EMPTY_FILTERS", true),
		AllowComplexFilters:                    getEnvBool("PRIVATE_RELAY_ALLOW_COMPLEX_FILTERS", true),
		ConnectionRateLimiterTokensPerInterval: getEnvInt("PRIVATE_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL", 3),
		ConnectionRateLimiterInterval:          getEnvInt("PRIVATE_RELAY_CONNECTION_RATE_LIMITER_INTERVAL", 5),
		ConnectionRateLimiterMaxTokens:         getEnvInt("PRIVATE_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS", 9),
	}

	chatRelayLimits = ChatRelayLimits{
		EventIPLimiterTokensPerInterval:        getEnvInt("CHAT_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL", 50),
		EventIPLimiterInterval:                 getEnvInt("CHAT_RELAY_EVENT_IP_LIMITER_INTERVAL", 1),
		EventIPLimiterMaxTokens:                getEnvInt("CHAT_RELAY_EVENT_IP_LIMITER_MAX_TOKENS", 100),
		AllowEmptyFilters:                      getEnvBool("CHAT_RELAY_ALLOW_EMPTY_FILTERS", false),
		AllowComplexFilters:                    getEnvBool("CHAT_RELAY_ALLOW_COMPLEX_FILTERS", false),
		ConnectionRateLimiterTokensPerInterval: getEnvInt("CHAT_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL", 3),
		ConnectionRateLimiterInterval:          getEnvInt("CHAT_RELAY_CONNECTION_RATE_LIMITER_INTERVAL", 3),
		ConnectionRateLimiterMaxTokens:         getEnvInt("CHAT_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS", 9),
	}

	inboxRelayLimits = InboxRelayLimits{
		EventIPLimiterTokensPerInterval:        getEnvInt("INBOX_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL", 10),
		EventIPLimiterInterval:                 getEnvInt("INBOX_RELAY_EVENT_IP_LIMITER_INTERVAL", 1),
		EventIPLimiterMaxTokens:                getEnvInt("INBOX_RELAY_EVENT_IP_LIMITER_MAX_TOKENS", 20),
		AllowEmptyFilters:                      getEnvBool("INBOX_RELAY_ALLOW_EMPTY_FILTERS", false),
		AllowComplexFilters:                    getEnvBool("INBOX_RELAY_ALLOW_COMPLEX_FILTERS", false),
		ConnectionRateLimiterTokensPerInterval: getEnvInt("INBOX_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL", 3),
		ConnectionRateLimiterInterval:          getEnvInt("INBOX_RELAY_CONNECTION_RATE_LIMITER_INTERVAL", 1),
		ConnectionRateLimiterMaxTokens:         getEnvInt("INBOX_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS", 9),
	}

	outboxRelayLimits = OutboxRelayLimits{
		EventIPLimiterTokensPerInterval:        getEnvInt("OUTBOX_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL", 10),
		EventIPLimiterInterval:                 getEnvInt("OUTBOX_RELAY_EVENT_IP_LIMITER_INTERVAL", 60),
		EventIPLimiterMaxTokens:                getEnvInt("OUTBOX_RELAY_EVENT_IP_LIMITER_MAX_TOKENS", 100),
		AllowEmptyFilters:                      getEnvBool("OUTBOX_RELAY_ALLOW_EMPTY_FILTERS", false),
		AllowComplexFilters:                    getEnvBool("OUTBOX_RELAY_ALLOW_COMPLEX_FILTERS", false),
		ConnectionRateLimiterTokensPerInterval: getEnvInt("OUTBOX_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL", 3),
		ConnectionRateLimiterInterval:          getEnvInt("OUTBOX_RELAY_CONNECTION_RATE_LIMITER_INTERVAL", 1),
		ConnectionRateLimiterMaxTokens:         getEnvInt("OUTBOX_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS", 9),
	}

	prettyPrintLimits("Private relay limits", privateRelayLimits)
	prettyPrintLimits("Chat relay limits", chatRelayLimits)
	prettyPrintLimits("Inbox relay limits", inboxRelayLimits)
	prettyPrintLimits("Outbox relay limits", outboxRelayLimits)
}

func prettyPrintLimits(label string, value any) {
	b, _ := json.MarshalIndent(value, "", "  ")
	log.Printf("🚧 %s:\n%s\n", label, string(b))
}
