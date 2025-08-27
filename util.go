package main

import (
	"slices"
	"strings"

	"github.com/nbd-wtf/go-nostr/nip19"
)

func nPubToPubkey(nPub string) string {
	_, v, err := nip19.Decode(nPub)
	if err != nil {
		panic(err)
	}
	return v.(string)
}

func nPubsToPubkeys(nPubs string) []string {
	npubs := strings.Split(nPubs, ",")
	pubkeys := make([]string, 0, len(npubs))

	for _, nPub := range npubs {
		pubkeys = append(pubkeys, nPubToPubkey(nPub))
	}
	return pubkeys
}

// isOwner checks if the given pubkey is in the comma-separated list of owner npubs
func isOwner(ownerNpubs string, pubkey string) bool {
	return slices.Contains(nPubsToPubkeys(ownerNpubs), pubkey)
}
