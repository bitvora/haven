package main

import (
	"context"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

func blast(ev *nostr.Event) {
	ctx := context.Background()
	for _, url := range config.BlastrRelays {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		relay, err := pool.EnsureRelay(url)
		if err != nil {
			cancel()
			log.Println("error connecting to relay", relay, err)
			return
		}
		relay.Publish(ctx, *ev)
		log.Println("🔫 blasted to", relay)
		cancel()
	}
}
