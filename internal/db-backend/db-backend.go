package dbbackend

import (
	"context"

	"github.com/bitvora/haven/internal/config"
	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/nbd-wtf/go-nostr"
)

type DBBackend interface {
	Init() error
	Close()
	CountEvents(ctx context.Context, filter nostr.Filter) (int64, error)
	DeleteEvent(ctx context.Context, evt *nostr.Event) error
	QueryEvents(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error)
	SaveEvent(ctx context.Context, evt *nostr.Event) error
	ReplaceEvent(ctx context.Context, evt *nostr.Event) error
	Serial() []byte
}

func NewDBBackend(cfg config.Config, path string) DBBackend {
	switch cfg.DBEngine {
	case "lmdb":
		return &lmdb.LMDBBackend{
			Path:    path,
			MapSize: cfg.LmdbMapSize,
		}
	case "badger":
		return &badger.BadgerBackend{
			Path: path,
		}
	default:
		return &lmdb.LMDBBackend{
			Path:    path,
			MapSize: cfg.LmdbMapSize,
		}
	}
}
