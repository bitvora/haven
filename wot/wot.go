package wot

import (
	"context"
	"log"
	"sync/atomic"
	"time"
)

type Model interface {
	Has(pubkey string) bool
}

type Refresher interface {
	Refresh(ctx context.Context)
}

type Initializer interface {
	Init()
}

var wotInstance atomic.Value

func GetInstance() Model {
	val := wotInstance.Load()
	if val == nil {
		return nil
	}
	return val.(Model)
}

func Initialize(model Model) {
	wotInstance.Store(model)
	if initializer, ok := model.(Initializer); ok {
		log.Printf("🌐 Initializing WoT (%T)...\n", model)
		initializer.Init()
		log.Println("✅ WoT initialized")
	}
}

func PeriodicRefresh(interval time.Duration) {
	ctx := context.Background()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			instance := GetInstance()
			if refresher, ok := instance.(Refresher); ok {
				log.Println("🌐 Refreshing WoT...")
				refresher.Refresh(ctx)
				log.Println("✅ WoT refreshed")
			}
		}
	}
}
