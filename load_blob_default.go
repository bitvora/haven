//go:build !darwin

package main

import (
	"context"
	"io"
	"log/slog"
)

func loadBlob(ctx context.Context, sha256 string, ext string) (io.ReadSeeker, error) {
	slog.Debug("loading blob", "sha256", sha256, "ext", ext)

	// For standard Linux/Docker environments, we use efficient
	// streaming directly from the filesystem.
	return fs.Open(config.BlossomPath + sha256)
}
