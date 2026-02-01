package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// mockMinioClient is a mock implementation of minioClient for testing.
type mockMinioClient struct {
	putObjectFunc          func(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	getObjectFunc          func(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	presignedGetObjectFunc func(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error)
	removeObjectFunc       func(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	statObjectFunc         func(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error)
	listObjectsFunc        func(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
}

func (m *mockMinioClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, bucketName, objectName, reader, objectSize, opts)
	}
	return minio.UploadInfo{}, nil
}

func (m *mockMinioClient) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, bucketName, objectName, opts)
	}
	return nil, nil //nolint:nilnil // intentional for mock default behavior
}

func (m *mockMinioClient) PresignedGetObject(ctx context.Context, bucketName, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error) {
	if m.presignedGetObjectFunc != nil {
		return m.presignedGetObjectFunc(ctx, bucketName, objectName, expiry, reqParams)
	}
	return &url.URL{Scheme: "https", Host: "example.com", Path: "/test"}, nil
}

func (m *mockMinioClient) RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error {
	if m.removeObjectFunc != nil {
		return m.removeObjectFunc(ctx, bucketName, objectName, opts)
	}
	return nil
}

func (m *mockMinioClient) StatObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	if m.statObjectFunc != nil {
		return m.statObjectFunc(ctx, bucketName, objectName, opts)
	}
	return minio.ObjectInfo{}, nil
}

func (m *mockMinioClient) ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
	if m.listObjectsFunc != nil {
		return m.listObjectsFunc(ctx, bucketName, opts)
	}
	ch := make(chan minio.ObjectInfo)
	close(ch)
	return ch
}

