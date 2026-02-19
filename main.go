package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/fiatjaf/khatru"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/afero"

	"github.com/bitvora/haven/pkg/wot"
)

var (
	pool   *nostr.SimplePool
	config = loadConfig()
	fs     afero.Fs
)

func main() {
	nostr.InfoLogger = log.New(io.Discard, "", 0)
	slog.SetLogLoggerLevel(getLogLevelFromConfig())
	green := "\033[32m"
	reset := "\033[0m"
	fmt.Println(green + art + reset)

	mainCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs = afero.NewOsFs()
	if err := fs.MkdirAll(config.BlossomPath, 0755); err != nil {
		log.Fatal("🚫 error creating blossom path:", err)
	}

	pool = nostr.NewSimplePool(mainCtx,
		nostr.WithPenaltyBox(),
		nostr.WithRelayOptions(
			nostr.WithRequestHeader{
				"User-Agent": []string{config.UserAgent},
			}),
	)

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "backup":
			runBackup(mainCtx)
			return
		case "restore":
			runRestore(mainCtx)
			return
		case "import":
			ensureImportRelays()
			runImport(mainCtx)
			return
		case "help":
			printHelp()
			return
		}

		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			printHelp()
			return
		}
	}

	flag.Parse()

	log.Println("🚀 HAVEN", config.RelayVersion, "is booting up")
	defer log.Println("🔌 HAVEN is shutting down")
	log.Println("👥 Number of whitelisted pubkeys:", len(config.WhitelistedPubKeys))
	log.Println("🚷 Number of blacklisted pubkeys:", len(config.BlacklistedPubKeys))

	ensureImportRelays()
	wotModel := wot.NewSimpleInMemory(
		pool,
		config.WhitelistedPubKeys,
		config.ImportSeedRelays,
		config.WotDepth,
		config.WotMinimumFollowers,
		config.WotFetchTimeoutSeconds,
	)
	wot.Initialize(mainCtx, wotModel)
	initRelays(mainCtx)

	go func() {
		go subscribeInboxAndChat(mainCtx)
		go startPeriodicCloudBackups(mainCtx)
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

func printHelp() {
	fmt.Println("haven is a personal nostr relay.")
	fmt.Println()
	fmt.Println("usage: haven [command]")
	fmt.Println()
	fmt.Println("commands:")
	fmt.Println("  backup  - backup the database")
	fmt.Println("  restore - restore the database")
	fmt.Println("  import  - import notes from seed relays")
	fmt.Println("  help    - show this help message")
	fmt.Println()
	fmt.Println("if no command is provided, the relay starts by default.")
	fmt.Println()
	fmt.Println("run 'haven [command] --help' for more information on a command.")
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
