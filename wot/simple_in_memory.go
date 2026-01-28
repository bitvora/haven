package wot

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"sync/atomic"
	"time"

	"github.com/nbd-wtf/go-nostr"
	"github.com/puzpuzpuz/xsync/v4"
)

type SimpleInMemory struct {
	pubkeys atomic.Pointer[map[string]bool]

	// Dependencies for Refresh
	Pool            *nostr.SimplePool
	OwnerPubkey     string
	SeedRelays      []string
	WotFetchTimeout int
	MinFollowers    int
}

func NewSimpleInMemory(pool *nostr.SimplePool, ownerPubkey string, seedRelays []string, wotFetchTimeout int, minFollowers int) *SimpleInMemory {
	return &SimpleInMemory{
		Pool:            pool,
		OwnerPubkey:     ownerPubkey,
		SeedRelays:      seedRelays,
		WotFetchTimeout: wotFetchTimeout,
		MinFollowers:    minFollowers,
	}
}

func (wt *SimpleInMemory) Has(pubkey string) bool {
	m := wt.pubkeys.Load()
	if m == nil {
		return false
	}
	return (*m)[pubkey]
}

func (wt *SimpleInMemory) Init() {
	wt.Refresh(context.Background())
}

func (wt *SimpleInMemory) Refresh(ctx context.Context) {
	pubkeyFollowerCount := xsync.NewMap[string, *atomic.Int64]()
	relaysDiscovered := xsync.NewMap[string, bool]()
	oneHopNetwork := make(map[string]bool)

	timeout := time.Duration(wt.WotFetchTimeout) * time.Second
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	filter := nostr.Filter{
		Authors: []string{wt.OwnerPubkey},
		Kinds:   []int{nostr.KindFollowList},
	}

	events := wt.Pool.FetchMany(timeoutCtx, wt.SeedRelays, filter)
	for ev := range events {
		for contact := range ev.Event.Tags.FindAll("p") {
			val, _ := pubkeyFollowerCount.LoadOrStore(contact[1], &atomic.Int64{})
			val.Add(1)
			oneHopNetwork[contact[1]] = true
		}
	}

	slog.Info("🛜 fetching Nostr events to build WoT")
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

			events := wt.Pool.FetchMany(timeoutCtx, wt.SeedRelays, filter)
			for ev := range events {
				eventsAnalysed.Add(1)
				for contact := range ev.Tags.FindAll("p") {
					if len(contact) > 1 {
						pubkeyFollowersCount, _ := pubkeyFollowerCount.LoadOrStore(contact[1], &atomic.Int64{})
						pubkeyFollowersCount.Add(1)
					}
				}

				for relay := range ev.Tags.FindAll("r") {
					relaysDiscovered.Store(relay[1], true)
				}
			}
			close(done)
		}()

		select {
		case <-done:
			slog.Info("🕸️ analysing Nostr events", "count", eventsAnalysed.Load())
		case <-timeoutCtx.Done():
			slog.Error("🚫 timeout while fetching events, moving to the next batch")
		}
	}

	// Split analysis into batches of 100 pubkeys
	keys := slices.Collect(maps.Keys(oneHopNetwork))
	for batch := range slices.Chunk(keys, 100) {
		processBatch(batch)
	}

	slog.Info("📈 totals", "🫂pubkeys", pubkeyFollowerCount.Size(), "🔗relays", relaysDiscovered.Size())

	// Filter out pubkeys with less than minimum followers
	newPubkeys := make(map[string]bool)
	minimumFollowers := int64(wt.MinFollowers)
	pubkeyFollowerCount.Range(func(pubkey string, count *atomic.Int64) bool {
		if count.Load() >= minimumFollowers {
			newPubkeys[pubkey] = true
		}
		return true
	})

	slog.Info("🫥 eliminating pubkeys without minimum followers", "minimum", wt.MinFollowers, "kept", len(newPubkeys))

	wt.pubkeys.Store(&newPubkeys)
}
