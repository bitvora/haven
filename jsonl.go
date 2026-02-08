package main

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
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
			rc.Close()
			return
		}
		rc.Close()
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
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

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

	if err := zipWriter.Close(); err != nil {
		slog.Error("❌ error closing zip writer", "error", err)
		return
	}

	slog.Info("✅ export complete", "file", zipFileName)
}

func exportDB(ctx context.Context, db DBBackend, w io.Writer) error {
	var lastTimestamp nostr.Timestamp
	var lastID string
	limit := 1000

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

		count := 0
		var currentLastEvent *nostr.Event
		foundLastID := (lastID == "")

		for event := range events {
			if !foundLastID {
				if event.ID == lastID {
					slog.Debug("🔍 found last ID", "id", lastID)
					foundLastID = true
				} else {
					slog.Debug("⏭️ skipping event", "id", event.ID, "lastID", lastID)
				}
				continue
			}

			line, err := json.Marshal(event)
			if err != nil {
				return err
			}
			if _, err := w.Write(line); err != nil {
				return err
			}
			if _, err := w.Write([]byte("\n")); err != nil {
				return err
			}

			currentLastEvent = event
			count++
		}

		if count == 0 {
			break
		}

		lastTimestamp = currentLastEvent.CreatedAt
		lastID = currentLastEvent.ID
	}

	return nil
}
