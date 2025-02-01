package limits

import (
	"encoding/json"
	"log"

	"github.com/bitvora/haven/internal/utils"
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

type Limits struct {
	PrivateRelayLimits PrivateRelayLimits
	ChatRelayLimits    ChatRelayLimits
	InboxRelayLimits   InboxRelayLimits
	OutboxRelayLimits  OutboxRelayLimits
}

func NewLimits() Limits {
	return Limits{
		PrivateRelayLimits: PrivateRelayLimits{
			EventIPLimiterTokensPerInterval:        utils.GetEnvInt("PRIVATE_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL", 50),
			EventIPLimiterInterval:                 utils.GetEnvInt("PRIVATE_RELAY_EVENT_IP_LIMITER_INTERVAL", 1),
			EventIPLimiterMaxTokens:                utils.GetEnvInt("PRIVATE_RELAY_EVENT_IP_LIMITER_MAX_TOKENS", 100),
			AllowEmptyFilters:                      utils.GetEnvBool("PRIVATE_RELAY_ALLOW_EMPTY_FILTERS", true),
			AllowComplexFilters:                    utils.GetEnvBool("PRIVATE_RELAY_ALLOW_COMPLEX_FILTERS", true),
			ConnectionRateLimiterTokensPerInterval: utils.GetEnvInt("PRIVATE_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL", 3),
			ConnectionRateLimiterInterval:          utils.GetEnvInt("PRIVATE_RELAY_CONNECTION_RATE_LIMITER_INTERVAL", 5),
			ConnectionRateLimiterMaxTokens:         utils.GetEnvInt("PRIVATE_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS", 9),
		},
		ChatRelayLimits: ChatRelayLimits{
			EventIPLimiterTokensPerInterval:        utils.GetEnvInt("CHAT_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL", 50),
			EventIPLimiterInterval:                 utils.GetEnvInt("CHAT_RELAY_EVENT_IP_LIMITER_INTERVAL", 1),
			EventIPLimiterMaxTokens:                utils.GetEnvInt("CHAT_RELAY_EVENT_IP_LIMITER_MAX_TOKENS", 100),
			AllowEmptyFilters:                      utils.GetEnvBool("CHAT_RELAY_ALLOW_EMPTY_FILTERS", false),
			AllowComplexFilters:                    utils.GetEnvBool("CHAT_RELAY_ALLOW_COMPLEX_FILTERS", false),
			ConnectionRateLimiterTokensPerInterval: utils.GetEnvInt("CHAT_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL", 3),
			ConnectionRateLimiterInterval:          utils.GetEnvInt("CHAT_RELAY_CONNECTION_RATE_LIMITER_INTERVAL", 3),
			ConnectionRateLimiterMaxTokens:         utils.GetEnvInt("CHAT_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS", 9),
		},
		InboxRelayLimits: InboxRelayLimits{
			EventIPLimiterTokensPerInterval:        utils.GetEnvInt("INBOX_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL", 10),
			EventIPLimiterInterval:                 utils.GetEnvInt("INBOX_RELAY_EVENT_IP_LIMITER_INTERVAL", 1),
			EventIPLimiterMaxTokens:                utils.GetEnvInt("INBOX_RELAY_EVENT_IP_LIMITER_MAX_TOKENS", 20),
			AllowEmptyFilters:                      utils.GetEnvBool("INBOX_RELAY_ALLOW_EMPTY_FILTERS", false),
			AllowComplexFilters:                    utils.GetEnvBool("INBOX_RELAY_ALLOW_COMPLEX_FILTERS", false),
			ConnectionRateLimiterTokensPerInterval: utils.GetEnvInt("INBOX_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL", 3),
			ConnectionRateLimiterInterval:          utils.GetEnvInt("INBOX_RELAY_CONNECTION_RATE_LIMITER_INTERVAL", 1),
			ConnectionRateLimiterMaxTokens:         utils.GetEnvInt("INBOX_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS", 9),
		},
		OutboxRelayLimits: OutboxRelayLimits{
			EventIPLimiterTokensPerInterval:        utils.GetEnvInt("OUTBOX_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL", 10),
			EventIPLimiterInterval:                 utils.GetEnvInt("OUTBOX_RELAY_EVENT_IP_LIMITER_INTERVAL", 60),
			EventIPLimiterMaxTokens:                utils.GetEnvInt("OUTBOX_RELAY_EVENT_IP_LIMITER_MAX_TOKENS", 100),
			AllowEmptyFilters:                      utils.GetEnvBool("OUTBOX_RELAY_ALLOW_EMPTY_FILTERS", false),
			AllowComplexFilters:                    utils.GetEnvBool("OUTBOX_RELAY_ALLOW_COMPLEX_FILTERS", false),
			ConnectionRateLimiterTokensPerInterval: utils.GetEnvInt("OUTBOX_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL", 3),
			ConnectionRateLimiterInterval:          utils.GetEnvInt("OUTBOX_RELAY_CONNECTION_RATE_LIMITER_INTERVAL", 1),
			ConnectionRateLimiterMaxTokens:         utils.GetEnvInt("OUTBOX_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS", 9),
		},
	}
}

func (l *Limits) PrettyPrintLimits() {
	prettyPrintLimits("Private relay limits", l.PrivateRelayLimits)
	prettyPrintLimits("Chat relay limits", l.ChatRelayLimits)
	prettyPrintLimits("Inbox relay limits", l.InboxRelayLimits)
	prettyPrintLimits("Outbox relay limits", l.OutboxRelayLimits)
}

func prettyPrintLimits(label string, limits interface{}) {
	b, _ := json.MarshalIndent(limits, "", "  ")
	log.Printf("🚧 %s:\n%s\n", label, string(b))
}
