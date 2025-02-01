package chatrelay

import (
	"github.com/bitvora/haven/internal/config"
	dbbackend "github.com/bitvora/haven/internal/db-backend"
	"github.com/fiatjaf/khatru"
)

type ChatRelay struct {
	Relay     *khatru.Relay
	DbBackend dbbackend.DBBackend
}

func NewChatRelay(cfg config.Config) *ChatRelay {
	return &ChatRelay{
		Relay:     khatru.NewRelay(),
		DbBackend: dbbackend.NewDBBackend(cfg, "db/chat"),
	}
}
