package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"text/template"

	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/blossom"
	"github.com/nbd-wtf/go-nostr"
	"github.com/puzpuzpuz/xsync/v3"
	"github.com/spf13/afero"
)

var (
	mainRelay = khatru.NewRelay()
	subRelays = xsync.NewMapOf[string, *khatru.Relay]()
	pool      = nostr.NewSimplePool(context.Background())
	config    = loadConfig()
	fs        afero.Fs
)

func main() {
	importFlag := flag.Bool("import", false, "Run the importNotes function after initializing relays")
	flag.Parse()

	nostr.InfoLogger = log.New(io.Discard, "", 0)
	green := "\033[32m"
	reset := "\033[0m"
	fmt.Println(green + art + reset)
	log.Println("🚀 haven is booting up")
	fs = afero.NewOsFs()
	fs.MkdirAll(config.BlossomPath, 0755)

	initRelays()

	go func() {
		refreshTrustNetwork()

		if *importFlag {
			log.Println("📦 importing notes")
			importOwnerNotes()
			importTaggedNotes()
			return
		}

		go subscribeInbox()
		go backupDatabase()
	}()

	http.HandleFunc("/", dynamicRelayHandler)

	addr := fmt.Sprintf("%s:%d", config.RelayBindAddress, config.RelayPort)

	log.Printf("🔗 listening at %s", addr)
	http.ListenAndServe(addr, nil)
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

		privateRelay.StoreEvent = append(privateRelay.StoreEvent, privateDB.SaveEvent)
		privateRelay.QueryEvents = append(privateRelay.QueryEvents, privateDB.QueryEvents)
		privateRelay.DeleteEvent = append(privateRelay.DeleteEvent, privateDB.DeleteEvent)
		privateRelay.CountEvents = append(privateRelay.CountEvents, privateDB.CountEvents)

		privateRelay.RejectFilter = append(privateRelay.RejectFilter, func(ctx context.Context, filter nostr.Filter) (bool, string) {
			authenticatedUser := khatru.GetAuthed(ctx)
			if authenticatedUser == nPubToPubkey(config.OwnerNpub) {
				return false, ""
			}

			return true, "auth-required: this query requires you to be authenticated"
		})

		privateRelay.RejectEvent = append(privateRelay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
			authenticatedUser := khatru.GetAuthed(ctx)

			if authenticatedUser == nPubToPubkey(config.OwnerNpub) {
				return false, ""
			}

			return true, "auth-required: publishing this event requires authentication"
		})

		mux := privateRelay.Router()

		mux.HandleFunc("/private", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("templates/index.html"))
			data := struct {
				RelayName        string
				RelayPubkey      string
				RelayDescription string
				RelayURL         string
			}{
				RelayName:        config.PrivateRelayName,
				RelayPubkey:      nPubToPubkey(config.PrivateRelayNpub),
				RelayDescription: config.PrivateRelayDescription,
				RelayURL:         "wss://" + config.RelayURL + "/private",
			}
			err := tmpl.Execute(w, data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		return privateRelay

	case "/chat":
		chatRelay.OnConnect = append(chatRelay.OnConnect, func(ctx context.Context) {
			khatru.RequestAuth(ctx)
		})

		chatRelay.StoreEvent = append(chatRelay.StoreEvent, chatDB.SaveEvent)
		chatRelay.QueryEvents = append(chatRelay.QueryEvents, chatDB.QueryEvents)
		chatRelay.DeleteEvent = append(chatRelay.DeleteEvent, chatDB.DeleteEvent)
		chatRelay.CountEvents = append(chatRelay.CountEvents, chatDB.CountEvents)

		chatRelay.RejectFilter = append(chatRelay.RejectFilter, func(ctx context.Context, filter nostr.Filter) (bool, string) {
			authenticatedUser := khatru.GetAuthed(ctx)

			if !wotMap[authenticatedUser] {
				return true, "you must be in the web of trust to chat with the relay owner"
			}

			return false, ""
		})

		allowedKinds := []int{
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
			nostr.KindGiftWrap,
		}

		chatRelay.RejectEvent = append(chatRelay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
			for _, kind := range allowedKinds {
				if event.Kind == kind {
					return false, ""
				}
			}

			return true, "only gift wrapped DMs are allowed"
		})

		mux := chatRelay.Router()

		mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("templates/index.html"))
			data := struct {
				RelayName        string
				RelayPubkey      string
				RelayDescription string
				RelayURL         string
			}{
				RelayName:        config.ChatRelayName,
				RelayPubkey:      nPubToPubkey(config.ChatRelayNpub),
				RelayDescription: config.ChatRelayDescription,
				RelayURL:         "wss://" + config.RelayURL + "/chat",
			}
			err := tmpl.Execute(w, data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		return chatRelay

	case "/inbox":
		inboxRelay.StoreEvent = append(inboxRelay.StoreEvent, inboxDB.SaveEvent)
		inboxRelay.QueryEvents = append(inboxRelay.QueryEvents, inboxDB.QueryEvents)
		inboxRelay.DeleteEvent = append(inboxRelay.DeleteEvent, inboxDB.DeleteEvent)
		inboxRelay.CountEvents = append(inboxRelay.CountEvents, inboxDB.CountEvents)

		inboxRelay.RejectEvent = append(inboxRelay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
			if !wotMap[event.PubKey] {
				return true, "you must be in the web of trust to post to this relay"
			}

			if event.Kind == nostr.KindEncryptedDirectMessage {
				return true, "only gift wrapped DMs are supported"
			}

			for _, tag := range event.Tags.GetAll([]string{"p"}) {
				if tag[1] == inboxRelay.Info.PubKey {
					return false, ""
				}
			}

			return true, "you can only post notes if you've tagged the owner of this relay"
		})

		mux := inboxRelay.Router()

		mux.HandleFunc("/inbox", func(w http.ResponseWriter, r *http.Request) {
			tmpl := template.Must(template.ParseFiles("templates/index.html"))
			data := struct {
				RelayName        string
				RelayPubkey      string
				RelayDescription string
				RelayURL         string
			}{
				RelayName:        config.InboxRelayName,
				RelayPubkey:      nPubToPubkey(config.InboxRelayNpub),
				RelayDescription: config.InboxRelayDescription,
				RelayURL:         "wss://" + config.RelayURL + "/inbox",
			}
			err := tmpl.Execute(w, data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		return inboxRelay

	default: // default to outbox
		outboxRelay.StoreEvent = append(outboxRelay.StoreEvent, outboxDB.SaveEvent, func(ctx context.Context, event *nostr.Event) error {
			go blast(event)
			return nil
		})
		outboxRelay.QueryEvents = append(outboxRelay.QueryEvents, outboxDB.QueryEvents)
		outboxRelay.DeleteEvent = append(outboxRelay.DeleteEvent, outboxDB.DeleteEvent)
		outboxRelay.CountEvents = append(outboxRelay.CountEvents, outboxDB.CountEvents)

		outboxRelay.RejectEvent = append(outboxRelay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
			if event.PubKey == nPubToPubkey(config.OwnerNpub) {
				return false, ""
			}
			return true, "only notes signed by the owner of this relay are allowed"
		})

		mux := outboxRelay.Router()

		mux.HandleFunc(relayType, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			tmpl := template.Must(template.ParseFiles("templates/index.html"))
			data := struct {
				RelayName        string
				RelayPubkey      string
				RelayDescription string
				RelayURL         string
			}{
				RelayName:        config.OutboxRelayName,
				RelayPubkey:      nPubToPubkey(config.OutboxRelayNpub),
				RelayDescription: config.OutboxRelayDescription,
				RelayURL:         "wss://" + config.RelayURL + "/outbox",
			}
			err := tmpl.Execute(w, data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		bl := blossom.New(outboxRelay, "https://"+config.RelayURL)
		bl.Store = blossom.EventStoreBlobIndexWrapper{Store: outboxDB, ServiceURL: bl.ServiceURL}
		bl.StoreBlob = append(bl.StoreBlob, func(ctx context.Context, sha256 string, body []byte) error {

			file, err := fs.Create(config.BlossomPath + sha256)
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, bytes.NewReader(body)); err != nil {
				return err
			}
			return nil
		})
		bl.LoadBlob = append(bl.LoadBlob, func(ctx context.Context, sha256 string) (io.Reader, error) {
			return fs.Open(config.BlossomPath + sha256)
		})
		bl.DeleteBlob = append(bl.DeleteBlob, func(ctx context.Context, sha256 string) error {
			return fs.Remove(config.BlossomPath + sha256)
		})
		bl.RejectUpload = append(bl.RejectUpload, func(ctx context.Context, event *nostr.Event, size int, ext string) (bool, string, int) {
			if event.PubKey == nPubToPubkey(config.OwnerNpub) {
				return false, ext, size
			}

			return true, "only notes signed by the owner of this relay are allowed", 0
		})

		return outboxRelay
	}
}
