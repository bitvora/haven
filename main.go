package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"

	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/afero"

	"github.com/bitvora/haven/wot"
)

var (
	pool   *nostr.SimplePool
	config = loadConfig()
	fs     afero.Fs
)

func main() {
	defer log.Println("🔌 HAVEN is shutting down")
	importFlag := flag.Bool("import", false, "Run the importNotes function after initializing relays")
	importJSONLFlag := flag.Bool("import-jsonl", false, "Import relay data from a jsonl zip file")
	exportFlag := flag.Bool("export-jsonl", false, "Export all relay data to a jsonl zip file")
	flag.Parse()

	nostr.InfoLogger = log.New(io.Discard, "", 0)
	slog.SetLogLoggerLevel(getLogLevelFromConfig())
	green := "\033[32m"
	reset := "\033[0m"
	fmt.Println(green + art + reset)
	log.Println("🚀 HAVEN", config.RelayVersion, "is booting up")
	fs = afero.NewOsFs()
	if err := fs.MkdirAll(config.BlossomPath, 0755); err != nil {
		log.Fatal("🚫 error creating blossom path:", err)
	}

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool = nostr.NewSimplePool(mainCtx, nostr.WithPenaltyBox())

	ensureImportRelays()

	wotModel := wot.NewSimpleInMemory(
		pool,
		config.OwnerNpubKey,
		config.ImportSeedRelays,
		config.WotDepth,
		config.WotMinimumFollowers,
		config.WotFetchTimeoutSeconds,
	)
	wot.Initialize(mainCtx, wotModel)

	initRelays(mainCtx)

	if *importFlag {
		log.Println("📦 importing notes")
		importOwnerNotes(mainCtx)
		importTaggedNotes(mainCtx)
		return
	}

	if *importJSONLFlag {
		importJSONL(mainCtx)
		return
	}

	if *exportFlag {
		exportJSONL(mainCtx)
		return
	}

	go func() {
		go subscribeInboxAndChat(mainCtx)
		go backupDatabase(mainCtx)
		go wot.PeriodicRefresh(mainCtx, config.WotRefreshInterval)
	}()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("templates/static"))))
	http.HandleFunc("/", dynamicRelayHandler)

	addr := fmt.Sprintf("%s:%d", config.RelayBindAddress, config.RelayPort)

	log.Printf("🔗 listening at %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("🚫 error starting server:", err)
	}
}

func dynamicRelayHandler(w http.ResponseWriter, r *http.Request) {
	var relay *khatru.Relay
	relayType := r.URL.Path

	switch relayType {
	case "/private":
		relay = privateRelay
	case "/chat":
		relay = chatRelay
	case "/inbox":
		relay = inboxRelay
	case "":
		relay = outboxRelay
	default:
		relay = outboxRelay
	}

	relay.ServeHTTP(w, r)
}

func getLogLevelFromConfig() slog.Level {
	switch config.LogLevel {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo // Default level
	}
}
