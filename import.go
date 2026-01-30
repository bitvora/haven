package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"

	"github.com/bitvora/haven/wot"
)

const layout = "2006-01-02"

func ensureImportRelays() {
	nErrors := 0
	log.Println("🧪 Testing import relays")
	for _, relay := range config.ImportSeedRelays {
		if _, err := pool.EnsureRelay(relay); err != nil {
			nErrors++
			slog.Error("🚫 Error connecting to relay", "relay", relay, "error", err)
		} else {
			slog.Debug("✅ Connected to relay", "relay", relay)
		}
	}
	if nErrors == 0 {
		slog.Info("✅ All relays connected successfully")
	} else if nErrors == len(config.ImportSeedRelays) {
		slog.Error("🚫 Unable to connect to any import relays, check your connectivity and relays_import.json file")
		os.Exit(1)
	} else {
		slog.Warn("⚠️ Some relays failed to connect, proceeding, but this may cause issues")
		slog.Info("ℹ️ If you always see this message during startup, consider removing the relays that are not working from your relays_import.json file")
	}
}

func importOwnerNotes(ctx context.Context) {
	ownerImportedNotes := 0
	nFailedImportNotes := 0
	wdb := eventstore.RelayWrapper{Store: outboxDB}

	startTime, err := time.Parse(layout, config.ImportStartDate)
	if err != nil {
		fmt.Println("Error parsing start date:", err)
		return
	}
	endTime := startTime.Add(240 * time.Hour)

	for {
		startTimestamp := nostr.Timestamp(startTime.Unix())
		endTimestamp := nostr.Timestamp(endTime.Unix())

		filter := nostr.Filter{
			Authors: []string{config.OwnerNpubKey},
			Since:   &startTimestamp,
			Until:   &endTimestamp,
		}

		done := make(chan int, 1)
		timeout := time.Duration(config.ImportOwnerNotesFetchTimeoutSeconds) * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)

		go func() {
			defer cancel()
			batchImportedNotes := 0

			events := pool.FetchMany(ctx, config.ImportSeedRelays, filter)
			for ev := range events {
				if ctx.Err() != nil {
					break // Stop the loop on timeout
				}
				if err := wdb.Publish(ctx, *ev.Event); err != nil {
					log.Println("🚫  error importing note", ev.ID, ":", err)
					nFailedImportNotes++
				}
				batchImportedNotes++
			}
			done <- batchImportedNotes
			close(done)
		}()

		select {
		case batchImportedNotes := <-done:
			ownerImportedNotes += batchImportedNotes
			if batchImportedNotes == 0 {
				log.Printf("ℹ️ No notes found for %s to %s", startTime.Format(layout), endTime.Format(layout))
			} else {
				log.Printf("📦 Imported %d notes from %s to %s", batchImportedNotes, startTime.Format(layout), endTime.Format(layout))
			}
		case <-ctx.Done():
			log.Printf("🚫 Timeout after %v while importing notes from %s to %s", timeout, startTime.Format(layout), endTime.Format(layout))
		}

		startTime = startTime.Add(240 * time.Hour)
		endTime = endTime.Add(240 * time.Hour)

		if startTime.After(time.Now()) {
			log.Println("✅ owner note import complete! Imported", ownerImportedNotes, "notes")
			break
		}
		if nFailedImportNotes > 0 {
			log.Printf("⚠️ Failed to import %d notes", nFailedImportNotes)
		}

		time.Sleep(1 * time.Second) // Avoid bombarding relays with too many requests
	}
}

