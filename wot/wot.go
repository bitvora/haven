package wot

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"
)

type Model interface {
	Has(ctx context.Context, pubkey string) bool
}

type Refresher interface {
	Refresh(ctx context.Context)
}

type Initializer interface {
	Init(ctx context.Context)
}

var wotInstance atomic.Value

func GetInstance() Model {
	val := wotInstance.Load()
	if val == nil {
		return nil
	}
	return val.(Model)
}

func Initialize(ctx context.Context, model Model) {
	wotInstance.Store(model)
	if initializer, ok := model.(Initializer); ok {
		slog.Info("🌐 Initializing WoT", "model", fmt.Sprintf("%T", model))
		initializer.Init(ctx)
		slog.Info("✅ WoT initialized")
	}
}

func PeriodicRefresh(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			instance := GetInstance()
			if refresher, ok := instance.(Refresher); ok {
				slog.Info("🌐 Refreshing WoT")
				refresher.Refresh(ctx)
				slog.Info("✅ WoT refreshed")
			}
		}
	}
}
