package cloud

import (
	"context"
	"io"
)

// Uploader is an interface for uploading objects to a cloud provider.
type Uploader interface {
	Upload(ctx context.Context, bucketName string, objectName string, r io.Reader, size int64) error
}
