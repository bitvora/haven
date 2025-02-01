package inboxrelay

import (
	"github.com/bitvora/haven/internal/config"
	dbbackend "github.com/bitvora/haven/internal/db-backend"
	"github.com/fiatjaf/khatru"
)

type InboxRelay struct {
	Relay     *khatru.Relay
	DbBackend dbbackend.DBBackend
}

func NewInboxRelay(cfg config.Config) *InboxRelay {
	return &InboxRelay{
		Relay:     khatru.NewRelay(),
		DbBackend: dbbackend.NewDBBackend(cfg, "db/inbox"),
	}
}
