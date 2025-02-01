package privaterelay

import (
	"github.com/bitvora/haven/internal/config"
	dbbackend "github.com/bitvora/haven/internal/db-backend"
	"github.com/fiatjaf/khatru"
)

type PrivateRelay struct {
	Relay     *khatru.Relay
	DbBackend dbbackend.DBBackend
}

func NewPrivateRelay(cfg config.Config) *PrivateRelay {
	return &PrivateRelay{
		Relay:     khatru.NewRelay(),
		DbBackend: dbbackend.NewDBBackend(cfg, "db/private"),
	}
}
