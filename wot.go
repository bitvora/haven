package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/puzpuzpuz/xsync/v4"
)

type Wot interface {
	Has(pubkey string) bool
}

var wotInstance atomic.Value

func Get() Wot {
	return wotInstance.Load().(Wot)
}

type InMemoryWot struct {
	pubkeys map[string]struct{}
}

func (wt *InMemoryWot) Has(pubkey string) bool {
	_, ok := wt.pubkeys[pubkey]
	return ok
}

func refreshTrustNetwork(ctx context.Context) {
	wt := &InMemoryWot{
		pubkeys: make(map[string]struct{}),
	}
	pubkeyFollowerCount := xsync.NewMapOf[string, *atomic.Int64]()
	relaysDiscovered := xsync.NewMapOf[string, struct{}]()
	oneHopNetworkMap := make(map[string]struct{})

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
			val, _ := pubkeyFollowerCount.LoadOrStore(contact[1], &atomic.Int64{})
			val.Add(1)
			oneHopNetworkMap[contact[1]] = struct{}{}
		}
	}

	log.Println("🌐 building web of trust graph")
	var eventsAnalysed atomic.Int64

	processBatch := func(pubkeys []string) {
		timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
		done := make(chan struct{})

		filter := nostr.Filter{
			Authors: pubkeys,
			Kinds:   []int{nostr.KindFollowList, nostr.KindRelayListMetadata},
		}

		go func() {
			defer cancel()

			events := pool.FetchMany(timeoutCtx, config.ImportSeedRelays, filter)
			for ev := range events {
				eventsAnalysed.Add(1)
				for contact := range ev.Tags.FindAll("p") {
					if len(contact) > 1 {
						pubkeyFollowersCount, _ := pubkeyFollowerCount.LoadOrStore(contact[1], &atomic.Int64{})
						pubkeyFollowersCount.Add(1)
					}
				}

				for relay := range ev.Tags.FindAll("r") {
					relaysDiscovered.Store(relay[1], struct{}{})
				}
			}
			close(done)
		}()

		select {
		case <-done:
			log.Println("🕸️ analysed", eventsAnalysed.Load(), "Nostr events so far")
		case <-timeoutCtx.Done():
			log.Println("🚫Timeout while fetching events, moving to the next batch")
		}
	}

	// Split analysis into batches of 100 pubkeys
	batch := make([]string, 0, 100)
	for key := range oneHopNetworkMap {
		batch = append(batch, key)
		if len(batch) == 100 {
			processBatch(batch)
			batch = make([]string, 0, 100)
		}
	}
	if len(batch) > 0 {
		processBatch(batch)
	}

	log.Println("🫂 total network size:", pubkeyFollowerCount.Size())
	log.Println("🔗 relays discovered:", relaysDiscovered.Size())

	// Filter out pubkeys with less than minimum followers
	minimumFollowers := int64(config.ChatRelayMinimumFollowers)
	pubkeyFollowerCount.Range(func(pubkey string, count *atomic.Int64) bool {
		if count.Load() >= minimumFollowers {
			wt.pubkeys[pubkey] = struct{}{}
		}
		return true
	})

	log.Println("🌐 pubkeys with minimum followers: ", len(wt.pubkeys), "keys")

	// Atomic replacement is safe
	wotInstance.Store(wt)
}

func periodicRefreshWot(ctx context.Context) {
	ticker := time.NewTicker(config.WotRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshTrustNetwork(ctx)
		}
	}
}
