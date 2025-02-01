package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/bitvora/haven/internal/backupr"
	"github.com/bitvora/haven/internal/blastr"
	chatrelay "github.com/bitvora/haven/internal/chat-relay"
	"github.com/bitvora/haven/internal/config"
	"github.com/bitvora/haven/internal/importr"
	inboxrelay "github.com/bitvora/haven/internal/inbox-relay"
	"github.com/bitvora/haven/internal/limits"
	outboxrelay "github.com/bitvora/haven/internal/outbox-relay"
	privaterelay "github.com/bitvora/haven/internal/private-relay"
	"github.com/bitvora/haven/internal/utils"
	"github.com/bitvora/haven/internal/wot"
	"github.com/fiatjaf/khatru"
	"github.com/fiatjaf/khatru/blossom"
	"github.com/fiatjaf/khatru/policies"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/afero"
)

func main() {
	cfg := config.LoadConfig()
	importFlag := flag.Bool("import", false, "Run the importNotes function after initializing relays")
	flag.Parse()

	nostr.InfoLogger = log.New(io.Discard, "", 0)
	green := "\033[32m"
	reset := "\033[0m"
	fmt.Println(green + config.Art() + reset)
	log.Println("🚀 haven is booting up")
	fs := afero.NewOsFs()
	fs.MkdirAll(cfg.BlossomPath, 0755)

	backupr := backupr.NewBackupr(cfg)
	layout := "2006-01-02"
	webOfTrust := wot.NewWoT()
	importr := importr.NewImportr(layout, webOfTrust)
	pool := nostr.NewSimplePool(context.Background())
	privateRelay := privaterelay.NewPrivateRelay(cfg)
	chatRelay := chatrelay.NewChatRelay(cfg)
	inboxRelay := inboxrelay.NewInboxRelay(cfg)
	outboxRelay := outboxrelay.NewOutboxRelay(cfg)

	initRelays(cfg, fs, pool, webOfTrust, privateRelay, chatRelay, inboxRelay, outboxRelay)

	go func() {
		webOfTrust.RefreshTrustNetwork(cfg, pool)

		if *importFlag {
			log.Println("📦 importing notes")
			importr.ImportOwnerNotes(cfg, pool, privateRelay.DbBackend)
			importr.ImportTaggedNotes(cfg, pool, inboxRelay.DbBackend)
			return
		}

		go importr.SubscribeInbox(cfg, pool, inboxRelay.DbBackend)
		go backupr.BackupDatabase()
	}()

	http.HandleFunc(
		"/",
		dynamicRelayHandler(cfg, privateRelay, chatRelay, inboxRelay, outboxRelay),
	)

	addr := fmt.Sprintf("%s:%d", cfg.RelayBindAddress, cfg.RelayPort)

	log.Printf("🔗 listening at %s", addr)
	http.ListenAndServe(addr, nil)
}

func dynamicRelayHandler(
	cfg config.Config,
	privateRelay *privaterelay.PrivateRelay,
	chatRelay *chatrelay.ChatRelay,
	inboxRelay *inboxrelay.InboxRelay,
	outboxRelay *outboxrelay.OutboxRelay,
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		relayType := r.URL.Path
		if relayType == "" {
			privateRelay.Relay.ServeHTTP(w, r)
		} else if relayType == "/private" {
			privateRelay.Relay.ServeHTTP(w, r)
		} else if relayType == "/chat" {
			chatRelay.Relay.ServeHTTP(w, r)
		} else if relayType == "/inbox" {
			inboxRelay.Relay.ServeHTTP(w, r)
		} else {
			outboxRelay.Relay.ServeHTTP(w, r)
		}
	}
}

