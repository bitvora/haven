package main

import (
	"context"
	"log"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

var (
	pubkeyFollowerCount = make(map[string]int)
	oneHopNetwork       []string
	wot                 []string
	wotRelays           []string
	wotMap              map[string]bool
)

func refreshTrustNetwork() {
	ctx := context.Background()
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)

	defer cancel()
	ownerPubkey := nPubToPubkey(config.OwnerNpub)

	filter := nostr.Filter{
		Authors: []string{ownerPubkey},
		Kinds:   []int{nostr.KindFollowList},
	}

	pool.FetchManyReplaceable(timeoutCtx, config.ImportSeedRelays, filter).Range(func(_ nostr.ReplaceableKey, ev *nostr.Event) bool {
		for contact := range ev.Tags.FindAll("p") {
			pubkeyFollowerCount[contact[1]]++
			appendOneHopNetwork(contact[1])
		}

		return true
	})

	log.Println("🌐 building web of trust graph")
	nPubkeys := uint(0)
	for i := 0; i < len(oneHopNetwork); i += 100 {
		timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		done := make(chan struct{})

		end := i + 100
		if end > len(oneHopNetwork) {
			end = len(oneHopNetwork)
		}

		filter = nostr.Filter{
			Authors: oneHopNetwork[i:end],
			Kinds:   []int{nostr.KindFollowList, nostr.KindRelayListMetadata},
		}

		go func() {
			defer cancel()

			pool.FetchManyReplaceable(timeout, config.ImportSeedRelays, filter).Range(func(_ nostr.ReplaceableKey, ev *nostr.Event) bool {
				nPubkeys++
				for contact := range ev.Tags.FindAll("p") {
					if len(contact) > 1 {
						pubkeyFollowerCount[contact[1]]++
					}
				}

				for relay := range ev.Tags.FindAll("r") {
					appendRelay(relay[1])
				}

				return true
			})
			close(done)
		}()

		select {
		case <-done:
			log.Println("🕸️ analysed", nPubkeys, "followed pubkeys so far")
		case <-timeout.Done():
			log.Println("🚫Timeout while fetching pubkeys, moving to the next batch")
		}
	}
	log.Println("🫂 total network size:", len(pubkeyFollowerCount))
	log.Println("🔗 relays discovered:", len(wotRelays))
	updateWoTMap()
}

func appendRelay(relay string) {
	for _, r := range wotRelays {
		if r == relay {
			return
		}
	}
	wotRelays = append(wotRelays, relay)
}

func appendPubkeyToWoT(pubkey string) {
	for _, pk := range wot {
		if pk == pubkey {
			return
		}
	}

	if len(pubkey) != 64 {
		return
	}

	wot = append(wot, pubkey)
}

func appendOneHopNetwork(pubkey string) {
	for _, pk := range oneHopNetwork {
		if pk == pubkey {
			return
		}
	}

	if len(pubkey) != 64 {
		return
	}

	oneHopNetwork = append(oneHopNetwork, pubkey)
}

func updateWoTMap() {
	wotMapTmp := make(map[string]bool)

	for pubkey, count := range pubkeyFollowerCount {
		if count >= config.ChatRelayMinimumFollowers {
			wotMapTmp[pubkey] = true
			appendPubkeyToWoT(pubkey)
		}
	}

	wotMap = wotMapTmp
	log.Println("🌐 pubkeys with minimum followers: ", len(wotMap), "keys")
}
