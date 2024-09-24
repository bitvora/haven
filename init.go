package main

import (
	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/fiatjaf/khatru"
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

func getPrivateDB() lmdb.LMDBBackend {
	return lmdb.LMDBBackend{
		Path: "db/private",
	}
}

func getChatDB() lmdb.LMDBBackend {
	return lmdb.LMDBBackend{
		Path: "db/chat",
	}
}

func getOutboxDB() lmdb.LMDBBackend {
	return lmdb.LMDBBackend{
		Path: "db/outbox",
	}
}

func getInboxDB() lmdb.LMDBBackend {
	return lmdb.LMDBBackend{
		Path: "db/inbox",
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

	privateRelay.Info.Name = config.PrivateRelayName
	privateRelay.Info.PubKey = nPubToPubkey(config.PrivateRelayNpub)
	privateRelay.Info.Description = config.PrivateRelayDescription
	privateRelay.Info.Icon = config.PrivateRelayIcon
	privateRelay.Info.Version = config.RelayVersion
	privateRelay.Info.Software = config.RelaySoftware
	privateRelay.ServiceURL = "https://" + config.RelayURL + "/private"

	chatRelay.Info.Name = config.ChatRelayName
	chatRelay.Info.PubKey = nPubToPubkey(config.ChatRelayNpub)
	chatRelay.Info.Description = config.ChatRelayDescription
	chatRelay.Info.Icon = config.ChatRelayIcon
	chatRelay.Info.Version = config.RelayVersion
	chatRelay.Info.Software = config.RelaySoftware
	chatRelay.ServiceURL = "https://" + config.RelayURL + "/chat"

	outboxRelay.Info.Name = config.OutboxRelayName
	outboxRelay.Info.PubKey = nPubToPubkey(config.OutboxRelayNpub)
	outboxRelay.Info.Description = config.OutboxRelayDescription
	outboxRelay.Info.Icon = config.OutboxRelayIcon
	outboxRelay.Info.Version = config.RelayVersion
	outboxRelay.Info.Software = config.RelaySoftware

	inboxRelay.Info.Name = config.InboxRelayName
	inboxRelay.Info.PubKey = nPubToPubkey(config.InboxRelayNpub)
	inboxRelay.Info.Description = config.InboxRelayDescription
	inboxRelay.Info.Icon = config.InboxRelayIcon
	inboxRelay.Info.Version = config.RelayVersion
	inboxRelay.Info.Software = config.RelaySoftware
	inboxRelay.ServiceURL = "https://" + config.RelayURL + "/inbox"
}