func initRelays(
	cfg config.Config,
	fs afero.Fs,
	pool *nostr.SimplePool,
	webOfTrust *wot.WoT,
	privateRelay *privaterelay.PrivateRelay,
	chatRelay *chatrelay.ChatRelay,
	inboxRelay *inboxrelay.InboxRelay,
	outboxRelay *outboxrelay.OutboxRelay,
) {
	log.Println("🔄 initializing relays")
	if err := privateRelay.DbBackend.Init(); err != nil {
		log.Fatal("failed to initialize private relay database: ", err)
	}
	if err := chatRelay.DbBackend.Init(); err != nil {
		log.Fatal("failed to initialize chat relay database: ", err)
	}
	if err := outboxRelay.DbBackend.Init(); err != nil {
		log.Fatal("failed to initialize outbox relay database: ", err)
	}
	if err := inboxRelay.DbBackend.Init(); err != nil {
		log.Fatal("failed to initialize inbox relay database: ", err)
	}

	relayLimits := limits.NewLimits()
	relayLimits.PrettyPrintLimits()

	privateRelay.Relay.Info.Name = cfg.PrivateRelayName
	privateRelay.Relay.Info.PubKey = utils.NPubToPubkey(cfg.PrivateRelayNpub)
	privateRelay.Relay.Info.Description = cfg.PrivateRelayDescription
	privateRelay.Relay.Info.Icon = cfg.PrivateRelayIcon
	privateRelay.Relay.Info.Version = cfg.RelayVersion
	privateRelay.Relay.Info.Software = cfg.RelaySoftware
	privateRelay.Relay.ServiceURL = "https://" + cfg.RelayURL + "/private"

	if !relayLimits.PrivateRelayLimits.AllowEmptyFilters {
		privateRelay.Relay.RejectFilter = append(privateRelay.Relay.RejectFilter, policies.NoEmptyFilters)
	}

	if !relayLimits.PrivateRelayLimits.AllowComplexFilters {
		privateRelay.Relay.RejectFilter = append(privateRelay.Relay.RejectFilter, policies.NoComplexFilters)
	}

	privateRelay.Relay.RejectEvent = append(privateRelay.Relay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(
			relayLimits.PrivateRelayLimits.EventIPLimiterTokensPerInterval,
			time.Minute*time.Duration(relayLimits.PrivateRelayLimits.EventIPLimiterInterval),
			relayLimits.PrivateRelayLimits.EventIPLimiterMaxTokens,
		),
	)

	privateRelay.Relay.RejectConnection = append(privateRelay.Relay.RejectConnection,
		policies.ConnectionRateLimiter(
			relayLimits.PrivateRelayLimits.ConnectionRateLimiterTokensPerInterval,
			time.Minute*time.Duration(relayLimits.PrivateRelayLimits.ConnectionRateLimiterInterval),
			relayLimits.PrivateRelayLimits.ConnectionRateLimiterMaxTokens,
		),
	)

	privateRelay.Relay.OnConnect = append(privateRelay.Relay.OnConnect, func(ctx context.Context) {
		khatru.RequestAuth(ctx)
	})

	privateRelay.Relay.StoreEvent = append(privateRelay.Relay.StoreEvent, privateRelay.DbBackend.SaveEvent)
	privateRelay.Relay.QueryEvents = append(privateRelay.Relay.QueryEvents, privateRelay.DbBackend.QueryEvents)
	privateRelay.Relay.DeleteEvent = append(privateRelay.Relay.DeleteEvent, privateRelay.DbBackend.DeleteEvent)
	privateRelay.Relay.CountEvents = append(privateRelay.Relay.CountEvents, privateRelay.DbBackend.CountEvents)
	privateRelay.Relay.ReplaceEvent = append(privateRelay.Relay.ReplaceEvent, privateRelay.DbBackend.ReplaceEvent)

	privateRelay.Relay.RejectFilter = append(privateRelay.Relay.RejectFilter, func(ctx context.Context, filter nostr.Filter) (bool, string) {
		authenticatedUser := khatru.GetAuthed(ctx)
		if authenticatedUser == utils.NPubToPubkey(cfg.OwnerNpub) {
			return false, ""
		}

		return true, "auth-required: this query requires you to be authenticated"
	})

	privateRelay.Relay.RejectEvent = append(privateRelay.Relay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
		authenticatedUser := khatru.GetAuthed(ctx)

		if authenticatedUser == utils.NPubToPubkey(cfg.OwnerNpub) {
			return false, ""
		}

		return true, "auth-required: publishing this event requires authentication"
	})

	mux := privateRelay.Relay.Router()

	mux.HandleFunc("GET /private", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		data := struct {
			RelayName        string
			RelayPubkey      string
			RelayDescription string
			RelayURL         string
		}{
			RelayName:        cfg.PrivateRelayName,
			RelayPubkey:      utils.NPubToPubkey(cfg.PrivateRelayNpub),
			RelayDescription: cfg.PrivateRelayDescription,
			RelayURL:         "wss://" + cfg.RelayURL + "/private",
		}
		err := tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	chatRelay.Relay.Info.Name = cfg.ChatRelayName
	chatRelay.Relay.Info.PubKey = utils.NPubToPubkey(cfg.ChatRelayNpub)
	chatRelay.Relay.Info.Description = cfg.ChatRelayDescription
	chatRelay.Relay.Info.Icon = cfg.ChatRelayIcon
	chatRelay.Relay.Info.Version = cfg.RelayVersion
	chatRelay.Relay.Info.Software = cfg.RelaySoftware
	chatRelay.Relay.ServiceURL = "https://" + cfg.RelayURL + "/chat"

	if !relayLimits.ChatRelayLimits.AllowEmptyFilters {
		chatRelay.Relay.RejectFilter = append(chatRelay.Relay.RejectFilter, policies.NoEmptyFilters)
	}

	if !relayLimits.ChatRelayLimits.AllowComplexFilters {
		chatRelay.Relay.RejectFilter = append(chatRelay.Relay.RejectFilter, policies.NoComplexFilters)
	}

	chatRelay.Relay.RejectEvent = append(chatRelay.Relay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(
			relayLimits.ChatRelayLimits.EventIPLimiterTokensPerInterval,
			time.Minute*time.Duration(relayLimits.ChatRelayLimits.EventIPLimiterInterval),
			relayLimits.ChatRelayLimits.EventIPLimiterMaxTokens,
		),
	)

	chatRelay.Relay.RejectConnection = append(chatRelay.Relay.RejectConnection,
		policies.ConnectionRateLimiter(
			relayLimits.ChatRelayLimits.ConnectionRateLimiterTokensPerInterval,
			time.Minute*time.Duration(relayLimits.ChatRelayLimits.ConnectionRateLimiterInterval),
			relayLimits.ChatRelayLimits.ConnectionRateLimiterMaxTokens,
		),
	)

	chatRelay.Relay.OnConnect = append(chatRelay.Relay.OnConnect, func(ctx context.Context) {
		khatru.RequestAuth(ctx)
	})

	chatRelay.Relay.StoreEvent = append(chatRelay.Relay.StoreEvent, chatRelay.DbBackend.SaveEvent)
	chatRelay.Relay.QueryEvents = append(chatRelay.Relay.QueryEvents, chatRelay.DbBackend.QueryEvents)
	chatRelay.Relay.DeleteEvent = append(chatRelay.Relay.DeleteEvent, chatRelay.DbBackend.DeleteEvent)
	chatRelay.Relay.CountEvents = append(chatRelay.Relay.CountEvents, chatRelay.DbBackend.CountEvents)
	chatRelay.Relay.ReplaceEvent = append(chatRelay.Relay.ReplaceEvent, chatRelay.DbBackend.ReplaceEvent)

	chatRelay.Relay.RejectFilter = append(chatRelay.Relay.RejectFilter, func(ctx context.Context, filter nostr.Filter) (bool, string) {
		authenticatedUser := khatru.GetAuthed(ctx)

		if !webOfTrust.IsInTrustNetwork(authenticatedUser) {
			return true, "you must be in the web of trust to chat with the relay owner"
		}

		return false, ""
	})

	allowedKinds := []int{
		// Regular kinds
		nostr.KindSimpleGroupChatMessage,
		nostr.KindSimpleGroupThreadedReply,
		nostr.KindSimpleGroupThread,
		nostr.KindSimpleGroupReply,
		nostr.KindChannelMessage,
		nostr.KindChannelHideMessage,

		nostr.KindGiftWrap,

		nostr.KindSimpleGroupPutUser,
		nostr.KindSimpleGroupRemoveUser,
		nostr.KindSimpleGroupEditMetadata,
		nostr.KindSimpleGroupDeleteEvent,
		nostr.KindSimpleGroupCreateGroup,
		nostr.KindSimpleGroupDeleteGroup,
		nostr.KindSimpleGroupCreateInvite,
		nostr.KindSimpleGroupJoinRequest,
		nostr.KindSimpleGroupLeaveRequest,

		// Addressable kinds
		nostr.KindSimpleGroupMetadata,
		nostr.KindSimpleGroupAdmins,
		nostr.KindSimpleGroupMembers,
		nostr.KindSimpleGroupRoles,
	}

	chatRelay.Relay.RejectEvent = append(chatRelay.Relay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
		for _, kind := range allowedKinds {
			if event.Kind == kind {
				return false, ""
			}
		}

		return true, "only gift wrapped DMs are allowed"
	})

	mux = chatRelay.Relay.Router()

	mux.HandleFunc("GET /chat", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		data := struct {
			RelayName        string
			RelayPubkey      string
			RelayDescription string
			RelayURL         string
		}{
			RelayName:        cfg.ChatRelayName,
			RelayPubkey:      utils.NPubToPubkey(cfg.ChatRelayNpub),
			RelayDescription: cfg.ChatRelayDescription,
			RelayURL:         "wss://" + cfg.RelayURL + "/chat",
		}
		err := tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	outboxRelay.Relay.Info.Name = cfg.OutboxRelayName
	outboxRelay.Relay.Info.PubKey = utils.NPubToPubkey(cfg.OutboxRelayNpub)
	outboxRelay.Relay.Info.Description = cfg.OutboxRelayDescription
	outboxRelay.Relay.Info.Icon = cfg.OutboxRelayIcon
	outboxRelay.Relay.Info.Version = cfg.RelayVersion
	outboxRelay.Relay.Info.Software = cfg.RelaySoftware
	outboxRelay.Relay.ServiceURL = "https://" + cfg.RelayURL

	if !relayLimits.OutboxRelayLimits.AllowEmptyFilters {
		outboxRelay.Relay.RejectFilter = append(outboxRelay.Relay.RejectFilter, policies.NoEmptyFilters)
	}

	if !relayLimits.OutboxRelayLimits.AllowComplexFilters {
		outboxRelay.Relay.RejectFilter = append(outboxRelay.Relay.RejectFilter, policies.NoComplexFilters)
	}

	outboxRelay.Relay.RejectEvent = append(outboxRelay.Relay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(
			relayLimits.OutboxRelayLimits.EventIPLimiterTokensPerInterval,
			time.Minute*time.Duration(relayLimits.OutboxRelayLimits.EventIPLimiterInterval),
			relayLimits.OutboxRelayLimits.EventIPLimiterMaxTokens,
		),
	)

	outboxRelay.Relay.RejectConnection = append(outboxRelay.Relay.RejectConnection,
		policies.ConnectionRateLimiter(
			relayLimits.OutboxRelayLimits.ConnectionRateLimiterTokensPerInterval,
			time.Minute*time.Duration(relayLimits.OutboxRelayLimits.ConnectionRateLimiterInterval),
			relayLimits.OutboxRelayLimits.ConnectionRateLimiterMaxTokens,
		),
	)

	outboxRelay.Relay.StoreEvent = append(outboxRelay.Relay.StoreEvent, outboxRelay.DbBackend.SaveEvent, func(ctx context.Context, event *nostr.Event) error {
		go blastr.Blast(cfg, pool, event)
		return nil
	})
	outboxRelay.Relay.QueryEvents = append(outboxRelay.Relay.QueryEvents, outboxRelay.DbBackend.QueryEvents)
	outboxRelay.Relay.DeleteEvent = append(outboxRelay.Relay.DeleteEvent, outboxRelay.DbBackend.DeleteEvent)
	outboxRelay.Relay.CountEvents = append(outboxRelay.Relay.CountEvents, outboxRelay.DbBackend.CountEvents)
	outboxRelay.Relay.ReplaceEvent = append(outboxRelay.Relay.ReplaceEvent, outboxRelay.DbBackend.ReplaceEvent)

	outboxRelay.Relay.RejectEvent = append(outboxRelay.Relay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
		if event.PubKey == utils.NPubToPubkey(cfg.OwnerNpub) {
			return false, ""
		}
		return true, "only notes signed by the owner of this relay are allowed"
	})

	mux = outboxRelay.Relay.Router()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		data := struct {
			RelayName        string
			RelayPubkey      string
			RelayDescription string
			RelayURL         string
		}{
			RelayName:        cfg.OutboxRelayName,
			RelayPubkey:      utils.NPubToPubkey(cfg.OutboxRelayNpub),
			RelayDescription: cfg.OutboxRelayDescription,
			RelayURL:         "wss://" + cfg.RelayURL + "/outbox",
		}
		err := tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	bl := blossom.New(outboxRelay.Relay, "https://"+cfg.RelayURL)
	bl.Store = blossom.EventStoreBlobIndexWrapper{Store: outboxRelay.DbBackend, ServiceURL: bl.ServiceURL}
	bl.StoreBlob = append(bl.StoreBlob, func(ctx context.Context, sha256 string, body []byte) error {
		file, err := fs.Create(cfg.BlossomPath + sha256)
		if err != nil {
			return err
		}
		if _, err := io.Copy(file, bytes.NewReader(body)); err != nil {
			return err
		}
		return nil
	})
	bl.LoadBlob = append(bl.LoadBlob, func(ctx context.Context, sha256 string) (io.ReadSeeker, error) {
		return fs.Open(cfg.BlossomPath + sha256)
	})
	bl.DeleteBlob = append(bl.DeleteBlob, func(ctx context.Context, sha256 string) error {
		return fs.Remove(cfg.BlossomPath + sha256)
	})
	bl.RejectUpload = append(bl.RejectUpload, func(ctx context.Context, event *nostr.Event, size int, ext string) (bool, string, int) {
		if event.PubKey == utils.NPubToPubkey(cfg.OwnerNpub) {
			return false, ext, size
		}
		return true, "only notes signed by the owner of this relay are allowed", 403
	})

	inboxRelay.Relay.Info.Name = cfg.InboxRelayName
	inboxRelay.Relay.Info.PubKey = utils.NPubToPubkey(cfg.InboxRelayNpub)
	inboxRelay.Relay.Info.Description = cfg.InboxRelayDescription
	inboxRelay.Relay.Info.Icon = cfg.InboxRelayIcon
	inboxRelay.Relay.Info.Version = cfg.RelayVersion
	inboxRelay.Relay.Info.Software = cfg.RelaySoftware
	inboxRelay.Relay.ServiceURL = "https://" + cfg.RelayURL + "/inbox"

	if !relayLimits.InboxRelayLimits.AllowEmptyFilters {
		inboxRelay.Relay.RejectFilter = append(inboxRelay.Relay.RejectFilter, policies.NoEmptyFilters)
	}

	if !relayLimits.InboxRelayLimits.AllowComplexFilters {
		inboxRelay.Relay.RejectFilter = append(inboxRelay.Relay.RejectFilter, policies.NoComplexFilters)
	}

	inboxRelay.Relay.RejectEvent = append(inboxRelay.Relay.RejectEvent,
		policies.RejectEventsWithBase64Media,
		policies.EventIPRateLimiter(
			relayLimits.InboxRelayLimits.EventIPLimiterTokensPerInterval,
			time.Minute*time.Duration(relayLimits.InboxRelayLimits.EventIPLimiterInterval),
			relayLimits.InboxRelayLimits.EventIPLimiterMaxTokens,
		),
	)

	inboxRelay.Relay.RejectConnection = append(inboxRelay.Relay.RejectConnection,
		policies.ConnectionRateLimiter(
			relayLimits.InboxRelayLimits.ConnectionRateLimiterTokensPerInterval,
			time.Minute*time.Duration(relayLimits.InboxRelayLimits.ConnectionRateLimiterInterval),
			relayLimits.InboxRelayLimits.ConnectionRateLimiterMaxTokens,
		),
	)

	inboxRelay.Relay.StoreEvent = append(inboxRelay.Relay.StoreEvent, inboxRelay.DbBackend.SaveEvent)
	inboxRelay.Relay.QueryEvents = append(inboxRelay.Relay.QueryEvents, inboxRelay.DbBackend.QueryEvents)
	inboxRelay.Relay.DeleteEvent = append(inboxRelay.Relay.DeleteEvent, inboxRelay.DbBackend.DeleteEvent)
	inboxRelay.Relay.CountEvents = append(inboxRelay.Relay.CountEvents, inboxRelay.DbBackend.CountEvents)
	inboxRelay.Relay.ReplaceEvent = append(inboxRelay.Relay.ReplaceEvent, inboxRelay.DbBackend.ReplaceEvent)

	inboxRelay.Relay.RejectEvent = append(inboxRelay.Relay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
		if !webOfTrust.IsInTrustNetwork(event.PubKey) {
			return true, "you must be in the web of trust to post to this relay"
		}

		if event.Kind == nostr.KindEncryptedDirectMessage {
			return true, "only gift wrapped DMs are supported"
		}

		for _, tag := range event.Tags.GetAll([]string{"p"}) {
			if tag[1] == inboxRelay.Relay.Info.PubKey {
				return false, ""
			}
		}
		return true, "you can only post notes if you've tagged the owner of this relay"
	})

	mux = inboxRelay.Relay.Router()

	mux.HandleFunc("GET /inbox", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("templates/index.html"))
		data := struct {
			RelayName        string
			RelayPubkey      string
			RelayDescription string
			RelayURL         string
		}{
			RelayName:        cfg.InboxRelayName,
			RelayPubkey:      utils.NPubToPubkey(cfg.InboxRelayNpub),
			RelayDescription: cfg.InboxRelayDescription,
			RelayURL:         "wss://" + cfg.RelayURL + "/inbox",
		}
		err := tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
