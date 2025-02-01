package wot

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/bitvora/haven/internal/config"
	"github.com/bitvora/haven/internal/utils"
	"github.com/nbd-wtf/go-nostr"
)

type WoT struct {
	mu                  sync.Mutex
	webOfTrustMap       map[string]bool
	pubkeyFollowerCount map[string]int
	oneHopNetwork       []string
	webOfTrust          []string
	webOfTrustRelays    []string
}

func NewWoT() *WoT {
	return &WoT{
		webOfTrustMap:       make(map[string]bool),
		pubkeyFollowerCount: make(map[string]int),
		oneHopNetwork:       []string{},
		webOfTrust:          []string{},
		webOfTrustRelays:    []string{},
	}
}

func (wot *WoT) RefreshTrustNetwork(cfg config.Config, pool *nostr.SimplePool) {
	wot.mu.Lock()
	defer wot.mu.Unlock()

	ctx := context.Background()
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)

	defer cancel()
	ownerPubkey := utils.NPubToPubkey(cfg.OwnerNpub)

	filters := []nostr.Filter{{
		Authors: []string{ownerPubkey},
		Kinds:   []int{nostr.KindFollowList},
	}}

	for ev := range pool.SubManyEose(timeoutCtx, cfg.ImportSeedRelays, filters) {
		for _, contact := range ev.Event.Tags.GetAll([]string{"p"}) {
			wot.pubkeyFollowerCount[contact[1]]++
			wot.appendOneHopNetwork(contact[1])
		}
	}

	log.Println("🌐 building web of trust graph")
	for i := 0; i < len(wot.oneHopNetwork); i += 100 {
		timeout, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()

		end := i + 100
		if end > len(wot.oneHopNetwork) {
			end = len(wot.oneHopNetwork)
		}

		filters = []nostr.Filter{{
			Authors: wot.oneHopNetwork[i:end],
			Kinds:   []int{nostr.KindFollowList, nostr.KindRelayListMetadata},
		}}

		for ev := range pool.SubManyEose(timeout, cfg.ImportSeedRelays, filters) {
			for _, contact := range ev.Event.Tags.GetAll([]string{"p"}) {
				if len(contact) > 1 {
					wot.pubkeyFollowerCount[contact[1]]++
				}
			}

			for _, relay := range ev.Event.Tags.GetAll([]string{"r"}) {
				wot.appendRelay(relay[1])
			}

		}
	}
	log.Println("🫂 total network size:", len(wot.pubkeyFollowerCount))
	log.Println("🔗 relays discovered:", len(wot.webOfTrustRelays))
	wot.updateWoTMap(cfg)
}

func (wot *WoT) IsInTrustNetwork(pubkey string) bool {
	wot.mu.Lock()
	defer wot.mu.Unlock()

	return wot.webOfTrustMap[pubkey]
}

func (wot *WoT) appendRelay(relay string) {
	wot.mu.Lock()
	defer wot.mu.Unlock()

	for _, r := range wot.webOfTrustRelays {
		if r == relay {
			return
		}
	}
	wot.webOfTrustRelays = append(wot.webOfTrustRelays, relay)
}

func (wot *WoT) appendPubkeyToWoT(pubkey string) {
	wot.mu.Lock()
	defer wot.mu.Unlock()

	for _, pk := range wot.webOfTrust {
		if pk == pubkey {
			return
		}
	}

	if len(pubkey) != 64 {
		return
	}

	wot.webOfTrust = append(wot.webOfTrust, pubkey)
}

func (wot *WoT) appendOneHopNetwork(pubkey string) {
	wot.mu.Lock()
	defer wot.mu.Unlock()

	for _, pk := range wot.oneHopNetwork {
		if pk == pubkey {
			return
		}
	}

	if len(pubkey) != 64 {
		return
	}

	wot.oneHopNetwork = append(wot.oneHopNetwork, pubkey)
}

func (wot *WoT) updateWoTMap(cfg config.Config) {
	wot.mu.Lock()
	defer wot.mu.Unlock()

	wotMapTmp := make(map[string]bool)

	for pubkey, count := range wot.pubkeyFollowerCount {
		if count >= cfg.ChatRelayMinimumFollowers {
			wotMapTmp[pubkey] = true
			wot.appendPubkeyToWoT(pubkey)
		}
	}

	wot.webOfTrustMap = wotMapTmp
	log.Println("🌐 pubkeys with minimum followers: ", len(wot.webOfTrustMap), "keys")
}
