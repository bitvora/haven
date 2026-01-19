package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type Wot struct {
	Set   map[string]struct{}
	Mutex sync.Mutex
}

func (wt *Wot) Has(pubkey string) bool {
	wt.Mutex.Lock()
	_, ok := wt.Set[pubkey]
	wt.Mutex.Unlock()

	return ok
}

var (
	pubkeyFollowerCount = make(map[string]int)
	oneHopNetwork       []string
	wot                 []string
	wotRelays           []string
	wotMap              Wot
)

func refreshTrustNetwork(ctx context.Context) {
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
		timeoutCtx, cancel = context.WithTimeout(ctx, timeout)
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
			close(done)
		}()

		select {
		case <-done:
			log.Println("🕸️ analysed", nPubkeys, "followed pubkeys so far")
		case <-timeoutCtx.Done():
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
	wotTmp := make(map[string]struct{}, len(pubkeyFollowerCount))

	for pubkey, count := range pubkeyFollowerCount {
		if count >= config.ChatRelayMinimumFollowers {
			wotTmp[pubkey] = struct{}{}
			appendPubkeyToWoT(pubkey)
		}
	}

	wotMap.Mutex.Lock()
	wotMap.Set = wotTmp
	wotMap.Mutex.Unlock()

	log.Println("🌐 pubkeys with minimum followers: ", len(wotMap.Set), "keys")
}
