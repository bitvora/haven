package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"text/template"
	"time"

	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/blossom"
	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"
)

var (
	privateRelay = khatru.NewRelay()
	privateDB    = getPrivateDB()
)

var (
	chatRelay = khatru.NewRelay()
	chatDB    = getChatDB()
)

var (
	outboxRelay = khatru.NewRelay()
	outboxDB    = getOutboxDB()
)

var (
	inboxRelay = khatru.NewRelay()
	inboxDB    = getInboxDB()
)

type DBBackend interface {
	Init() error
	Close()
	CountEvents(ctx context.Context, filter nostr.Filter) (int64, error)
	DeleteEvent(ctx context.Context, evt *nostr.Event) error
	QueryEvents(ctx context.Context, filter nostr.Filter) (chan *nostr.Event, error)
	SaveEvent(ctx context.Context, evt *nostr.Event) error
	Serial() []byte
}

func getPrivateDB() DBBackend {
	switch config.DBEngine {
	case "lmdb":
		return &lmdb.LMDBBackend{
			Path: "db/private",
		}
	case "badger":
		return &badger.BadgerBackend{
			Path: "db/private",
		}
	default:
		return &lmdb.LMDBBackend{
			Path: "db/private",
		}
	}
}

func getChatDB() DBBackend {
	switch config.DBEngine {
	case "lmdb":
		return &lmdb.LMDBBackend{
			Path: "db/chat",
		}
	case "badger":
		return &badger.BadgerBackend{
			Path: "db/chat",
		}
	default:
		return &lmdb.LMDBBackend{
			Path: "db/chat",
		}
	}
}

func getOutboxDB() DBBackend {
	switch config.DBEngine {
	case "lmdb":
		return &lmdb.LMDBBackend{
			Path: "db/outbox",
		}
	case "badger":
		return &badger.BadgerBackend{
			Path: "db/outbox",
		}
	default:
		return &lmdb.LMDBBackend{
			Path: "db/outbox",
		}
	}
}

func getInboxDB() DBBackend {
	switch config.DBEngine {
	case "lmdb":
		return &lmdb.LMDBBackend{
			Path: "db/inbox",
		}
	case "badger":
		return &badger.BadgerBackend{
			Path: "db/inbox",
		}
	default:
		return &lmdb.LMDBBackend{
			Path: "db/inbox",
		}
	}
}

