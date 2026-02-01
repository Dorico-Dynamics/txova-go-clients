package storage

import (
	"context"
	"testing"

	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

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

func TestUpload_Validation(t *testing.T) {
	client := createTestClient(t)

	t.Run("returns error for empty key", func(t *testing.T) {
		err := client.Upload(context.Background(), "", nil, 0, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for nil reader", func(t *testing.T) {
		err := client.Upload(context.Background(), "test/key", nil, 0, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDownload_Validation(t *testing.T) {
	client := createTestClient(t)

	t.Run("returns error for empty key", func(t *testing.T) {
		_, err := client.Download(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetPresignedURL_Validation(t *testing.T) {
	client := createTestClient(t)

	t.Run("returns error for empty key", func(t *testing.T) {
		_, err := client.GetPresignedURL(context.Background(), "", 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDelete_Validation(t *testing.T) {
	client := createTestClient(t)

	t.Run("returns error for empty key", func(t *testing.T) {
		err := client.Delete(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestExists_Validation(t *testing.T) {
	client := createTestClient(t)

	t.Run("returns error for empty key", func(t *testing.T) {
		_, err := client.Exists(context.Background(), "")
		if err == nil {
			t.Fatal("expected error, got nil")
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
