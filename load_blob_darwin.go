//go:build darwin

package main

import (
	"context"
	"io"
)

func loadBlob(ctx context.Context, sha256 string, ext string) (io.ReadSeeker, error) {
	f, err := fs.Open(config.BlossomPath + sha256)
	if err != nil {
		return nil, err
	}

	// In the macOS Sandbox, streaming files via http.ServeContent (sendfile)
	// often fails or cuts off at 512 bytes. See https://github.com/golang/go/issues/70000
	// By wrapping the file, we hide the internal file descriptor from the
	// network stack, forcing Go to use a standard buffered read loop
	// instead of the problematic sendfile system call. This is memory-safe
	// and works for files of any size.
	return struct{ io.ReadSeeker }{f}, nil
}
