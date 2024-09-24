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

	filters := []nostr.Filter{{
		Authors: []string{ownerPubkey},
		Kinds:   []int{nostr.KindFollowList},
	}}

	for ev := range pool.SubManyEose(timeoutCtx, config.ImportSeedRelays, filters) {
		for _, contact := range ev.Event.Tags.GetAll([]string{"p"}) {
			pubkeyFollowerCount[contact[1]]++
			appendOneHopNetwork(contact[1])
		}
	}

	log.Println("🌐 building web of trust graph")
	for i := 0; i < len(oneHopNetwork); i += 100 {
		timeout, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()

		end := i + 100
		if end > len(oneHopNetwork) {
			end = len(oneHopNetwork)
		}

		filters = []nostr.Filter{{
			Authors: oneHopNetwork[i:end],
			Kinds:   []int{nostr.KindFollowList, nostr.KindRelayListMetadata},
		}}

		for ev := range pool.SubManyEose(timeout, config.ImportSeedRelays, filters) {
			for _, contact := range ev.Event.Tags.GetAll([]string{"p"}) {
				if len(contact) > 1 {
					pubkeyFollowerCount[contact[1]]++
				}
			}

			for _, relay := range ev.Event.Tags.GetAll([]string{"r"}) {
				appendRelay(relay[1])
			}

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
