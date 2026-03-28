package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"cloud.google.com/go/storage"
)

// GCSClient is a singleton for GCS operations
var gcsClient *storage.Client

// InitGCS initializes the GCS client
func InitGCS() error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %v", err)
	}
	gcsClient = client
	return nil
}

// UploadToGCS uploads a file to Google Cloud Storage
func UploadToGCS(ctx context.Context, bucketName, objectName string, data []byte) (string, error) {
	if gcsClient == nil {
		return "", fmt.Errorf("GCS client not initialized")
	}

	bucket := gcsClient.Bucket(bucketName)
	obj := bucket.Object(objectName)

	wctx, cancel := context.WithTimeout(ctx, 30*60)
	defer cancel()

	w := obj.NewWriter(wctx)
	if _, err := io.Copy(w, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("failed to write to GCS: %v", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close GCS writer: %v", err)
	}

	// Return the URI reference
	uri := fmt.Sprintf("gs://%s/%s", bucketName, objectName)
	log.Printf("Uploaded file to %s", uri)
	return uri, nil
}

// GetGCSBucket returns the configured GCS bucket name
func GetGCSBucket() string {
	bucket := os.Getenv("GCS_BUCKET")
	if bucket == "" {
		bucket = "axiom-bridge-files"
	}
	return bucket
}

// CloseGCS closes the GCS client
func CloseGCS() error {
	if gcsClient != nil {
		return gcsClient.Close()
	}
	return nil
}