func TestNewClient(t *testing.T) {
	t.Run("returns error with nil config", func(t *testing.T) {
		_, err := NewClient(nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without endpoint", func(t *testing.T) {
		cfg := &Config{
			AccessKey: "testkey",
			SecretKey: "testsecret",
			Bucket:    "testbucket",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without access key", func(t *testing.T) {
		cfg := &Config{
			Endpoint:  "play.min.io",
			SecretKey: "testsecret",
			Bucket:    "testbucket",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without secret key", func(t *testing.T) {
		cfg := &Config{
			Endpoint:  "play.min.io",
			AccessKey: "testkey",
			Bucket:    "testbucket",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without bucket", func(t *testing.T) {
		cfg := &Config{
			Endpoint:  "play.min.io",
			AccessKey: "testkey",
			SecretKey: "testsecret",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := &Config{
			Endpoint:  "play.min.io",
			AccessKey: "testkey",
			SecretKey: "testsecret",
			Bucket:    "testbucket",
			UseSSL:    true,
			Region:    "us-east-1",
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
		if client.GetBucket() != "testbucket" {
			t.Errorf("expected bucket 'testbucket', got %s", client.GetBucket())
		}
		if client.GetClient() == nil {
			t.Error("expected MinIO client, got nil")
		}
	})
}

func TestUpload(t *testing.T) {
	t.Run("returns error for empty key", func(t *testing.T) {
		client := createTestClient(t)
		err := client.Upload(context.Background(), "", nil, 0, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for nil reader", func(t *testing.T) {
		client := createTestClient(t)
		err := client.Upload(context.Background(), "test/key", nil, 0, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("successful upload", func(t *testing.T) {
		mock := &mockMinioClient{
			putObjectFunc: func(_ context.Context, bucketName, objectName string, _ io.Reader, _ int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
				if bucketName != "testbucket" {
					t.Errorf("expected bucket 'testbucket', got %s", bucketName)
				}
				if objectName != "test/key" {
					t.Errorf("expected key 'test/key', got %s", objectName)
				}
				if opts.ContentType != "image/jpeg" {
					t.Errorf("expected content type 'image/jpeg', got %s", opts.ContentType)
				}
				return minio.UploadInfo{Key: objectName}, nil
			},
		}
		client := createMockClient(mock)
		data := bytes.NewReader([]byte("test data"))
		err := client.Upload(context.Background(), "test/key", data, int64(data.Len()), "image/jpeg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error on upload failure", func(t *testing.T) {
		mock := &mockMinioClient{
			putObjectFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64, _ minio.PutObjectOptions) (minio.UploadInfo, error) {
				return minio.UploadInfo{}, errors.New("upload failed")
			},
		}
		client := createMockClient(mock)
		data := bytes.NewReader([]byte("test data"))
		err := client.Upload(context.Background(), "test/key", data, int64(data.Len()), "image/jpeg")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDownload(t *testing.T) {
	t.Run("returns error for empty key", func(t *testing.T) {
		client := createTestClient(t)
		_, err := client.Download(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("successful download", func(t *testing.T) {
		mock := &mockMinioClient{
			getObjectFunc: func(_ context.Context, bucketName, objectName string, _ minio.GetObjectOptions) (*minio.Object, error) {
				if bucketName != "testbucket" {
					t.Errorf("expected bucket 'testbucket', got %s", bucketName)
				}
				if objectName != "test/key" {
					t.Errorf("expected key 'test/key', got %s", objectName)
				}
				return nil, nil //nolint:nilnil // minio.Object can't be easily mocked
			},
		}
		client := createMockClient(mock)
		_, err := client.Download(context.Background(), "test/key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error on download failure", func(t *testing.T) {
		mock := &mockMinioClient{
			getObjectFunc: func(_ context.Context, _, _ string, _ minio.GetObjectOptions) (*minio.Object, error) {
				return nil, errors.New("download failed")
			},
		}
		client := createMockClient(mock)
		_, err := client.Download(context.Background(), "test/key")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetPresignedURL(t *testing.T) {
	t.Run("returns error for empty key", func(t *testing.T) {
		client := createTestClient(t)
		_, err := client.GetPresignedURL(context.Background(), "", 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("successful presigned URL generation", func(t *testing.T) {
		expectedURL := &url.URL{Scheme: "https", Host: "storage.example.com", Path: "/testbucket/test/key"}
		mock := &mockMinioClient{
			presignedGetObjectFunc: func(_ context.Context, bucketName, objectName string, expiry time.Duration, _ url.Values) (*url.URL, error) {
				if bucketName != "testbucket" {
					t.Errorf("expected bucket 'testbucket', got %s", bucketName)
				}
				if objectName != "test/key" {
					t.Errorf("expected key 'test/key', got %s", objectName)
				}
				if expiry != 2*time.Hour {
					t.Errorf("expected expiry 2h, got %v", expiry)
				}
				return expectedURL, nil
			},
		}
		client := createMockClient(mock)
		result, err := client.GetPresignedURL(context.Background(), "test/key", 2*time.Hour)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != expectedURL.String() {
			t.Errorf("expected %s, got %s", expectedURL.String(), result)
		}
	})

	t.Run("uses default expiry when zero", func(t *testing.T) {
		mock := &mockMinioClient{
			presignedGetObjectFunc: func(_ context.Context, _, _ string, expiry time.Duration, _ url.Values) (*url.URL, error) {
				if expiry != 1*time.Hour {
					t.Errorf("expected default expiry 1h, got %v", expiry)
				}
				return &url.URL{Scheme: "https", Host: "example.com"}, nil
			},
		}
		client := createMockClient(mock)
		_, err := client.GetPresignedURL(context.Background(), "test/key", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error on presigned URL failure", func(t *testing.T) {
		mock := &mockMinioClient{
			presignedGetObjectFunc: func(_ context.Context, _, _ string, _ time.Duration, _ url.Values) (*url.URL, error) {
				return nil, errors.New("presigned URL failed")
			},
		}
		client := createMockClient(mock)
		_, err := client.GetPresignedURL(context.Background(), "test/key", time.Hour)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("returns error for empty key", func(t *testing.T) {
		client := createTestClient(t)
		err := client.Delete(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("successful delete", func(t *testing.T) {
		mock := &mockMinioClient{
			removeObjectFunc: func(_ context.Context, bucketName, objectName string, _ minio.RemoveObjectOptions) error {
				if bucketName != "testbucket" {
					t.Errorf("expected bucket 'testbucket', got %s", bucketName)
				}
				if objectName != "test/key" {
					t.Errorf("expected key 'test/key', got %s", objectName)
				}
				return nil
			},
		}
		client := createMockClient(mock)
		err := client.Delete(context.Background(), "test/key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error on delete failure", func(t *testing.T) {
		mock := &mockMinioClient{
			removeObjectFunc: func(_ context.Context, _, _ string, _ minio.RemoveObjectOptions) error {
				return errors.New("delete failed")
			},
		}
		client := createMockClient(mock)
		err := client.Delete(context.Background(), "test/key")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestExists(t *testing.T) {
	t.Run("returns error for empty key", func(t *testing.T) {
		client := createTestClient(t)
		_, err := client.Exists(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns true when object exists", func(t *testing.T) {
		mock := &mockMinioClient{
			statObjectFunc: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
				return minio.ObjectInfo{Key: "test/key", Size: 100}, nil
			},
		}
		client := createMockClient(mock)
		exists, err := client.Exists(context.Background(), "test/key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("expected exists to be true")
		}
	})

	t.Run("returns false when object does not exist", func(t *testing.T) {
		mock := &mockMinioClient{
			statObjectFunc: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
				return minio.ObjectInfo{}, minio.ErrorResponse{Code: "NoSuchKey"}
			},
		}
		client := createMockClient(mock)
		exists, err := client.Exists(context.Background(), "test/key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists {
			t.Error("expected exists to be false")
		}
	})

	t.Run("returns error on stat failure", func(t *testing.T) {
		mock := &mockMinioClient{
			statObjectFunc: func(_ context.Context, _, _ string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
				return minio.ObjectInfo{}, minio.ErrorResponse{Code: "AccessDenied"}
			},
		}
		client := createMockClient(mock)
		_, err := client.Exists(context.Background(), "test/key")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestList(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		mock := &mockMinioClient{
			listObjectsFunc: func(_ context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				if bucketName != "testbucket" {
					t.Errorf("expected bucket 'testbucket', got %s", bucketName)
				}
				if opts.Prefix != "users/" {
					t.Errorf("expected prefix 'users/', got %s", opts.Prefix)
				}
				ch := make(chan minio.ObjectInfo, 2)
				ch <- minio.ObjectInfo{Key: "users/1/profile.jpg", Size: 1000}
				ch <- minio.ObjectInfo{Key: "users/2/profile.jpg", Size: 2000}
				close(ch)
				return ch
			},
		}
		client := createMockClient(mock)
		objects, err := client.List(context.Background(), "users/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(objects) != 2 {
			t.Errorf("expected 2 objects, got %d", len(objects))
		}
		if objects[0].Key != "users/1/profile.jpg" {
			t.Errorf("expected key 'users/1/profile.jpg', got %s", objects[0].Key)
		}
	})

	t.Run("returns error on list failure", func(t *testing.T) {
		mock := &mockMinioClient{
			listObjectsFunc: func(_ context.Context, _ string, _ minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				ch := make(chan minio.ObjectInfo, 1)
				ch <- minio.ObjectInfo{Err: errors.New("list failed")}
				close(ch)
				return ch
			},
		}
		client := createMockClient(mock)
		_, err := client.List(context.Background(), "users/")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns empty list", func(t *testing.T) {
		mock := &mockMinioClient{
			listObjectsFunc: func(_ context.Context, _ string, _ minio.ListObjectsOptions) <-chan minio.ObjectInfo {
				ch := make(chan minio.ObjectInfo)
				close(ch)
				return ch
			},
		}
		client := createMockClient(mock)
		objects, err := client.List(context.Background(), "empty/")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(objects) != 0 {
			t.Errorf("expected 0 objects, got %d", len(objects))
		}
	})
}

func TestGetClient(t *testing.T) {
	t.Run("returns nil for mock client", func(t *testing.T) {
		client := createMockClient(&mockMinioClient{})
		if client.GetClient() != nil {
			t.Error("expected nil for mock client")
		}
	})

	t.Run("returns minio client for real client", func(t *testing.T) {
		client := createTestClient(t)
		if client.GetClient() == nil {
			t.Error("expected non-nil minio client")
		}
	})
}

func TestKeyGenerators(t *testing.T) {
	userID := ids.MustNewUserID()
	driverID := ids.MustNewDriverID()
	vehicleID := ids.MustNewVehicleID()
	rideID := ids.MustNewRideID()

	t.Run("ProfilePhotoKey", func(t *testing.T) {
		key := ProfilePhotoKey(userID)
		expected := "users/" + userID.String() + "/profile.jpg"
		if key != expected {
			t.Errorf("expected %s, got %s", expected, key)
		}
	})

	t.Run("DriverDocumentKey", func(t *testing.T) {
		key := DriverDocumentKey(driverID, "license")
		expected := "drivers/" + driverID.String() + "/documents/license.pdf"
		if key != expected {
			t.Errorf("expected %s, got %s", expected, key)
		}
	})

	t.Run("VehiclePhotoKey", func(t *testing.T) {
		key := VehiclePhotoKey(vehicleID, 1)
		expected := "vehicles/" + vehicleID.String() + "/photos/1.jpg"
		if key != expected {
			t.Errorf("expected %s, got %s", expected, key)
		}
	})

	t.Run("RideReceiptKey", func(t *testing.T) {
		key := RideReceiptKey(rideID)
		expected := "rides/" + rideID.String() + "/receipt.pdf"
		if key != expected {
			t.Errorf("expected %s, got %s", expected, key)
		}
	})
}

func createTestClient(t *testing.T) *Client {
	t.Helper()
	cfg := &Config{
		Endpoint:  "play.min.io",
		AccessKey: "testkey",
		SecretKey: "testsecret",
		Bucket:    "testbucket",
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	return client
}

func createMockClient(mock *mockMinioClient) *Client {
	client := &Client{
		client: mock,
		bucket: "testbucket",
		logger: nil,
	}
	return client
}
