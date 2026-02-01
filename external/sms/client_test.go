package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-types/contact"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := &Config{
			Username: "testuser",
			APIKey:   "testapikey",
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
		if client.baseURL != productionBaseURL {
			t.Errorf("expected production URL, got %s", client.baseURL)
		}
	})

	t.Run("uses sandbox URL when enabled", func(t *testing.T) {
		cfg := &Config{
			Username: "testuser",
			APIKey:   "testapikey",
			Sandbox:  true,
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.baseURL != sandboxBaseURL {
			t.Errorf("expected sandbox URL, got %s", client.baseURL)
		}
	})

	t.Run("returns error with nil config", func(t *testing.T) {
		_, err := NewClient(nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without username", func(t *testing.T) {
		cfg := &Config{
			APIKey: "testapikey",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without API key", func(t *testing.T) {
		cfg := &Config{
			Username: "testuser",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSend(t *testing.T) {
	phone := contact.MustParsePhoneNumber("841234567")

	t.Run("successful send", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/messaging" {
				t.Errorf("expected path /messaging, got %s", r.URL.Path)
			}
			if r.Header.Get("apiKey") == "" {
				t.Error("missing apiKey header")
			}
			if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
				t.Errorf("expected content-type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
			}

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"SMSMessageData": {
					"Message": "Sent to 1/1 Total Cost: MZN 0.50",
					"Recipients": [{
						"statusCode": 101,
						"number": "+258841234567",
						"status": "Success",
						"cost": "MZN 0.50",
						"messageId": "ATXid_123456789"
					}]
				}
			}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		result, err := client.Send(context.Background(), phone, "Hello, World!")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.MessageID != "ATXid_123456789" {
			t.Errorf("expected message ID 'ATXid_123456789', got %s", result.MessageID)
		}
		if result.Status != "Success" {
			t.Errorf("expected status 'Success', got %s", result.Status)
		}
	})

	t.Run("returns error for zero phone", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Send(context.Background(), contact.PhoneNumber{}, "Hello")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty message", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Send(context.Background(), phone, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on API failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "Internal Server Error"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.Send(context.Background(), phone, "Hello")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSendBulk(t *testing.T) {
	phones := []contact.PhoneNumber{
		contact.MustParsePhoneNumber("841234567"),
		contact.MustParsePhoneNumber("841234568"),
	}

	t.Run("successful bulk send", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			to := r.FormValue("to")
			if to == "" {
				t.Error("missing 'to' field")
			}

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"SMSMessageData": {
					"Message": "Sent to 2/2 Total Cost: MZN 1.00",
					"Recipients": [
						{
							"statusCode": 101,
							"number": "+258841234567",
							"status": "Success",
							"cost": "MZN 0.50",
							"messageId": "ATXid_1"
						},
						{
							"statusCode": 101,
							"number": "+258841234568",
							"status": "Success",
							"cost": "MZN 0.50",
							"messageId": "ATXid_2"
						}
					]
				}
			}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		results, err := client.SendBulk(context.Background(), phones, "Bulk message")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("returns error for empty phones", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.SendBulk(context.Background(), []contact.PhoneNumber{}, "Hello")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for invalid phone in list", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		invalidPhones := []contact.PhoneNumber{
			contact.MustParsePhoneNumber("841234567"),
			{}, // Zero value
		}
		_, err := client.SendBulk(context.Background(), invalidPhones, "Hello")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetBalance(t *testing.T) {
	t.Run("successful get balance", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.Header.Get("apiKey") == "" {
				t.Error("missing apiKey header")
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"UserData": {
					"balance": "MZN 150.00"
				}
			}`))
		}))
		defer server.Close()

		client := createTestClientWithBalanceURL(t, server.URL, server.URL)
		balance, err := client.GetBalance(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if balance.Value != "MZN 150.00" {
			t.Errorf("expected balance 'MZN 150.00', got %s", balance.Value)
		}
	})

	t.Run("returns error on network failure", func(t *testing.T) {
		cfg := &Config{
			Username: "testuser",
			APIKey:   "testapikey",
			Timeout:  100 * time.Millisecond,
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Use a transport that always fails
		client.httpClient = &http.Client{
			Timeout:   100 * time.Millisecond,
			Transport: &failingTransport{},
		}

		_, err = client.GetBalance(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error on API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "Unauthorized"}`))
		}))
		defer server.Close()

		client := createTestClientWithBalanceURL(t, server.URL, server.URL)
		_, err := client.GetBalance(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`not valid json`))
		}))
		defer server.Close()

		client := createTestClientWithBalanceURL(t, server.URL, server.URL)
		_, err := client.GetBalance(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Block forever
			select {}
		}))
		defer server.Close()

		client := createTestClientWithBalanceURL(t, server.URL, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := client.GetBalance(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// failingTransport is an http.RoundTripper that always returns an error.
type failingTransport struct{}

func (t *failingTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("network error")
}

func TestParseDeliveryCallback(t *testing.T) {
	t.Run("parses valid callback", func(t *testing.T) {
		body := []byte(`{
			"id": "ATXid_123",
			"status": "Success",
			"phoneNumber": "+258841234567",
			"networkCode": "63907",
			"failureReason": ""
		}`)

		callback, err := ParseDeliveryCallback(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if callback.ID != "ATXid_123" {
			t.Errorf("expected ID 'ATXid_123', got %s", callback.ID)
		}
		if callback.Status != "Success" {
			t.Errorf("expected status 'Success', got %s", callback.Status)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := ParseDeliveryCallback([]byte(`invalid json`))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestParseDeliveryCallbackForm(t *testing.T) {
	t.Run("parses form data", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/callback", http.NoBody)
		req.Form = map[string][]string{
			"id":            {"ATXid_123"},
			"status":        {"Success"},
			"phoneNumber":   {"+258841234567"},
			"networkCode":   {"63907"},
			"failureReason": {""},
			"retryCount":    {"0"},
		}

		callback, err := ParseDeliveryCallbackForm(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if callback.ID != "ATXid_123" {
			t.Errorf("expected ID 'ATXid_123', got %s", callback.ID)
		}
	})
}

func TestNewMockResponse(t *testing.T) {
	resp := NewMockResponse(http.StatusOK, `{"test": "data"}`)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestSetHTTPClient(t *testing.T) {
	cfg := &Config{
		Username: "testuser",
		APIKey:   "testapikey",
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	customClient := &http.Client{Timeout: 5 * time.Second}
	client.SetHTTPClient(customClient)

	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestSendSMSErrors(t *testing.T) {
	phone := contact.MustParsePhoneNumber("841234567")

	t.Run("handles network error", func(t *testing.T) {
		cfg := &Config{
			Username: "testuser",
			APIKey:   "testapikey",
			Timeout:  100 * time.Millisecond,
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}
		client.baseURL = "http://localhost:59999" // Invalid port

		_, err = client.Send(context.Background(), phone, "Test message")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`not valid json`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.Send(context.Background(), phone, "Test message")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Block forever
			select {}
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := client.Send(ctx, phone, "Test message")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when no recipients in response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{
				"SMSMessageData": {
					"Message": "Sent to 0/1",
					"Recipients": []
				}
			}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.Send(context.Background(), phone, "Test message")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestParseDeliveryCallbackFormError(t *testing.T) {
	t.Run("handles form parse error", func(t *testing.T) {
		// Create a request with an invalid body that will cause ParseForm to fail
		req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader("%invalid%"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		_, err := ParseDeliveryCallbackForm(req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func createTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	cfg := &Config{
		Username: "testuser",
		APIKey:   "testapikey",
		SenderID: "TXOVA",
		Timeout:  10 * time.Second,
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.baseURL = baseURL
	return client
}

func createTestClientWithBalanceURL(t *testing.T, baseURL, balanceURL string) *Client {
	t.Helper()
	cfg := &Config{
		Username: "testuser",
		APIKey:   "testapikey",
		SenderID: "TXOVA",
		Timeout:  10 * time.Second,
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.baseURL = baseURL
	client.balanceURL = balanceURL
	return client
}
