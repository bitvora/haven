package outboxrelay

import (
	"github.com/bitvora/haven/internal/config"
	dbbackend "github.com/bitvora/haven/internal/db-backend"
	"github.com/fiatjaf/khatru"
)

type OutboxRelay struct {
	Relay     *khatru.Relay
	DbBackend dbbackend.DBBackend
}

func NewOutboxRelay(cfg config.Config) *OutboxRelay {
	return &OutboxRelay{
		Relay:     khatru.NewRelay(),
		DbBackend: dbbackend.NewDBBackend(cfg, "db/outbox"),
	}
}
