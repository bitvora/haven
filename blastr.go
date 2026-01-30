package main

import (
	"context"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

func blast(ctx context.Context, ev *nostr.Event) {
	for _, url := range config.BlastrRelays {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		relay, err := pool.EnsureRelay(url)
		if err != nil {
			cancel()
			log.Println("error connecting to relay", relay, err)
			continue
		}
		if err := relay.Publish(ctx, *ev); err != nil {
			log.Println("🚫 error publishing to relay", relay, err)
		}
		cancel()
	}
	log.Println("🔫 blasted", ev.ID, "to", len(config.BlastrRelays), "relays")
}
