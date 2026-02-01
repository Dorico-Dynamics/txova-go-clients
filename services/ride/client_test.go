package ride

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/geo"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
	"github.com/Dorico-Dynamics/txova-go-types/money"
	"github.com/Dorico-Dynamics/txova-go-types/pagination"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := DefaultConfig("http://ride-service:8080")
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
	})

	t.Run("returns error with nil config", func(t *testing.T) {
		_, err := NewClient(nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error with empty base URL", func(t *testing.T) {
		cfg := &Config{BaseURL: ""}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("http://ride-service:8080")

	if cfg.BaseURL != "http://ride-service:8080" {
		t.Errorf("expected BaseURL 'http://ride-service:8080', got %s", cfg.BaseURL)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", cfg.Timeout)
	}
}

func TestGetRide(t *testing.T) {
	rideID := ids.MustNewRideID()
	riderID := ids.MustNewUserID()
	driverID := ids.MustNewDriverID()
	now := time.Now().Truncate(time.Second)

	t.Run("successful get ride", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/rides/" + rideID.String()
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			ride := Ride{
				ID:              rideID,
				RiderID:         riderID,
				DriverID:        &driverID,
				ServiceType:     enums.ServiceTypeStandard,
				Status:          enums.RideStatusInProgress,
				PickupLocation:  geo.MustNewLocation(-25.9692, 32.5732),
				DropoffLocation: geo.MustNewLocation(-25.9532, 32.5892),
				EstimatedFare:   money.FromMZN(150),
				DistanceKM:      5.5,
				DurationMinutes: 15,
				RequestedAt:     now,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ride)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		ride, err := client.GetRide(context.Background(), rideID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ride.ID != rideID {
			t.Errorf("expected ride ID %s, got %s", rideID, ride.ID)
		}
		if ride.Status != enums.RideStatusInProgress {
			t.Errorf("expected status 'in_progress', got %s", ride.Status)
		}
		if ride.DriverID == nil || *ride.DriverID != driverID {
			t.Errorf("expected driver ID %s", driverID)
		}
	})

	t.Run("returns error for zero ride ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetRide(context.Background(), ids.RideID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"ride not found"}}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.GetRide(context.Background(), rideID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetActiveRide(t *testing.T) {
	userID := ids.MustNewUserID()
	rideID := ids.MustNewRideID()

	t.Run("successful get active ride", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/rides/active" {
				t.Errorf("expected path /rides/active, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("user_id") != userID.String() {
				t.Errorf("expected user_id query param %s, got %s", userID.String(), r.URL.Query().Get("user_id"))
			}

			ride := Ride{
				ID:          rideID,
				RiderID:     userID,
				ServiceType: enums.ServiceTypeStandard,
				Status:      enums.RideStatusInProgress,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ride)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		ride, err := client.GetActiveRide(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ride.RiderID != userID {
			t.Errorf("expected rider ID %s, got %s", userID, ride.RiderID)
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetActiveRide(context.Background(), ids.UserID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetRideHistory(t *testing.T) {
	userID := ids.MustNewUserID()

	t.Run("successful get ride history", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/rides/history" {
				t.Errorf("expected path /rides/history, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("user_id") != userID.String() {
				t.Errorf("expected user_id query param %s", userID.String())
			}
			if r.URL.Query().Get("limit") != "10" {
				t.Errorf("expected limit 10, got %s", r.URL.Query().Get("limit"))
			}
			if r.URL.Query().Get("offset") != "0" {
				t.Errorf("expected offset 0, got %s", r.URL.Query().Get("offset"))
			}

			response := pagination.PageResponse[Ride]{
				Items: []Ride{{
					ID:          ids.MustNewRideID(),
					RiderID:     userID,
					ServiceType: enums.ServiceTypeStandard,
					Status:      enums.RideStatusCompleted,
				}},
				Total:   1,
				Limit:   10,
				Offset:  0,
				HasMore: false,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		page := pagination.PageRequest{Limit: 10, Offset: 0}
		result, err := client.GetRideHistory(context.Background(), userID, page)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("expected total count 1, got %d", result.Total)
		}
		if len(result.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(result.Items))
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetRideHistory(context.Background(), ids.UserID{}, pagination.PageRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestCancelRide(t *testing.T) {
	rideID := ids.MustNewRideID()

	t.Run("successful cancel ride", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			expectedPath := "/rides/" + rideID.String() + "/cancel"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			var req CancelRideRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req.Reason != enums.CancellationReasonRiderCancelled {
				t.Errorf("expected reason %s, got %s", enums.CancellationReasonRiderCancelled, req.Reason)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.CancelRide(context.Background(), rideID, enums.CancellationReasonRiderCancelled)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero ride ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.CancelRide(context.Background(), ids.RideID{}, enums.CancellationReasonRiderCancelled)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for invalid reason", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.CancelRide(context.Background(), rideID, enums.CancellationReason("invalid"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetRideStatus(t *testing.T) {
	rideID := ids.MustNewRideID()

	t.Run("successful get ride status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/rides/" + rideID.String() + "/status"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"completed"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		status, err := client.GetRideStatus(context.Background(), rideID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if status != enums.RideStatusCompleted {
			t.Errorf("expected status 'completed', got %s", status)
		}
	})

	t.Run("returns error for zero ride ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetRideStatus(context.Background(), ids.RideID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRideHealthCheck(t *testing.T) {
	t.Run("healthy service", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/health" {
				t.Errorf("expected path /health, got %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.HealthCheck(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unhealthy service", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.HealthCheck(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func createTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	cfg := &Config{
		BaseURL: baseURL,
		Timeout: 10 * time.Second,
		Retry:   base.RetryConfig{MaxRetries: 0},
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	return client
}
