package main

import (
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
		_, v, err := nip19.Decode(nPub)
		if err != nil {
			panic(err)
		}
		pubkeys = append(pubkeys, v.(string))
	}
	return pubkeys
}
