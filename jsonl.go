package main

import (
	"archive/zip"
	"bufio"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/fiatjaf/eventstore"
	"github.com/nbd-wtf/go-nostr"
)

func importJSONL(ctx context.Context) {
	zipFileName := "haven_export.zip"
	slog.Info("🛬 starting import", "file", zipFileName)

	zipFile, err := zip.OpenReader(zipFileName)
	if err != nil {
		slog.Error("❌ error opening zip file", "error", err)
		return
	}
	defer func(zipFile *zip.ReadCloser) {
		err := zipFile.Close()
		if err != nil {
			slog.Error("❌ error closing zip file", "error", err)
		}
	}(zipFile)

	dbs := map[string]DBBackend{
		"private.jsonl": privateDB,
		"chat.jsonl":    chatDB,
		"outbox.jsonl":  outboxDB,
		"inbox.jsonl":   inboxDB,
		"blossom.jsonl": blossomDB,
	}

	for _, file := range zipFile.File {
		db, ok := dbs[file.Name]
		if !ok {
			slog.Warn("⏭️ skipping unknown file in zip", "file", file.Name)
			continue
		}

		slog.Info("📦 importing file to db", "file", file.Name)

		rc, err := file.Open()
		if err != nil {
			slog.Error("❌ error opening zip entry", "file", file.Name, "error", err)
			return
		}

		if err := importDB(ctx, db, rc); err != nil {
			slog.Error("❌ error importing", "file", file.Name, "error", err)
			_ = rc.Close()
			return
		}
		_ = rc.Close()
	}

	slog.Info("✅ import complete", "file", zipFileName)
}

func importDB(ctx context.Context, db DBBackend, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	// Nostr events can be large, increase buffer size if necessary.
	// Default is 64KB, which might be enough for most events, but let's be safe.
	const maxCapacity = 100 * 1024 * 1024 // 100MB
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxCapacity)

	count := 0
	for scanner.Scan() {
		var event nostr.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return err
		}

		if err := db.SaveEvent(ctx, &event); err != nil {
			if errors.Is(err, eventstore.ErrDupEvent) {
				slog.Debug("⏭️ skipping duplicate event", "id", event.ID)
				continue
			}
			return err
		}
		count++
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	slog.Info("📥 imported events", "count", count)
	return nil
}

func exportJSONL(ctx context.Context) {
	dbs := []struct {
		name string
		db   DBBackend
	}{
		{"private.jsonl", privateDB},
		{"chat.jsonl", chatDB},
		{"outbox.jsonl", outboxDB},
		{"inbox.jsonl", inboxDB},
		{"blossom.jsonl", blossomDB},
	}

	zipFileName := "haven_export.zip"
	slog.Info("🛫 starting export", "file", zipFileName)
	zipFile, err := os.Create(zipFileName)
	if err != nil {
		slog.Error("❌ error creating zip file", "error", err)
		return
	}
	defer func(zipFile *os.File) {
		err := zipFile.Close()
		if err != nil {
			slog.Error("❌ error closing zip file", "error", err)
		}
	}(zipFile)

	zipWriter := zip.NewWriter(zipFile)
	defer func(zipWriter *zip.Writer) {
		err := zipWriter.Close()
		if err != nil {
			slog.Error("❌ error closing zip writer", "error", err)
		}
	}(zipWriter)

	for _, entry := range dbs {
		slog.Info("📦 exporting db to file", "file", entry.name)

		header := &zip.FileHeader{
			Name:     entry.name,
			Method:   zip.Deflate,
			Modified: time.Now(),
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			slog.Error("❌ error creating zip entry", "file", entry.name, "error", err)
			return
		}

		if err := exportDB(ctx, entry.db, writer); err != nil {
			slog.Error("❌ error exporting", "file", entry.name, "error", err)
			return
		}
	}

	slog.Info("✅ export complete", "file", zipFileName)
}

func exportDB(ctx context.Context, db DBBackend, w io.Writer) error {
	const limit = 1000
	var lastTimestamp nostr.Timestamp
	count := 0

	var eventBuffer []*nostr.Event

	flushBuffer := func() error {
		for _, e := range eventBuffer {
			if _, err := fmt.Fprintln(w, e); err != nil {
				return err
			}
			count++
		}
		eventBuffer = eventBuffer[:0]
		return nil
	}

	for {
		filter := nostr.Filter{
			Limit: limit,
		}
		if lastTimestamp != 0 {
			filter.Until = &lastTimestamp
		}

		events, err := db.QueryEvents(ctx, filter)
		if err != nil {
			return err
		}

		initialCount := count
		initialBufferSize := len(eventBuffer)

		for event := range events {
			if len(eventBuffer) > 0 && event.CreatedAt != eventBuffer[0].CreatedAt {
				if err := flushBuffer(); err != nil {
					return err
				}
			}

			// Insert into eventBuffer maintaining ID order and skipping duplicates.
			// This works around eventstore LMDB backend that don't guarantee NIP-01
			// REQ order when paginating with a limit.
			//
			// This ensures that:
			// 1. Identical JSONL exports are produced (checksum guarantee) regardless of the
			//    underlying DB implementation.
			// 2. Roundtrip consistency is maintained when importing back into a DB, even
			//    if the original source had duplicates (e.g. due to bugs in older versions
			//    of eventstore and khatru).
			pos, found := slices.BinarySearchFunc(eventBuffer, event, func(a, b *nostr.Event) int {
				return cmp.Compare(a.ID, b.ID)
			})
			if !found {
				eventBuffer = slices.Insert(eventBuffer, pos, event)
			} else {
				slog.Debug("⏭️ skipping duplicated event", "id", event.ID)
			}

			lastTimestamp = event.CreatedAt
		}

		if count == initialCount && len(eventBuffer) == initialBufferSize {
			break
		}
	}

	if err := flushBuffer(); err != nil {
		return err
	}

	slog.Info("📤 exported events", "count", count)

	return nil
}