func initRelays() {
	if err := privateDB.Init(); err != nil {
		panic(err)
	}

	if err := chatDB.Init(); err != nil {
		panic(err)
	}

	if err := outboxDB.Init(); err != nil {
		panic(err)
	}

	if err := inboxDB.Init(); err != nil {
		panic(err)
	}

	initRelayLimits()

	privateRelay.Info.Name = config.PrivateRelayName
	privateRelay.Info.PubKey = nPubToPubkey(config.PrivateRelayNpub)
	privateRelay.Info.Description = config.PrivateRelayDescription
	privateRelay.Info.Icon = config.PrivateRelayIcon
	privateRelay.Info.Version = config.RelayVersion
	privateRelay.Info.Software = config.RelaySoftware
	privateRelay.ServiceURL = "https://" + config.RelayURL + "/private"

	if !privateRelayLimits.AllowEmptyFilters {
		privateRelay.RejectFilter = append(privateRelay.RejectFilter, policies.NoEmptyFilters)
	}

	if !privateRelayLimits.AllowComplexFilters {
		privateRelay.RejectFilter = append(privateRelay.RejectFilter, policies.NoComplexFilters)
	}

	privateRelay.RejectEvent = append(privateRelay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(
			privateRelayLimits.EventIPLimiterTokensPerInterval,
			time.Minute*time.Duration(privateRelayLimits.EventIPLimiterInterval),
			privateRelayLimits.EventIPLimiterMaxTokens,
		),
	)

	privateRelay.RejectConnection = append(privateRelay.RejectConnection,
		policies.ConnectionRateLimiter(
			privateRelayLimits.ConnectionRateLimiterTokensPerInterval,
			time.Minute*time.Duration(privateRelayLimits.ConnectionRateLimiterInterval),
			privateRelayLimits.ConnectionRateLimiterMaxTokens,
		),
	)

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

	chatRelay.Info.Name = config.ChatRelayName
	chatRelay.Info.PubKey = nPubToPubkey(config.ChatRelayNpub)
	chatRelay.Info.Description = config.ChatRelayDescription
	chatRelay.Info.Icon = config.ChatRelayIcon
	chatRelay.Info.Version = config.RelayVersion
	chatRelay.Info.Software = config.RelaySoftware
	chatRelay.ServiceURL = "https://" + config.RelayURL + "/chat"

	if !chatRelayLimits.AllowEmptyFilters {
		chatRelay.RejectFilter = append(chatRelay.RejectFilter, policies.NoEmptyFilters)
	}

	if !chatRelayLimits.AllowComplexFilters {
		chatRelay.RejectFilter = append(chatRelay.RejectFilter, policies.NoComplexFilters)
	}

	chatRelay.RejectEvent = append(chatRelay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(
			chatRelayLimits.EventIPLimiterTokensPerInterval,
			time.Minute*time.Duration(chatRelayLimits.EventIPLimiterInterval),
			chatRelayLimits.EventIPLimiterMaxTokens,
		),
	)

	chatRelay.RejectConnection = append(chatRelay.RejectConnection,
		policies.ConnectionRateLimiter(
			chatRelayLimits.ConnectionRateLimiterTokensPerInterval,
			time.Minute*time.Duration(chatRelayLimits.ConnectionRateLimiterInterval),
			chatRelayLimits.ConnectionRateLimiterMaxTokens,
		),
	)

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

	mux = chatRelay.Router()

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

	outboxRelay.Info.Name = config.OutboxRelayName
	outboxRelay.Info.PubKey = nPubToPubkey(config.OutboxRelayNpub)
	outboxRelay.Info.Description = config.OutboxRelayDescription
	outboxRelay.Info.Icon = config.OutboxRelayIcon
	outboxRelay.Info.Version = config.RelayVersion
	outboxRelay.Info.Software = config.RelaySoftware
	outboxRelay.ServiceURL = "https://" + config.RelayURL

	if !outboxRelayLimits.AllowEmptyFilters {
		outboxRelay.RejectFilter = append(outboxRelay.RejectFilter, policies.NoEmptyFilters)
	}

	if !outboxRelayLimits.AllowComplexFilters {
		outboxRelay.RejectFilter = append(outboxRelay.RejectFilter, policies.NoComplexFilters)
	}

	outboxRelay.RejectEvent = append(outboxRelay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(
			outboxRelayLimits.EventIPLimiterTokensPerInterval,
			time.Minute*time.Duration(outboxRelayLimits.EventIPLimiterInterval),
			outboxRelayLimits.EventIPLimiterMaxTokens,
		),
	)

	outboxRelay.RejectConnection = append(outboxRelay.RejectConnection,
		policies.ConnectionRateLimiter(
			outboxRelayLimits.ConnectionRateLimiterTokensPerInterval,
			time.Minute*time.Duration(outboxRelayLimits.ConnectionRateLimiterInterval),
			outboxRelayLimits.ConnectionRateLimiterMaxTokens,
		),
	)

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

	mux = outboxRelay.Router()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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

	inboxRelay.Info.Name = config.InboxRelayName
	inboxRelay.Info.PubKey = nPubToPubkey(config.InboxRelayNpub)
	inboxRelay.Info.Description = config.InboxRelayDescription
	inboxRelay.Info.Icon = config.InboxRelayIcon
	inboxRelay.Info.Version = config.RelayVersion
	inboxRelay.Info.Software = config.RelaySoftware
	inboxRelay.ServiceURL = "https://" + config.RelayURL + "/inbox"

	if !inboxRelayLimits.AllowEmptyFilters {
		inboxRelay.RejectFilter = append(inboxRelay.RejectFilter, policies.NoEmptyFilters)
	}

	if !inboxRelayLimits.AllowComplexFilters {
		inboxRelay.RejectFilter = append(inboxRelay.RejectFilter, policies.NoComplexFilters)
	}

	inboxRelay.RejectEvent = append(inboxRelay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(
			inboxRelayLimits.EventIPLimiterTokensPerInterval,
			time.Minute*time.Duration(inboxRelayLimits.EventIPLimiterInterval),
			inboxRelayLimits.EventIPLimiterMaxTokens,
		),
	)

	inboxRelay.RejectConnection = append(inboxRelay.RejectConnection,
		policies.ConnectionRateLimiter(
			inboxRelayLimits.ConnectionRateLimiterTokensPerInterval,
			time.Minute*time.Duration(inboxRelayLimits.ConnectionRateLimiterInterval),
			inboxRelayLimits.ConnectionRateLimiterMaxTokens,
		),
	)

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

	mux = inboxRelay.Router()

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

}
