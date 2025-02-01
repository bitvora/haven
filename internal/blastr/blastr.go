package blastr

import (
	"context"
	"log"
	"time"

	"github.com/bitvora/haven/internal/config"
	"github.com/nbd-wtf/go-nostr"
)

func Blast(cfg config.Config, pool *nostr.SimplePool, ev *nostr.Event) {
	ctx := context.Background()
	for _, url := range cfg.BlastrRelays {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		relay, err := pool.EnsureRelay(url)
		if err != nil {
			cancel()
			log.Println("error connecting to relay", relay, err)
			continue
		}
		relay.Publish(ctx, *ev)
		cancel()
	}
	log.Println("🔫 blasted", ev.ID, "to", len(cfg.BlastrRelays), "relays")
}
