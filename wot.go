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
	timeout := time.Duration(config.WotFetchTimeoutSeconds) * time.Second
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()
	ownerPubkey := nPubToPubkey(config.OwnerNpub)

	filter := nostr.Filter{
		Authors: []string{ownerPubkey},
		Kinds:   []int{nostr.KindFollowList},
	}

	events := pool.FetchMany(timeoutCtx, config.ImportSeedRelays, filter)
	for ev := range events {
		for contact := range ev.Event.Tags.FindAll("p") {
			pubkeyFollowerCount[contact[1]]++
			appendOneHopNetwork(contact[1])
		}
	}

	log.Println("🌐 building web of trust graph")
	nPubkeys := uint(0)
	for i := 0; i < len(oneHopNetwork); i += 100 {
		end := i + 100
		if end > len(oneHopNetwork) {
			end = len(oneHopNetwork)
		}

		filter = nostr.Filter{
			Authors: oneHopNetwork[i:end],
			Kinds:   []int{nostr.KindFollowList, nostr.KindRelayListMetadata},
		}

		// this does not need to be a goroutine, since we
		// were waiting for "done" before moving to the next batch
		// making it syncronous ends the race condition with pubkeyFollowerCount
		func() {
			// make sure the timeout is not already cancelled
			timeoutCtx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()

			events := pool.FetchMany(timeoutCtx, config.ImportSeedRelays, filter)
			for ev := range events {
				nPubkeys++
				for contact := range ev.Tags.FindAll("p") {
					if len(contact) > 1 {
						pubkeyFollowerCount[contact[1]]++
					}
				}

				for relay := range ev.Tags.FindAll("r") {
					appendRelay(relay[1])
				}
			}
		}()

		log.Println("🕸️ analysed", nPubkeys, "followed pubkeys so far")
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
