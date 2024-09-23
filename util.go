package main

import (
	"github.com/nbd-wtf/go-nostr/nip19"
)

func nPubToPubkey(nPub string) string {
	_, v, err := nip19.Decode(nPub)
	if err != nil {
		panic(err)
	}
	return v.(string)
}
