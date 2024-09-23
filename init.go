package main

import (
	"github.com/fiatjaf/eventstore/badger"
	"github.com/fiatjaf/khatru"
)

var privateRelay = khatru.NewRelay()
var privateDB = getPrivateDB()

var chatRelay = khatru.NewRelay()
var chatDB = getChatDB()

var outboxRelay = khatru.NewRelay()
var outboxDB = getOutboxDB()

var inboxRelay = khatru.NewRelay()
var inboxDB = getInboxDB()

func getPrivateDB() badger.BadgerBackend {
	return badger.BadgerBackend{
		Path: "db/private",
	}
}

func getChatDB() badger.BadgerBackend {
	return badger.BadgerBackend{
		Path: "db/chat",
	}
}

func getOutboxDB() badger.BadgerBackend {
	return badger.BadgerBackend{
		Path: "db/outbox",
	}
}

func getInboxDB() badger.BadgerBackend {
	return badger.BadgerBackend{
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

	chatRelay.Info.Name = config.ChatRelayName
	chatRelay.Info.PubKey = nPubToPubkey(config.ChatRelayNpub)
	chatRelay.Info.Description = config.ChatRelayDescription
	chatRelay.Info.Icon = config.ChatRelayIcon
	chatRelay.Info.Version = config.RelayVersion
	chatRelay.Info.Software = config.RelaySoftware

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
}
