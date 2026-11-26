package storage

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/storage"
)

// GCSUploader uploads image bytes to a Google Cloud Storage bucket using Application Default Credentials.
type GCSUploader struct {
	client *storage.Client
	bucket string
}

// NewGCSUploader creates a new GCSUploader authenticated via Application Default Credentials.
func NewGCSUploader(ctx context.Context, bucket string) (*GCSUploader, error) {
	if bucket == "" {
		return nil, errors.New("GCS bucket name is required")
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSUploader{
		client: client,
		bucket: bucket,
	}, nil
}

// Upload writes data to the given object path in the configured bucket with the specified Content-Type.
// It does not set per-object ACLs so it works with uniform bucket-level access.
func (u *GCSUploader) Upload(ctx context.Context, objectName string, data []byte, contentType string) (string, error) {
	w := u.client.Bucket(u.bucket).Object(objectName).NewWriter(ctx)
	w.ContentType = contentType

	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return "", fmt.Errorf("failed to write object %q to GCS: %w", objectName, err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize object %q in GCS: %w", objectName, err)
	}

	return PublicURL(u.bucket, objectName), nil
}

// PublicURL returns the public HTTPS URL for an object in a Google Cloud Storage bucket.
func PublicURL(bucket, objectName string) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, objectName)
}

// Close closes the underlying GCS client.
func (u *GCSUploader) Close() error {
	if u.client == nil {
		return nil
	}
	return u.client.Close()
}
