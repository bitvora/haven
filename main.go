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
		relay = outboxRelay
	} else if relayType == "/private" {
		relay = privateRelay
	} else if relayType == "/chat" {
		relay = chatRelay
	} else if relayType == "/inbox" {
		relay = inboxRelay
	} else {
		relay = outboxRelay
	}

	relay.ServeHTTP(w, r)
}
