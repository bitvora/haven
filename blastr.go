package main

import (
	"context"
	"log"

	"github.com/nbd-wtf/go-nostr"
)

func blast(ev *nostr.Event) {
	ctx := context.Background()
	for _, relay := range config.BlastrRelays {
		go blastRoutine(ctx, relay, ev)
	}
}

func blastRoutine(ctx context.Context, relay string, ev *nostr.Event) {
	connect, err := nostr.RelayConnect(ctx, relay)
	if err != nil {
		log.Println("error connecting to relay", relay, err)
	}
	connect.Publish(ctx, *ev)
	log.Println("🔫 blasted to", relay)
}
