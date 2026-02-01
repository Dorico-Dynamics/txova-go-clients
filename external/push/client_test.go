package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := &Config{
			ProjectID:   "test-project",
			AccessToken: "test-token",
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

	t.Run("returns error without project ID", func(t *testing.T) {
		cfg := &Config{
			AccessToken: "test-token",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without access token", func(t *testing.T) {
		cfg := &Config{
			ProjectID: "test-project",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSendToDevice(t *testing.T) {
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

			var req fcmRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			if req.Message.Token == "" {
				t.Error("expected token in request")
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name": "projects/test/messages/12345"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		notification := &Notification{
			Title: "Test",
			Body:  "Test message",
		}
		result, err := client.SendToDevice(context.Background(), "test-token", notification, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Error("expected success")
		}
		if result.MessageID == "" {
			t.Error("expected message ID")
		}
	})

	t.Run("returns error for empty token", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.SendToDevice(context.Background(), "", nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles FCM error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"error": {
					"code": 400,
					"message": "Invalid registration token",
					"status": "INVALID_ARGUMENT"
				}
			}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		result, err := client.SendToDevice(context.Background(), "invalid-token", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Success {
			t.Error("expected failure")
		}
		if result.Error == "" {
			t.Error("expected error message")
		}
	})
}

func TestSendToTopic(t *testing.T) {
	t.Run("successful send", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req fcmRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			if req.Message.Topic != "news" {
				t.Errorf("expected topic 'news', got %s", req.Message.Topic)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name": "projects/test/messages/12345"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		notification := &Notification{
			Title: "Breaking News",
			Body:  "Something happened",
		}
		result, err := client.SendToTopic(context.Background(), "news", notification, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Error("expected success")
		}
	})

	t.Run("returns error for empty topic", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.SendToTopic(context.Background(), "", nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSendMessage(t *testing.T) {
	t.Run("successful send with custom message", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name": "projects/test/messages/12345"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		msg := Message{
			Token: "test-token",
			Notification: &Notification{
				Title: "Custom",
			},
			Data: map[string]string{
				"key": "value",
			},
			Android: &AndroidConfig{
				Priority: "high",
			},
		}
		result, err := client.SendMessage(context.Background(), msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Success {
			t.Error("expected success")
		}
	})

	t.Run("returns error for message without target", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		msg := Message{
			Notification: &Notification{Title: "Test"},
		}
		_, err := client.SendMessage(context.Background(), msg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSendMulticast(t *testing.T) {
	t.Run("successful multicast", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name": "projects/test/messages/12345"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		tokens := []string{"token1", "token2", "token3"}
		notification := &Notification{
			Title: "Broadcast",
			Body:  "Message to all",
		}

		result, err := client.SendMulticast(context.Background(), tokens, notification, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.SuccessCount != 3 {
			t.Errorf("expected 3 successes, got %d", result.SuccessCount)
		}
		if result.FailureCount != 0 {
			t.Errorf("expected 0 failures, got %d", result.FailureCount)
		}
		if len(result.Results) != 3 {
			t.Errorf("expected 3 results, got %d", len(result.Results))
		}
	})

	t.Run("returns error for empty tokens", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.SendMulticast(context.Background(), []string{}, nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles partial failures", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			callCount++
			if callCount == 2 {
				// Fail the second request
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"error": {"code": 400, "message": "Invalid token"}}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name": "projects/test/messages/12345"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		tokens := []string{"token1", "token2", "token3"}

		result, err := client.SendMulticast(context.Background(), tokens, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.SuccessCount != 2 {
			t.Errorf("expected 2 successes, got %d", result.SuccessCount)
		}
		if result.FailureCount != 1 {
			t.Errorf("expected 1 failure, got %d", result.FailureCount)
		}
	})
}

func TestNotificationHelpers(t *testing.T) {
	t.Run("NewRideNotification", func(t *testing.T) {
		n := NewRideNotification("123 Main St")
		if n.Title != "New Ride Request" {
			t.Errorf("unexpected title: %s", n.Title)
		}
	})

	t.Run("RideAcceptedNotification", func(t *testing.T) {
		n := RideAcceptedNotification("John", 5)
		if n.Title != "Ride Accepted" {
			t.Errorf("unexpected title: %s", n.Title)
		}
	})

	t.Run("DriverArrivedNotification", func(t *testing.T) {
		n := DriverArrivedNotification("John")
		if n.Title != "Driver Arrived" {
			t.Errorf("unexpected title: %s", n.Title)
		}
	})

	t.Run("RideCompletedNotification", func(t *testing.T) {
		n := RideCompletedNotification("MZN 150.00")
		if n.Title != "Ride Completed" {
			t.Errorf("unexpected title: %s", n.Title)
		}
	})

	t.Run("PaymentReceivedNotification", func(t *testing.T) {
		n := PaymentReceivedNotification("MZN 150.00")
		if n.Title != "Payment Received" {
			t.Errorf("unexpected title: %s", n.Title)
		}
	})
}

func TestSetAPIURL(t *testing.T) {
	cfg := &Config{
		ProjectID:   "test-project",
		AccessToken: "test-token",
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	original := client.apiURL

	client.SetAPIURL("https://test.example.com/%s/messages:send")
	if client.apiURL == original {
		t.Error("expected URL to be changed")
	}

	client.ResetAPIURL()
	if client.apiURL != original {
		t.Error("expected URL to be reset")
	}
}

func createTestClient(t *testing.T, testServerURL string) *Client {
	t.Helper()

	cfg := &Config{
		ProjectID:   "test-project",
		AccessToken: "test-token",
		Timeout:     10 * time.Second,
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	client.SetAPIURL(testServerURL + "/%s/messages:send")

	return client
}
