// Package storage provides a client for object storage (MinIO/S3 compatible).
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// Client is the storage client for MinIO/S3.
type Client struct {
	client *minio.Client
	bucket string
	logger *logging.Logger
}

// Config holds the configuration for the storage client.
type Config struct {
	// Endpoint is the storage endpoint (e.g., "play.min.io").
	Endpoint string

	// AccessKey is the access key ID.
	AccessKey string

	// SecretKey is the secret access key.
	SecretKey string

	// Bucket is the default bucket name.
	Bucket string

	// UseSSL enables TLS/SSL.
	UseSSL bool

	// Region is the bucket region (optional).
	Region string
}

// NewClient creates a new storage client.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if cfg.AccessKey == "" {
		return nil, fmt.Errorf("access key is required")
	}
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("secret key is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &Client{
		client: client,
		bucket: cfg.Bucket,
		logger: logger,
	}, nil
}

// ObjectInfo represents information about a stored object.
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ContentType  string    `json:"content_type"`
	ETag         string    `json:"etag"`
}

// Upload uploads an object to storage.
func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if reader == nil {
		return fmt.Errorf("reader is required")
	}

	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}

	_, err := c.client.PutObject(ctx, c.bucket, key, reader, size, opts)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	return nil
}

// Download downloads an object from storage.
func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	obj, err := c.client.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	return obj, nil
}

// GetPresignedURL generates a presigned URL for an object.
func (c *Client) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key is required")
	}
	if expiry <= 0 {
		expiry = 1 * time.Hour
	}

	reqParams := make(url.Values)
	presignedURL, err := c.client.PresignedGetObject(ctx, c.bucket, key, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

// Delete deletes an object from storage.
func (c *Client) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	err := c.client.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// Exists checks if an object exists in storage.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("key is required")
	}

	_, err := c.client.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// List lists objects with a given prefix.
func (c *Client) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	var objects []ObjectInfo
	for obj := range c.client.ListObjects(ctx, c.bucket, opts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", obj.Err)
		}
		objects = append(objects, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
		})
	}

	return objects, nil
}

// ProfilePhotoKey generates the storage key for a user's profile photo.
func ProfilePhotoKey(userID ids.UserID) string {
	return fmt.Sprintf("users/%s/profile.jpg", userID)
}

// DriverDocumentKey generates the storage key for a driver's document.
func DriverDocumentKey(driverID ids.DriverID, docType string) string {
	return fmt.Sprintf("drivers/%s/documents/%s.pdf", driverID, docType)
}

// VehiclePhotoKey generates the storage key for a vehicle photo.
func VehiclePhotoKey(vehicleID ids.VehicleID, index int) string {
	return fmt.Sprintf("vehicles/%s/photos/%d.jpg", vehicleID, index)
}

// RideReceiptKey generates the storage key for a ride receipt.
func RideReceiptKey(rideID ids.RideID) string {
	return fmt.Sprintf("rides/%s/receipt.pdf", rideID)
}

// GetClient returns the underlying MinIO client for advanced operations.
func (c *Client) GetClient() *minio.Client {
	return c.client
}

// GetBucket returns the configured bucket name.
func (c *Client) GetBucket() string {
	return c.bucket
}
