package safety

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
	"github.com/Dorico-Dynamics/txova-go-types/rating"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := DefaultConfig("http://safety-service:8080")
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
	cfg := DefaultConfig("http://safety-service:8080")

	if cfg.BaseURL != "http://safety-service:8080" {
		t.Errorf("expected BaseURL 'http://safety-service:8080', got %s", cfg.BaseURL)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", cfg.Timeout)
	}
}

func TestGetUserRating(t *testing.T) {
	userID := ids.MustNewUserID()

	t.Run("successful get user rating", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/users/" + userID.String() + "/rating"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			ra := RatingAggregate{
				AverageRating: rating.MustNewRating(5),
				TotalRatings:  100,
				FiveStars:     80,
				FourStars:     15,
				ThreeStars:    3,
				TwoStars:      1,
				OneStar:       1,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ra)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		ra, err := client.GetUserRating(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ra.TotalRatings != 100 {
			t.Errorf("expected total ratings 100, got %d", ra.TotalRatings)
		}
		if ra.FiveStars != 80 {
			t.Errorf("expected five stars 80, got %d", ra.FiveStars)
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetUserRating(context.Background(), ids.UserID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"user not found"}}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.GetUserRating(context.Background(), userID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetDriverRating(t *testing.T) {
	driverID := ids.MustNewDriverID()

	t.Run("successful get driver rating", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/drivers/" + driverID.String() + "/rating"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			ra := RatingAggregate{
				AverageRating: rating.MustNewRating(5),
				TotalRatings:  500,
				FiveStars:     400,
				FourStars:     75,
				ThreeStars:    15,
				TwoStars:      7,
				OneStar:       3,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ra)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		ra, err := client.GetDriverRating(context.Background(), driverID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ra.TotalRatings != 500 {
			t.Errorf("expected total ratings 500, got %d", ra.TotalRatings)
		}
	})

	t.Run("returns error for zero driver ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetDriverRating(context.Background(), ids.DriverID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestReportIncident(t *testing.T) {
	rideID := ids.MustNewRideID()
	reporterID := ids.MustNewUserID()
	incidentID := ids.MustNewIncidentID()
	now := time.Now().Truncate(time.Second)

	t.Run("successful report incident", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/incidents" {
				t.Errorf("expected path /incidents, got %s", r.URL.Path)
			}

			var req IncidentReport
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req.RideID != rideID {
				t.Errorf("expected ride ID %s, got %s", rideID, req.RideID)
			}
			if req.Severity != enums.IncidentSeverityHigh {
				t.Errorf("expected severity 'high', got %s", req.Severity)
			}

			incident := Incident{
				ID:          incidentID,
				RideID:      req.RideID,
				ReporterID:  req.ReporterID,
				Severity:    req.Severity,
				Status:      enums.IncidentStatusReported,
				Type:        req.Type,
				Description: req.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(incident)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		report := &IncidentReport{
			RideID:      rideID,
			ReporterID:  reporterID,
			Severity:    enums.IncidentSeverityHigh,
			Type:        "harassment",
			Description: "Driver was verbally abusive",
		}
		incident, err := client.ReportIncident(context.Background(), report)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if incident.ID != incidentID {
			t.Errorf("expected incident ID %s, got %s", incidentID, incident.ID)
		}
		if incident.Status != enums.IncidentStatusReported {
			t.Errorf("expected status 'reported', got %s", incident.Status)
		}
	})

	t.Run("returns error for nil report", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.ReportIncident(context.Background(), nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for zero ride ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		report := &IncidentReport{
			RideID:      ids.RideID{},
			ReporterID:  reporterID,
			Severity:    enums.IncidentSeverityHigh,
			Description: "Test",
		}
		_, err := client.ReportIncident(context.Background(), report)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for zero reporter ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		report := &IncidentReport{
			RideID:      rideID,
			ReporterID:  ids.UserID{},
			Severity:    enums.IncidentSeverityHigh,
			Description: "Test",
		}
		_, err := client.ReportIncident(context.Background(), report)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for invalid severity", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		report := &IncidentReport{
			RideID:      rideID,
			ReporterID:  reporterID,
			Severity:    enums.IncidentSeverity("invalid"),
			Description: "Test",
		}
		_, err := client.ReportIncident(context.Background(), report)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty description", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		report := &IncidentReport{
			RideID:      rideID,
			ReporterID:  reporterID,
			Severity:    enums.IncidentSeverityHigh,
			Description: "",
		}
		_, err := client.ReportIncident(context.Background(), report)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetIncident(t *testing.T) {
	incidentID := ids.MustNewIncidentID()
	rideID := ids.MustNewRideID()
	reporterID := ids.MustNewUserID()
	now := time.Now().Truncate(time.Second)

	t.Run("successful get incident", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/incidents/" + incidentID.String()
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			incident := Incident{
				ID:          incidentID,
				RideID:      rideID,
				ReporterID:  reporterID,
				Severity:    enums.IncidentSeverityMedium,
				Status:      enums.IncidentStatusInvestigating,
				Type:        "unsafe_driving",
				Description: "Driver was speeding",
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(incident)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		incident, err := client.GetIncident(context.Background(), incidentID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if incident.ID != incidentID {
			t.Errorf("expected incident ID %s, got %s", incidentID, incident.ID)
		}
		if incident.Status != enums.IncidentStatusInvestigating {
			t.Errorf("expected status 'investigating', got %s", incident.Status)
		}
	})

	t.Run("returns error for zero incident ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetIncident(context.Background(), ids.IncidentID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestTriggerEmergency(t *testing.T) {
	rideID := ids.MustNewRideID()
	location := geo.MustNewLocation(-25.9692, 32.5732)

	t.Run("successful trigger emergency", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			expectedPath := "/rides/" + rideID.String() + "/emergency"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			var req TriggerEmergencyRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.TriggerEmergency(context.Background(), rideID, location)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero ride ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.TriggerEmergency(context.Background(), ids.RideID{}, location)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSafetyHealthCheck(t *testing.T) {
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
