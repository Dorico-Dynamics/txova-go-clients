package payment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
	"github.com/Dorico-Dynamics/txova-go-types/money"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := DefaultConfig("http://payment-service:8080")
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
	cfg := DefaultConfig("http://payment-service:8080")

	if cfg.BaseURL != "http://payment-service:8080" {
		t.Errorf("expected BaseURL 'http://payment-service:8080', got %s", cfg.BaseURL)
	}
	if cfg.Timeout != 15*time.Second {
		t.Errorf("expected Timeout 15s, got %v", cfg.Timeout)
	}
}

func TestGetPayment(t *testing.T) {
	paymentID := ids.MustNewPaymentID()
	rideID := ids.MustNewRideID()
	userID := ids.MustNewUserID()
	now := time.Now().Truncate(time.Second)

	t.Run("successful get payment", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/payments/" + paymentID.String()
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			p := Payment{
				ID:        paymentID,
				RideID:    rideID,
				UserID:    userID,
				Amount:    money.FromMZN(150),
				Method:    enums.PaymentMethodMPesa,
				Status:    enums.PaymentStatusCompleted,
				CreatedAt: now,
				UpdatedAt: now,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(p)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		p, err := client.GetPayment(context.Background(), paymentID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if p.ID != paymentID {
			t.Errorf("expected payment ID %s, got %s", paymentID, p.ID)
		}
		if p.Status != enums.PaymentStatusCompleted {
			t.Errorf("expected status 'completed', got %s", p.Status)
		}
	})

	t.Run("returns error for zero payment ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetPayment(context.Background(), ids.PaymentID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"payment not found"}}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.GetPayment(context.Background(), paymentID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetPaymentByRide(t *testing.T) {
	rideID := ids.MustNewRideID()
	paymentID := ids.MustNewPaymentID()

	t.Run("successful get payment by ride", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/payments/by-ride" {
				t.Errorf("expected path /payments/by-ride, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("ride_id") != rideID.String() {
				t.Errorf("expected ride_id query param %s, got %s", rideID.String(), r.URL.Query().Get("ride_id"))
			}

			p := Payment{
				ID:     paymentID,
				RideID: rideID,
				Method: enums.PaymentMethodMPesa,
				Status: enums.PaymentStatusCompleted,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(p)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		p, err := client.GetPaymentByRide(context.Background(), rideID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if p.RideID != rideID {
			t.Errorf("expected ride ID %s, got %s", rideID, p.RideID)
		}
	})

	t.Run("returns error for zero ride ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetPaymentByRide(context.Background(), ids.RideID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestInitiateRefund(t *testing.T) {
	paymentID := ids.MustNewPaymentID()
	amount := money.FromMZN(50)

	t.Run("successful initiate refund", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			expectedPath := "/payments/" + paymentID.String() + "/refund"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			var req InitiateRefundRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req.Reason != "customer requested" {
				t.Errorf("expected reason 'customer requested', got %s", req.Reason)
			}

			refund := Refund{
				ID:        "refund-123",
				PaymentID: paymentID,
				Amount:    amount,
				Reason:    req.Reason,
				Status:    enums.PaymentStatusPending,
				CreatedAt: time.Now(),
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(refund)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		refund, err := client.InitiateRefund(context.Background(), paymentID, amount, "customer requested")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if refund.PaymentID != paymentID {
			t.Errorf("expected payment ID %s, got %s", paymentID, refund.PaymentID)
		}
		if refund.Reason != "customer requested" {
			t.Errorf("expected reason 'customer requested', got %s", refund.Reason)
		}
	})

	t.Run("returns error for zero payment ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.InitiateRefund(context.Background(), ids.PaymentID{}, amount, "reason")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty reason", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.InitiateRefund(context.Background(), paymentID, amount, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetWalletBalance(t *testing.T) {
	userID := ids.MustNewUserID()

	t.Run("successful get wallet balance", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/wallets/" + userID.String() + "/balance"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			balance := WalletBalance{
				UserID:    userID,
				Balance:   money.FromMZN(1000),
				UpdatedAt: time.Now(),
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(balance)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		balance, err := client.GetWalletBalance(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if balance.UserID != userID {
			t.Errorf("expected user ID %s, got %s", userID, balance.UserID)
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetWalletBalance(context.Background(), ids.UserID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetPaymentStatus(t *testing.T) {
	paymentID := ids.MustNewPaymentID()

	t.Run("successful get payment status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/payments/" + paymentID.String() + "/status"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"completed"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		status, err := client.GetPaymentStatus(context.Background(), paymentID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if status != enums.PaymentStatusCompleted {
			t.Errorf("expected status 'completed', got %s", status)
		}
	})

	t.Run("returns error for zero payment ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetPaymentStatus(context.Background(), ids.PaymentID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestPaymentHealthCheck(t *testing.T) {
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
		Timeout: 15 * time.Second,
		Retry:   base.RetryConfig{MaxRetries: 0},
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}
	return client
}
