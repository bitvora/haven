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

func (wt *SimpleInMemory) Has(_ context.Context, pubkey string) bool {
	m := wt.pubkeys.Load()
	if m == nil {
		return false
	}
	return (*m)[pubkey]
}

func (wt *SimpleInMemory) Init(ctx context.Context) {
	wt.Refresh(ctx)
}

func (wt *SimpleInMemory) Refresh(ctx context.Context) {
	pubkeyFollowers := xsync.NewMap[string, *xsync.Map[string, bool]]()
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
			followers, _ := pubkeyFollowers.LoadOrStore(contact[1], xsync.NewMap[string, bool]())
			followers.Store(ev.Event.PubKey, true)
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
						followers, _ := pubkeyFollowers.LoadOrStore(contact[1], xsync.NewMap[string, bool]())
						followers.Store(ev.PubKey, true)
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
	minimumFollowers := wt.MinFollowers
	pubkeyFollowers.Range(func(pubkey string, followers *xsync.Map[string, bool]) bool {
		if followers.Size() >= minimumFollowers {
			newPubkeys[pubkey] = true
		}
		return true
	})

	slog.Info("🫥 eliminated pubkeys without minimum followers", "minimum", wt.MinFollowers, "kept", len(newPubkeys))

	wt.pubkeys.Store(&newPubkeys)
}
