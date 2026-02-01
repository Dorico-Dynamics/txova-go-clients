package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-types/contact"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/ids"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := DefaultConfig("http://user-service:8080")
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
	cfg := DefaultConfig("http://user-service:8080")

	if cfg.BaseURL != "http://user-service:8080" {
		t.Errorf("expected BaseURL 'http://user-service:8080', got %s", cfg.BaseURL)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", cfg.Timeout)
	}
}

func TestGetUser(t *testing.T) {
	userID := ids.MustNewUserID()
	now := time.Now().Truncate(time.Second)

	t.Run("successful get user", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/users/" + userID.String()
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			user := User{
				ID:        userID,
				Phone:     contact.MustParsePhoneNumber("841234567"),
				FirstName: "John",
				LastName:  "Doe",
				Type:      enums.UserTypeRider,
				Status:    enums.UserStatusActive,
				CreatedAt: now,
				UpdatedAt: now,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(user)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		user, err := client.GetUser(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if user.ID != userID {
			t.Errorf("expected user ID %s, got %s", userID, user.ID)
		}
		if user.FirstName != "John" {
			t.Errorf("expected first name 'John', got %s", user.FirstName)
		}
		if user.Status != enums.UserStatusActive {
			t.Errorf("expected status 'active', got %s", user.Status)
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetUser(context.Background(), ids.UserID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"user not found"}}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.GetUser(context.Background(), userID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetUserByPhone(t *testing.T) {
	phone := contact.MustParsePhoneNumber("841234567")
	userID := ids.MustNewUserID()

	t.Run("successful get user by phone", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/users/by-phone" {
				t.Errorf("expected path /users/by-phone, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("phone") != phone.String() {
				t.Errorf("expected phone query param %s, got %s", phone.String(), r.URL.Query().Get("phone"))
			}

			user := User{
				ID:        userID,
				Phone:     phone,
				FirstName: "Jane",
				LastName:  "Doe",
				Type:      enums.UserTypeRider,
				Status:    enums.UserStatusActive,
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(user)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		user, err := client.GetUserByPhone(context.Background(), phone)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if user.Phone.String() != phone.String() {
			t.Errorf("expected phone %s, got %s", phone.String(), user.Phone.String())
		}
	})

	t.Run("returns error for zero phone", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetUserByPhone(context.Background(), contact.PhoneNumber{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestVerifyUser(t *testing.T) {
	userID := ids.MustNewUserID()

	t.Run("successful verify user", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			expectedPath := "/users/" + userID.String() + "/verify"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.VerifyUser(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.VerifyUser(context.Background(), ids.UserID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSuspendUser(t *testing.T) {
	userID := ids.MustNewUserID()

	t.Run("successful suspend user", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			expectedPath := "/users/" + userID.String() + "/suspend"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			var req SuspendUserRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if req.Reason != "fraud detected" {
				t.Errorf("expected reason 'fraud detected', got %s", req.Reason)
			}

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.SuspendUser(context.Background(), userID, "fraud detected")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SuspendUser(context.Background(), ids.UserID{}, "reason")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty reason", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SuspendUser(context.Background(), userID, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetUserStatus(t *testing.T) {
	userID := ids.MustNewUserID()

	t.Run("successful get user status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			expectedPath := "/users/" + userID.String() + "/status"
			if r.URL.Path != expectedPath {
				t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"active"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		status, err := client.GetUserStatus(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if status != enums.UserStatusActive {
			t.Errorf("expected status 'active', got %s", status)
		}
	})

	t.Run("returns error for zero user ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.GetUserStatus(context.Background(), ids.UserID{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestHealthCheck(t *testing.T) {
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
