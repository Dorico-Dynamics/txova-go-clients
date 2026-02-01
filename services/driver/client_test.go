package driver

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
	"github.com/Dorico-Dynamics/txova-go-types/rating"
	"github.com/Dorico-Dynamics/txova-go-types/vehicle"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := DefaultConfig("http://driver-service:8080")
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
	cfg := DefaultConfig("http://driver-service:8080")

	if cfg.BaseURL != "http://driver-service:8080" {
		t.Errorf("expected BaseURL 'http://driver-service:8080', got %s", cfg.BaseURL)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", cfg.Timeout)
	}
}

func TestGetDriver(t *testing.T) {
	driverID := ids.MustNewDriverID()
	userID := ids.MustNewUserID()
	now := time.Now().Truncate(time.Second)

	t.Run("successful get driver", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/drivers/" + driverID.String()
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			driver := Driver{
				ID:                 driverID,
				UserID:             userID,
				Status:             enums.DriverStatusApproved,
				AvailabilityStatus: enums.AvailabilityStatusOnline,
				Rating:             rating.MustNewRating(5),
				TotalRides:         150,
				CreatedAt:          now,
				UpdatedAt:          now,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(driver)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		driver, err := client.GetDriver(context.Background(), driverID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if driver.ID != driverID {
			t.Errorf("expected driver ID %s, got %s", driverID, driver.ID)
		}
		if driver.Status != enums.DriverStatusApproved {
			t.Errorf("expected status 'approved', got %s", driver.Status)
		}
		if driver.TotalRides != 150 {
			t.Errorf("expected total rides 150, got %d", driver.TotalRides)
		}
	})

	t.Run("returns error for zero driver ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetDriver(context.Background(), ids.DriverID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"driver not found"}}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.GetDriver(context.Background(), driverID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetDriverByUserID(t *testing.T) {
	userID := ids.MustNewUserID()
	driverID := ids.MustNewDriverID()

	t.Run("successful get driver by user ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/drivers/by-user" {
				t.Errorf("expected path /drivers/by-user, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("user_id") != userID.String() {
				t.Errorf("expected user_id query param %s, got %s", userID.String(), r.URL.Query().Get("user_id"))
			}

			driver := Driver{
				ID:                 driverID,
				UserID:             userID,
				Status:             enums.DriverStatusApproved,
				AvailabilityStatus: enums.AvailabilityStatusOffline,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(driver)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		driver, err := client.GetDriverByUserID(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if driver.UserID != userID {
			t.Errorf("expected user ID %s, got %s", userID, driver.UserID)
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetDriverByUserID(context.Background(), ids.UserID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetActiveVehicle(t *testing.T) {
	driverID := ids.MustNewDriverID()
	vehicleID := ids.MustNewVehicleID()

	t.Run("successful get active vehicle", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/drivers/" + driverID.String() + "/vehicle"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			v := Vehicle{
				ID:           vehicleID,
				DriverID:     driverID,
				LicensePlate: vehicle.MustParseLicensePlate("AAA-123-MC"),
				Make:         "Toyota",
				Model:        "Corolla",
				Year:         2020,
				Color:        "White",
				Status:       enums.VehicleStatusActive,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(v)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		v, err := client.GetActiveVehicle(context.Background(), driverID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if v.ID != vehicleID {
			t.Errorf("expected vehicle ID %s, got %s", vehicleID, v.ID)
		}
		if v.Make != "Toyota" {
			t.Errorf("expected make 'Toyota', got %s", v.Make)
		}
	})

	t.Run("returns error for zero driver ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetActiveVehicle(context.Background(), ids.DriverID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRecordEarnings(t *testing.T) {
	driverID := ids.MustNewDriverID()
	rideID := ids.MustNewRideID()
	amount := money.FromMZN(500.00)

	t.Run("successful record earnings", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			expectedPath := "/drivers/" + driverID.String() + "/earnings"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			var req RecordEarningsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req.RideID != rideID {
				t.Errorf("expected ride ID %s, got %s", rideID, req.RideID)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.RecordEarnings(context.Background(), driverID, rideID, amount)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero driver ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.RecordEarnings(context.Background(), ids.DriverID{}, rideID, amount)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for zero ride ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.RecordEarnings(context.Background(), driverID, ids.RideID{}, amount)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetDriverStatus(t *testing.T) {
	driverID := ids.MustNewDriverID()

	t.Run("successful get driver status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/drivers/" + driverID.String() + "/status"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"online"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		status, err := client.GetDriverStatus(context.Background(), driverID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if status != enums.AvailabilityStatusOnline {
			t.Errorf("expected status 'online', got %s", status)
		}
	})

	t.Run("returns error for zero driver ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetDriverStatus(context.Background(), ids.DriverID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetNearbyDrivers(t *testing.T) {
	location := geo.MustNewLocation(-25.9692, 32.5732)
	driverID := ids.MustNewDriverID()

	t.Run("successful get nearby drivers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/drivers/nearby" {
				t.Errorf("expected path /drivers/nearby, got %s", r.URL.Path)
			}

			lat := r.URL.Query().Get("lat")
			lon := r.URL.Query().Get("lon")
			radiusKM := r.URL.Query().Get("radius_km")

			if lat == "" || lon == "" || radiusKM == "" {
				t.Error("missing query parameters")
			}

			response := struct {
				Drivers []*NearbyDriver `json:"drivers"`
			}{
				Drivers: []*NearbyDriver{
					{
						Driver: Driver{
							ID:                 driverID,
							Status:             enums.DriverStatusApproved,
							AvailabilityStatus: enums.AvailabilityStatusOnline,
						},
						Location:   location,
						DistanceKM: 1.5,
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		drivers, err := client.GetNearbyDrivers(context.Background(), location, 5.0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(drivers) != 1 {
			t.Fatalf("expected 1 driver, got %d", len(drivers))
		}
		if drivers[0].DistanceKM != 1.5 {
			t.Errorf("expected distance 1.5km, got %.2f", drivers[0].DistanceKM)
		}
	})
}

func TestUpdateLocation(t *testing.T) {
	driverID := ids.MustNewDriverID()
	location := geo.MustNewLocation(-25.9692, 32.5732)

	t.Run("successful update location", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", r.Method)
			}
			expectedPath := "/drivers/" + driverID.String() + "/location"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			var req UpdateLocationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.UpdateLocation(context.Background(), driverID, location)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero driver ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.UpdateLocation(context.Background(), ids.DriverID{}, location)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSetAvailability(t *testing.T) {
	driverID := ids.MustNewDriverID()

	t.Run("successful set availability", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", r.Method)
			}
			expectedPath := "/drivers/" + driverID.String() + "/availability"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.SetAvailability(context.Background(), driverID, enums.AvailabilityStatusOnline)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero driver ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SetAvailability(context.Background(), ids.DriverID{}, enums.AvailabilityStatusOnline)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SetAvailability(context.Background(), driverID, enums.AvailabilityStatus("invalid"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDriverHealthCheck(t *testing.T) {
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
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
