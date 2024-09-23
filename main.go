package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
	"github.com/puzpuzpuz/xsync/v3"
)

var mainRelay = khatru.NewRelay()
var subRelays = xsync.NewMapOf[string, *khatru.Relay]()
var pool *nostr.SimplePool
var config = loadConfig()

func main() {
	importFlag := flag.Bool("import", false, "Run the importNotes function after initializing relays")
	flag.Parse()

	nostr.InfoLogger = log.New(io.Discard, "", 0)
	green := "\033[32m"
	reset := "\033[0m"
	fmt.Println(green + art + reset)
	log.Println("🚀 haven is booting up")
	initRelays()

	if *importFlag {
		log.Println("📦 importing notes")
		importOwnerNotes()
		importTaggedNotes()
		return
	}

	go refreshTrustNetwork()
	go subscribeInbox()

	handler := http.HandlerFunc(dynamicRelayHandler)

	log.Printf("🔗 listening at http://localhost:3355")
	http.ListenAndServe("0.0.0.0:3355", handler)
}

func dynamicRelayHandler(w http.ResponseWriter, r *http.Request) {
	var relay *khatru.Relay
	relayType := r.URL.Path

	if relayType == "" {
		relay = mainRelay
	} else {
		relay, _ = subRelays.LoadOrCompute(relayType, func() *khatru.Relay {
			return makeNewRelay(relayType)
		})
	}

	relay.ServeHTTP(w, r)
}

func makeNewRelay(relayType string) *khatru.Relay {
	switch relayType {
	case "/private":
		privateRelay.OnConnect = append(privateRelay.OnConnect, func(ctx context.Context) {
			khatru.RequestAuth(ctx)
		})

		privateRelay.RejectConnection = append(privateRelay.RejectConnection, func(r *http.Request) bool {
			ctx := r.Context()
			authenticatedUser := khatru.GetAuthed(ctx)

			if authenticatedUser == nPubToPubkey(config.OwnerNpub) {
				return false
			}

			return true
		})

		privateRelay.StoreEvent = append(privateRelay.StoreEvent, privateDB.SaveEvent)
		privateRelay.QueryEvents = append(privateRelay.QueryEvents, privateDB.QueryEvents)
		privateRelay.DeleteEvent = append(privateRelay.DeleteEvent, privateDB.DeleteEvent)

		privateRelay.RejectFilter = append(privateRelay.RejectFilter, func(ctx context.Context, filter nostr.Filter) (bool, string) {
			authenticatedUser := khatru.GetAuthed(ctx)

			if authenticatedUser == nPubToPubkey(config.OwnerNpub) {
				return false, ""
			}

			return true, "only the owner can access this relay"
		})

		return privateRelay

	case "/chat":
		chatRelay.OnConnect = append(chatRelay.OnConnect, func(ctx context.Context) {
			khatru.RequestAuth(ctx)
		})

		chatRelay.RejectConnection = append(chatRelay.RejectConnection, func(r *http.Request) bool {
			ctx := r.Context()
			authenticatedUser := khatru.GetAuthed(ctx)

			if !wotMap[authenticatedUser] {
				return true
			}

			return false
		})

		chatRelay.StoreEvent = append(chatRelay.StoreEvent, chatDB.SaveEvent)
		chatRelay.QueryEvents = append(chatRelay.QueryEvents, chatDB.QueryEvents)
		chatRelay.DeleteEvent = append(chatRelay.DeleteEvent, chatDB.DeleteEvent)

		chatRelay.RejectFilter = append(chatRelay.RejectFilter, func(ctx context.Context, filter nostr.Filter) (bool, string) {
			authenticatedUser := khatru.GetAuthed(ctx)

			if !wotMap[authenticatedUser] {
				return true, "you must be in the web of trust to chat with the relay owner"
			}

			return false, ""
		})

		allowedKinds := []int{
			nostr.KindEncryptedDirectMessage,
			nostr.KindSimpleGroupAddPermission,
			nostr.KindSimpleGroupAddUser,
			nostr.KindSimpleGroupAdmins,
			nostr.KindSimpleGroupChatMessage,
			nostr.KindSimpleGroupCreateGroup,
			nostr.KindSimpleGroupDeleteEvent,
			nostr.KindSimpleGroupDeleteGroup,
			nostr.KindSimpleGroupEditGroupStatus,
			nostr.KindSimpleGroupEditMetadata,
			nostr.KindSimpleGroupJoinRequest,
			nostr.KindSimpleGroupLeaveRequest,
			nostr.KindSimpleGroupMembers,
			nostr.KindSimpleGroupMetadata,
			nostr.KindSimpleGroupRemovePermission,
			nostr.KindSimpleGroupRemoveUser,
			nostr.KindSimpleGroupReply,
			nostr.KindSimpleGroupThread,
			nostr.KindChannelHideMessage,
			nostr.KindChannelMessage,
		}

		chatRelay.RejectEvent = append(chatRelay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
			for _, kind := range allowedKinds {
				if event.Kind == kind {
					return false, ""
				}
			}

			return true, "only direct messages are allowed in this relay"
		})

		return chatRelay

	case "/inbox":
		inboxRelay.StoreEvent = append(inboxRelay.StoreEvent, inboxDB.SaveEvent)
		inboxRelay.QueryEvents = append(inboxRelay.QueryEvents, inboxDB.QueryEvents)

		inboxRelay.RejectEvent = append(inboxRelay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
			if !wotMap[event.PubKey] {
				return true, "you must be in the web of trust to post to this relay"
			}

			for _, tag := range event.Tags.GetAll([]string{"p"}) {
				if tag[1] == inboxRelay.Info.PubKey {
					return false, ""
				}
			}

			return true, "you can only post notes if you've tagged the owner of this relay"
		})

		return inboxRelay

	default: // default to outbox
		outboxRelay.StoreEvent = append(outboxRelay.StoreEvent, outboxDB.SaveEvent, func(ctx context.Context, event *nostr.Event) error {
			go blast(event)
			return nil
		})
		outboxRelay.QueryEvents = append(outboxRelay.QueryEvents, outboxDB.QueryEvents)
		outboxRelay.DeleteEvent = append(outboxRelay.DeleteEvent, outboxDB.DeleteEvent)

		outboxRelay.RejectEvent = append(outboxRelay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
			if event.PubKey == nPubToPubkey(config.OwnerNpub) {
				return false, ""
			}
			return true, "only notes signed by the owner of this relay are allowed"
		})

		return outboxRelay
	}
}
