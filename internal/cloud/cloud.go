package cloud

import (
	"context"
	"io"
)

// Uploader is an interface for uploading objects to a cloud provider.
type Uploader interface {
	Upload(ctx context.Context, bucketName string, objectName string, r io.Reader, size int64, contentType string) error
}

// Downloader is an interface for downloading objects from a cloud provider.
type Downloader interface {
	Download(ctx context.Context, bucketName string, objectName string) (io.ReadCloser, error)
}

// Provider is an interface that can both upload and download objects.
type Provider interface {
	Uploader
	Downloader
}
