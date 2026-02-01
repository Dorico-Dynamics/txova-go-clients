package pricing

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

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := DefaultConfig("http://pricing-service:8080")
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
	cfg := DefaultConfig("http://pricing-service:8080")

	if cfg.BaseURL != "http://pricing-service:8080" {
		t.Errorf("expected BaseURL 'http://pricing-service:8080', got %s", cfg.BaseURL)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected Timeout 5s, got %v", cfg.Timeout)
	}
}

func TestGetEstimate(t *testing.T) {
	pickup := geo.MustNewLocation(-25.9692, 32.5732)
	dropoff := geo.MustNewLocation(-25.9532, 32.5892)

	t.Run("successful get estimate", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/pricing/estimate" {
				t.Errorf("expected path /pricing/estimate, got %s", r.URL.Path)
			}

			var req GetEstimateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req.ServiceType != enums.ServiceTypeStandard {
				t.Errorf("expected service type 'standard', got %s", req.ServiceType)
			}

			estimate := FareEstimate{
				MinFare:         money.FromMZN(100),
				MaxFare:         money.FromMZN(200),
				EstimatedFare:   money.FromMZN(150),
				DistanceKM:      5.5,
				DurationMinutes: 15,
				SurgeMultiplier: 1.0,
				ServiceType:     enums.ServiceTypeStandard,
				BaseFare:        money.FromMZN(30),
				PerKMRate:       money.FromMZN(20),
				PerMinuteRate:   money.FromMZN(5),
				BookingFee:      money.FromMZN(10),
				ValidUntil:      time.Now().Add(5 * time.Minute),
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(estimate)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		estimate, err := client.GetEstimate(context.Background(), pickup, dropoff, enums.ServiceTypeStandard)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if estimate.ServiceType != enums.ServiceTypeStandard {
			t.Errorf("expected service type 'standard', got %s", estimate.ServiceType)
		}
		if estimate.DistanceKM != 5.5 {
			t.Errorf("expected distance 5.5, got %f", estimate.DistanceKM)
		}
	})

	t.Run("returns error for invalid service type", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetEstimate(context.Background(), pickup, dropoff, enums.ServiceType("invalid"))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetSurgeMultiplier(t *testing.T) {
	location := geo.MustNewLocation(-25.9692, 32.5732)

	t.Run("successful get surge multiplier", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/pricing/surge" {
				t.Errorf("expected path /pricing/surge, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("lat") == "" {
				t.Error("expected lat query param")
			}
			if r.URL.Query().Get("lon") == "" {
				t.Error("expected lon query param")
			}

			surge := SurgeInfo{
				Multiplier: 1.5,
				Location:   location,
				Reason:     "high demand",
				ExpiresAt:  time.Now().Add(10 * time.Minute),
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(surge)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		surge, err := client.GetSurgeMultiplier(context.Background(), location)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if surge.Multiplier != 1.5 {
			t.Errorf("expected multiplier 1.5, got %f", surge.Multiplier)
		}
		if surge.Reason != "high demand" {
			t.Errorf("expected reason 'high demand', got %s", surge.Reason)
		}
	})

	t.Run("returns error on server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"server error"}}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.GetSurgeMultiplier(context.Background(), location)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestValidateFare(t *testing.T) {
	rideID := ids.MustNewRideID()
	fare := money.FromMZN(150)

	t.Run("successful validate fare", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			expectedPath := "/pricing/validate/" + rideID.String()
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			var req ValidateFareRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			validation := FareValidation{
				Valid:             true,
				ExpectedFare:      money.FromMZN(150),
				ActualFare:        fare,
				Difference:        money.Zero(),
				DifferencePercent: 0.0,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(validation)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		validation, err := client.ValidateFare(context.Background(), rideID, fare)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !validation.Valid {
			t.Error("expected valid fare")
		}
		if validation.DifferencePercent != 0.0 {
			t.Errorf("expected 0%% difference, got %f%%", validation.DifferencePercent)
		}
	})

	t.Run("returns error for zero ride ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.ValidateFare(context.Background(), ids.RideID{}, fare)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetServiceTypes(t *testing.T) {
	t.Run("successful get service types", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/pricing/service-types" {
				t.Errorf("expected path /pricing/service-types, got %s", r.URL.Path)
			}

			response := struct {
				ServiceTypes []ServiceTypePricing `json:"service_types"`
			}{
				ServiceTypes: []ServiceTypePricing{
					{
						ServiceType:   enums.ServiceTypeStandard,
						DisplayName:   "Standard",
						Description:   "Affordable everyday rides",
						BaseFare:      money.FromMZN(30),
						PerKMRate:     money.FromMZN(20),
						PerMinuteRate: money.FromMZN(5),
						MinimumFare:   money.FromMZN(50),
						BookingFee:    money.FromMZN(10),
						Active:        true,
					},
					{
						ServiceType:   enums.ServiceTypePremium,
						DisplayName:   "Premium",
						Description:   "Top-rated drivers, premium vehicles",
						BaseFare:      money.FromMZN(50),
						PerKMRate:     money.FromMZN(35),
						PerMinuteRate: money.FromMZN(8),
						MinimumFare:   money.FromMZN(100),
						BookingFee:    money.FromMZN(15),
						Active:        true,
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		serviceTypes, err := client.GetServiceTypes(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(serviceTypes) != 2 {
			t.Errorf("expected 2 service types, got %d", len(serviceTypes))
		}
		if serviceTypes[0].ServiceType != enums.ServiceTypeStandard {
			t.Errorf("expected first service type 'standard', got %s", serviceTypes[0].ServiceType)
		}
		if serviceTypes[1].ServiceType != enums.ServiceTypePremium {
			t.Errorf("expected second service type 'premium', got %s", serviceTypes[1].ServiceType)
		}
	})

	t.Run("returns error on server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"server error"}}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.GetServiceTypes(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPricingHealthCheck(t *testing.T) {
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
		Timeout: 5 * time.Second,
		Retry:   base.RetryConfig{MaxRetries: 0},
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	return client
}
