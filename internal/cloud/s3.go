package cloud

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type GenericS3Provider struct {
	client *minio.Client
}

func NewGenericS3Provider(endpoint, accessKey, secret, region string) (*GenericS3Provider, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secret, ""),
		Region: region,
		Secure: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create s3 client: %w", err)
	}

	return &GenericS3Provider{
		client: client,
	}, nil
}

func (s *GenericS3Provider) Upload(ctx context.Context, bucketName string, objectName string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(
		ctx,
		bucketName,
		objectName,
		r,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to upload object to s3: %w", err)
	}

	return nil
}

func (s *GenericS3Provider) Download(ctx context.Context, bucketName string, objectName string) (io.ReadCloser, error) {
	reader, err := s.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download object from s3: %w", err)
	}

	return reader, nil
}
