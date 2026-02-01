package email

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-types/contact"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := &Config{
			APIKey:    "test-api-key",
			FromEmail: "noreply@txova.co.mz",
			FromName:  "Txova",
		}
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

	t.Run("returns error without API key", func(t *testing.T) {
		cfg := &Config{
			FromEmail: "noreply@txova.co.mz",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without from email", func(t *testing.T) {
		cfg := &Config{
			APIKey: "test-api-key",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSend(t *testing.T) {
	email := contact.MustParseEmail("user@example.com")

	t.Run("successful send", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.Header.Get("Authorization") == "" {
				t.Error("missing Authorization header")
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected content-type application/json, got %s", r.Header.Get("Content-Type"))
			}

			var req sendGridRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			if len(req.Content) == 0 {
				t.Error("expected content in request")
			}
			if req.Content[0].Type != "text/plain" {
				t.Errorf("expected content type 'text/plain', got %s", req.Content[0].Type)
			}

			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.Send(context.Background(), email, "Test Subject", "Test Body")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero email", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.Send(context.Background(), contact.Email{}, "Subject", "Body")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty subject", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.Send(context.Background(), email, "", "Body")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty body", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.Send(context.Background(), email, "Subject", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on API failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors": [{"message": "Invalid email"}]}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.Send(context.Background(), email, "Subject", "Body")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSendHTML(t *testing.T) {
	email := contact.MustParseEmail("user@example.com")

	t.Run("successful send HTML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req sendGridRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			if len(req.Content) == 0 {
				t.Error("expected content in request")
			}
			if req.Content[0].Type != "text/html" {
				t.Errorf("expected content type 'text/html', got %s", req.Content[0].Type)
			}

			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.SendHTML(context.Background(), email, "Test Subject", "<h1>Hello</h1>")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero email", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SendHTML(context.Background(), contact.Email{}, "Subject", "<h1>Hi</h1>")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty subject", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SendHTML(context.Background(), email, "", "<h1>Hi</h1>")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty HTML body", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SendHTML(context.Background(), email, "Subject", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSendTemplate(t *testing.T) {
	email := contact.MustParseEmail("user@example.com")

	t.Run("successful send template", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req sendGridRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			if req.TemplateID != "d-abc123" {
				t.Errorf("expected template ID 'd-abc123', got %s", req.TemplateID)
			}

			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		data := map[string]string{
			"name":    "John",
			"company": "Txova",
		}
		err := client.SendTemplate(context.Background(), email, "d-abc123", data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for zero email", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SendTemplate(context.Background(), contact.Email{}, "d-abc123", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty template ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SendTemplate(context.Background(), email, "", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSendToMultiple(t *testing.T) {
	emails := []contact.Email{
		contact.MustParseEmail("user1@example.com"),
		contact.MustParseEmail("user2@example.com"),
	}

	t.Run("successful send to multiple", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req sendGridRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			if len(req.Personalizations) == 0 {
				t.Error("expected personalizations")
			}
			if len(req.Personalizations[0].To) != 2 {
				t.Errorf("expected 2 recipients, got %d", len(req.Personalizations[0].To))
			}

			w.WriteHeader(http.StatusAccepted)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		err := client.SendToMultiple(context.Background(), emails, "Bulk Subject", "Bulk Body")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for empty recipients", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SendToMultiple(context.Background(), []contact.Email{}, "Subject", "Body")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for invalid email in list", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		invalidEmails := []contact.Email{
			contact.MustParseEmail("user1@example.com"),
			{}, // Zero value
		}
		err := client.SendToMultiple(context.Background(), invalidEmails, "Subject", "Body")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty subject", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SendToMultiple(context.Background(), emails, "", "Body")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty body", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.SendToMultiple(context.Background(), emails, "Subject", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSetAPIURL(t *testing.T) {
	originalURL := apiURL

	SetAPIURL("https://test.example.com")
	if apiURL != "https://test.example.com" {
		t.Errorf("expected URL to be set")
	}

	ResetAPIURL()
	if apiURL != originalURL {
		t.Errorf("expected URL to be reset")
	}
}

func createTestClient(t *testing.T, testServerURL string) *Client {
	t.Helper()

	// Set the API URL to the test server
	SetAPIURL(testServerURL)
	t.Cleanup(ResetAPIURL)

	cfg := &Config{
		APIKey:    "test-api-key",
		FromEmail: "noreply@txova.co.mz",
		FromName:  "Txova",
		Timeout:   10 * time.Second,
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}