func importTaggedNotes(ctx context.Context) {
	taggedImportedNotes := 0
	done := make(chan struct{}, 1)
	timeout := time.Duration(config.ImportTaggedNotesFetchTimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	wdbInbox := eventstore.RelayWrapper{Store: inboxDB}
	wdbChat := eventstore.RelayWrapper{Store: chatDB}
	filter := nostr.Filter{
		Tags: nostr.TagMap{
			"p": {config.OwnerNpubKey},
		},
	}

	log.Println("📦 importing inbox notes, please wait up to", timeout)

	go func() {
		events := pool.FetchMany(ctx, config.ImportSeedRelays, filter)
		for ev := range events {
			if ctx.Err() != nil {
				break // Stop the loop on timeout
			}

			if !wot.GetInstance().Has(ctx, ev.Event.PubKey) && ev.Kind != nostr.KindGiftWrap {
				continue
			}
			for tag := range ev.Tags.FindAll("p") {
				if len(tag) < 2 {
					continue
				}
				if tag[1] == config.OwnerNpubKey {
					dbToWrite := wdbInbox
					if ev.Kind == nostr.KindGiftWrap {
						dbToWrite = wdbChat
					}
					if err := dbToWrite.Publish(ctx, *ev.Event); err != nil {
						log.Println("🚫 error importing tagged note", ev.ID, ":", err)
					}
					taggedImportedNotes++
				}
			}
		}
		close(done)
	}()

	select {
	case <-done:
		log.Println("📦 imported", taggedImportedNotes, "tagged notes")
	case <-ctx.Done():
		log.Println("🚫 Timeout after", timeout, "while importing tagged notes")
	}

	log.Println("✅ tagged import complete")
}

func subscribeInboxAndChat(ctx context.Context) {
	wdbInbox := eventstore.RelayWrapper{Store: inboxDB}
	wdbChat := eventstore.RelayWrapper{Store: chatDB}
	startTime := nostr.Timestamp(time.Now().Add(-time.Minute * 5).Unix())
	filter := nostr.Filter{
		Tags: nostr.TagMap{
			"p": {config.OwnerNpubKey},
		},
		Since: &startTime,
	}

	log.Println("📢 subscribing to inbox")

	for ev := range pool.SubscribeMany(ctx, config.ImportSeedRelays, filter) {
		if !wot.GetInstance().Has(ctx, ev.Event.PubKey) && ev.Event.Kind != nostr.KindGiftWrap {
			continue
		}
		for tag := range ev.Event.Tags.FindAll("p") {
			if len(tag) < 2 {
				continue
			}
			if tag[1] == config.OwnerNpubKey {
				dbToPublish := wdbInbox
				if ev.Event.Kind == nostr.KindGiftWrap {
					dbToPublish = wdbChat
				}

				slog.Debug("ℹ️ importing event", "kind", ev.Kind, "id", ev.Event.ID, "relay", ev.Relay.URL)

				if isDuplicate(ctx, dbToPublish, ev.Event) {
					slog.Debug("ℹ️ skipping duplicate event", "id", ev.Event.ID)
					break // Avoid re-importing duplicates
				}

				if err := dbToPublish.Publish(ctx, *ev.Event); err != nil {
					log.Println("🚫 error importing tagged note", ev.Event.ID, ":", "from relay", ev.Relay.URL, ":", err)
					break
				}

				switch ev.Event.Kind {
				case nostr.KindTextNote:
					log.Println("📰 new note in your inbox")
				case nostr.KindReaction:
					log.Println(ev.Event.Content, "new reaction in your inbox")
				case nostr.KindZap:
					log.Println("⚡️ new zap in your inbox")
				case nostr.KindEncryptedDirectMessage:
					log.Println("🔒✉️ new encrypted message in your inbox")
				case nostr.KindGiftWrap:
					log.Println("🎁🔒️✉️ new gift-wrapped message in your chat relay")
				case nostr.KindRepost:
					log.Println("🔁 new repost in your inbox")
				case nostr.KindFollowList:
					// do nothing
				default:
					log.Println("📦 new event kind", ev.Event.Kind, "event in your inbox")
				}
			}
		}
	}
}

func isDuplicate(ctx context.Context, db eventstore.RelayWrapper, event *nostr.Event) bool {
	filter := nostr.Filter{
		IDs:   []string{event.ID},
		Since: &event.CreatedAt,
		Limit: 1,
	}

	events, err := db.QuerySync(ctx, filter)
	if err != nil {
		log.Println("🚫 error querying for event", event.ID, ":", err)
		return false
	}

	return len(events) > 0
}
