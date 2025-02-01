package importr

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bitvora/haven/internal/config"
	dbbackend "github.com/bitvora/haven/internal/db-backend"
	"github.com/bitvora/haven/internal/utils"
	"github.com/bitvora/haven/internal/wot"
	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
)

type Importr struct {
	mu                  sync.Mutex
	layout              string
	ownerImportedNotes  int
	taggedImportedNotes int
	wot                 *wot.WoT
}

func NewImportr(layout string, wot *wot.WoT) *Importr {
	return &Importr{
		layout:              layout,
		ownerImportedNotes:  0,
		taggedImportedNotes: 0,
		wot:                 wot,
	}
}

func (i *Importr) ImportOwnerNotes(cfg config.Config, pool *nostr.SimplePool, outboxDB dbbackend.DBBackend) {
	i.mu.Lock()
	defer i.mu.Unlock()

	ctx := context.Background()
	wdb := eventstore.RelayWrapper{Store: outboxDB}

	startTime, err := time.Parse(i.layout, cfg.ImportStartDate)
	if err != nil {
		fmt.Println("Error parsing start date:", err)
		return
	}
	endTime := startTime.Add(240 * time.Hour)

	for {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		startTimestamp := nostr.Timestamp(startTime.Unix())
		endTimestamp := nostr.Timestamp(endTime.Unix())

		filters := []nostr.Filter{{
			Authors: []string{utils.NPubToPubkey(cfg.OwnerNpub)},
			Since:   &startTimestamp,
			Until:   &endTimestamp,
		}}

		for ev := range pool.SubManyEose(ctx, cfg.ImportSeedRelays, filters) {
			wdb.Publish(ctx, *ev.Event)
			i.ownerImportedNotes++
		}
		log.Println("📦 imported", i.ownerImportedNotes, "owner notes")
		time.Sleep(5 * time.Second)

		startTime = startTime.Add(240 * time.Hour)
		endTime = endTime.Add(240 * time.Hour)

		if startTime.After(time.Now()) {
			log.Println("✅ owner note import complete! ")
			break
		}
	}
}

func (i *Importr) ImportTaggedNotes(cfg config.Config, pool *nostr.SimplePool, inboxDB dbbackend.DBBackend) {
	i.mu.Lock()
	defer i.mu.Unlock()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	wdb := eventstore.RelayWrapper{Store: inboxDB}
	filters := []nostr.Filter{{
		Tags: nostr.TagMap{
			"p": {utils.NPubToPubkey(cfg.OwnerNpub)},
		},
	}}

	log.Println("📦 importing inbox notes, please wait 2 minutes")
	for ev := range pool.SubMany(ctx, cfg.ImportSeedRelays, filters) {
		if !i.wot.IsInTrustNetwork(ev.Event.PubKey) {
			continue
		}
		for _, tag := range ev.Event.Tags.GetAll([]string{"p"}) {
			if len(tag) < 2 {
				continue
			}
			if tag[1] == utils.NPubToPubkey(cfg.OwnerNpub) {
				wdb.Publish(ctx, *ev.Event)
				i.taggedImportedNotes++
			}
		}
	}
	log.Println("📦 imported", i.taggedImportedNotes, "tagged notes")
	log.Println("✅ tagged import complete. please restart the relay")
}

func (i *Importr) SubscribeInbox(cfg config.Config, pool *nostr.SimplePool, inboxDB dbbackend.DBBackend) {
	i.mu.Lock()
	defer i.mu.Unlock()

	ctx := context.Background()
	wdb := eventstore.RelayWrapper{Store: inboxDB}
	startTime := nostr.Timestamp(time.Now().Add(-time.Minute * 5).Unix())
	filters := []nostr.Filter{{
		Tags: nostr.TagMap{
			"p": {utils.NPubToPubkey(cfg.OwnerNpub)},
		},
		Since: &startTime,
	}}

	log.Println("📢 subscribing to inbox")
	for ev := range pool.SubMany(ctx, cfg.ImportSeedRelays, filters) {
		if !i.wot.IsInTrustNetwork(ev.Event.PubKey) {
			continue
		}
		for _, tag := range ev.Event.Tags.GetAll([]string{"p"}) {
			if len(tag) < 2 {
				continue
			}
			if tag[1] == utils.NPubToPubkey(cfg.OwnerNpub) {
				wdb.Publish(ctx, *ev.Event)
				switch ev.Event.Kind {
				case nostr.KindTextNote:
					log.Println("📰 new note in your inbox")
				case nostr.KindReaction:
					log.Println(ev.Event.Content, "new reaction in your inbox")
				case nostr.KindZap:
					log.Println("⚡️ new zap in your inbox")
				case nostr.KindEncryptedDirectMessage:
					log.Println("🔒 new encrypted message in your inbox")
				case nostr.KindRepost:
					log.Println("🔁 new repost in your inbox")
				case nostr.KindFollowList:
					// do nothing
				default:
					log.Println("📦 new event in your inbox")
				}
				i.taggedImportedNotes++
			}
		}
	}
}
