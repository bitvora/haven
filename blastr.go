package main

import (
	"context"
	"log"

	"github.com/nbd-wtf/go-nostr"
)

func blast(ev *nostr.Event) {
	ctx := context.Background()
	for _, relay := range config.BlastrRelays {
		log.Println("🔫 blasting to", relay)
		connect, err := nostr.RelayConnect(ctx, relay)
		if err != nil {
			log.Println("error connecting to relay", relay, err)
			continue
		}
		connect.Publish(ctx, *ev)
	}
}
