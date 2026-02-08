package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

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
