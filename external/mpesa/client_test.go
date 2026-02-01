package mpesa

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-types/contact"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
	"github.com/Dorico-Dynamics/txova-go-types/money"
)

// mockPublisher is a mock implementation of EventPublisher for testing.
type mockPublisher struct {
	publishFunc     func(ctx context.Context, env *envelope.Envelope, partitionKey string) error
	publishedEvents []*envelope.Envelope
}

func (m *mockPublisher) Publish(ctx context.Context, env *envelope.Envelope, partitionKey string) error {
	m.publishedEvents = append(m.publishedEvents, env)
	if m.publishFunc != nil {
		return m.publishFunc(ctx, env, partitionKey)
	}
	return nil
}

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := &Config{
			APIKey:              "test-api-key",
			PublicKey:           "test-public-key",
			ServiceProviderCode: "171717",
			Origin:              "developer.mpesa.vm.co.mz",
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
		if client.baseURL != ProductionBaseURL {
			t.Errorf("expected production URL, got %s", client.baseURL)
		}
	})

	t.Run("uses sandbox URL when enabled", func(t *testing.T) {
		cfg := &Config{
			APIKey:              "test-api-key",
			PublicKey:           "test-public-key",
			ServiceProviderCode: "171717",
			Origin:              "developer.mpesa.vm.co.mz",
			Sandbox:             true,
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.baseURL != SandboxBaseURL {
			t.Errorf("expected sandbox URL, got %s", client.baseURL)
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
			PublicKey:           "test-public-key",
			ServiceProviderCode: "171717",
			Origin:              "developer.mpesa.vm.co.mz",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without public key", func(t *testing.T) {
		cfg := &Config{
			APIKey:              "test-api-key",
			ServiceProviderCode: "171717",
			Origin:              "developer.mpesa.vm.co.mz",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without service provider code", func(t *testing.T) {
		cfg := &Config{
			APIKey:    "test-api-key",
			PublicKey: "test-public-key",
			Origin:    "developer.mpesa.vm.co.mz",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error without origin", func(t *testing.T) {
		cfg := &Config{
			APIKey:              "test-api-key",
			PublicKey:           "test-public-key",
			ServiceProviderCode: "171717",
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for AllowUnencryptedAPIKey in production", func(t *testing.T) {
		cfg := &Config{
			APIKey:                 "test-api-key",
			PublicKey:              "test-public-key",
			ServiceProviderCode:    "171717",
			Origin:                 "developer.mpesa.vm.co.mz",
			Sandbox:                false, // Production mode.
			AllowUnencryptedAPIKey: true,  // Not allowed in production.
		}
		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "unencrypted API key is only allowed in sandbox mode" {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestInitiate(t *testing.T) {
	phone := contact.MustParsePhoneNumber("841234567")
	amount := money.FromMZN(100)

	t.Run("successful initiation", func(t *testing.T) {
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

			var req c2bRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}

			if req.InputServiceProviderCode != "171717" {
				t.Errorf("expected service provider code '171717', got %s", req.InputServiceProviderCode)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"output_TransactionID": "TXN123456",
				"output_ConversationID": "CONV123456",
				"output_ResponseCode": "INS-0",
				"output_ResponseDesc": "Request processed successfully",
				"output_ThirdPartyReference": "REF123"
			}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		result, err := client.Initiate(context.Background(), phone, amount, "REF123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TransactionID != "TXN123456" {
			t.Errorf("expected transaction ID 'TXN123456', got %s", result.TransactionID)
		}
		if !result.IsSuccess() {
			t.Error("expected success")
		}
	})

	t.Run("returns error for zero phone", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Initiate(context.Background(), contact.PhoneNumber{}, amount, "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for zero amount", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Initiate(context.Background(), phone, money.Zero(), "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for negative amount", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Initiate(context.Background(), phone, money.FromMZN(-100), "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty reference", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Initiate(context.Background(), phone, amount, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles API error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "Invalid request"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		_, err := client.Initiate(context.Background(), phone, amount, "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestQuery(t *testing.T) {
	t.Run("successful query", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"output_TransactionID": "TXN123456",
				"output_ConversationID": "CONV123456",
				"output_ResponseCode": "INS-0",
				"output_ResponseDesc": "Request processed successfully",
				"output_ThirdPartyReference": "REF123"
			}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		result, err := client.Query(context.Background(), "TXN123456", "REF123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TransactionID != "TXN123456" {
			t.Errorf("expected transaction ID 'TXN123456', got %s", result.TransactionID)
		}
		if !result.IsSuccess() {
			t.Error("expected success")
		}
	})

	t.Run("returns error for empty transaction ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Query(context.Background(), "", "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty third party ref", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Query(context.Background(), "TXN123", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRefund(t *testing.T) {
	amount := money.FromMZN(50)

	t.Run("successful refund", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", r.Method)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"output_TransactionID": "REV123456",
				"output_ConversationID": "CONV123456",
				"output_ResponseCode": "INS-0",
				"output_ResponseDesc": "Request processed successfully"
			}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		result, err := client.Refund(context.Background(), "TXN123456", amount, "REF123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TransactionID != "REV123456" {
			t.Errorf("expected transaction ID 'REV123456', got %s", result.TransactionID)
		}
		if !result.IsSuccess() {
			t.Error("expected success")
		}
	})

	t.Run("returns error for empty transaction ID", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Refund(context.Background(), "", amount, "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for zero amount", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Refund(context.Background(), "TXN123", money.Zero(), "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for empty third party ref", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		_, err := client.Refund(context.Background(), "TXN123", amount, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestParseCallback(t *testing.T) {
	t.Run("parses valid callback", func(t *testing.T) {
		body := []byte(`{
			"output_TransactionID": "TXN123456",
			"output_ConversationID": "CONV123456",
			"output_ThirdPartyReference": "REF123",
			"output_ResponseCode": "INS-0",
			"output_ResponseDesc": "Success"
		}`)

		callback, err := ParseCallback(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if callback.TransactionID != "TXN123456" {
			t.Errorf("expected transaction ID 'TXN123456', got %s", callback.TransactionID)
		}
		if callback.ResponseCode != "INS-0" {
			t.Errorf("expected response code 'INS-0', got %s", callback.ResponseCode)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := ParseCallback([]byte(`invalid json`))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestResponseCodeDescription(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"INS-0", "Request processed successfully"},
		{"INS-1", "Internal error"},
		{"INS-9", "Request timeout"},
		{"INS-10", "Duplicate transaction"},
		{"UNKNOWN", "Unknown error code: UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			desc := ResponseCodeDescription(tt.code)
			if desc != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, desc)
			}
		})
	}
}

func TestFormatPhoneForMPesa(t *testing.T) {
	phone := contact.MustParsePhoneNumber("841234567")
	formatted := formatPhoneForMPesa(phone)
	expected := "258841234567"
	if formatted != expected {
		t.Errorf("expected %s, got %s", expected, formatted)
	}
}

func TestGenerateTransactionRef(t *testing.T) {
	ref1 := generateTransactionRef()
	ref2 := generateTransactionRef()

	if ref1 == ref2 {
		t.Error("expected unique transaction references")
	}

	if len(ref1) < 10 {
		t.Errorf("expected longer reference, got %s", ref1)
	}

	if ref1[:3] != "TXV" {
		t.Errorf("expected prefix 'TXV', got %s", ref1[:3])
	}
}

func TestHandleCallback(t *testing.T) {
	t.Run("returns error for nil callback", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		err := client.HandleCallback(context.Background(), ids.PaymentID{}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "callback is required" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns nil when no producer configured", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		callback := &Callback{
			TransactionID: "TXN123",
			ResponseCode:  ResponseCodeSuccess,
		}
		err := client.HandleCallback(context.Background(), ids.PaymentID{}, callback)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles failed callback without producer", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:8080")
		callback := &Callback{
			TransactionID: "TXN123",
			ResponseCode:  "INS-1",
			ResponseDesc:  "Internal error",
		}
		err := client.HandleCallback(context.Background(), ids.PaymentID{}, callback)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestInitiateWithEvent(t *testing.T) {
	phone := contact.MustParsePhoneNumber("841234567")
	amount := money.FromMZN(100)

	t.Run("successful initiation without producer", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"output_TransactionID": "TXN123456",
				"output_ConversationID": "CONV123456",
				"output_ResponseCode": "INS-0",
				"output_ResponseDesc": "Request processed successfully",
				"output_ThirdPartyReference": "REF123"
			}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		paymentID := ids.MustNewPaymentID()
		rideID := ids.MustNewRideID()

		result, err := client.InitiateWithEvent(context.Background(), paymentID, rideID, phone, amount, "REF123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.TransactionID != "TXN123456" {
			t.Errorf("expected transaction ID 'TXN123456', got %s", result.TransactionID)
		}
	})

	t.Run("returns error on API failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "Invalid request"}`))
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		paymentID := ids.MustNewPaymentID()
		rideID := ids.MustNewRideID()

		_, err := client.InitiateWithEvent(context.Background(), paymentID, rideID, phone, amount, "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestInitiateResultIsSuccess(t *testing.T) {
	t.Run("returns true for success code", func(t *testing.T) {
		result := &InitiateResult{ResponseCode: ResponseCodeSuccess}
		if !result.IsSuccess() {
			t.Error("expected IsSuccess to return true")
		}
	})

	t.Run("returns false for failure code", func(t *testing.T) {
		result := &InitiateResult{ResponseCode: "INS-1"}
		if result.IsSuccess() {
			t.Error("expected IsSuccess to return false")
		}
	})
}

func TestTransactionStatusIsSuccess(t *testing.T) {
	t.Run("returns true for success code", func(t *testing.T) {
		status := &TransactionStatus{ResponseCode: ResponseCodeSuccess}
		if !status.IsSuccess() {
			t.Error("expected IsSuccess to return true")
		}
	})

	t.Run("returns false for failure code", func(t *testing.T) {
		status := &TransactionStatus{ResponseCode: "INS-1"}
		if status.IsSuccess() {
			t.Error("expected IsSuccess to return false")
		}
	})
}

func TestRefundResultIsSuccess(t *testing.T) {
	t.Run("returns true for success code", func(t *testing.T) {
		result := &RefundResult{ResponseCode: ResponseCodeSuccess}
		if !result.IsSuccess() {
			t.Error("expected IsSuccess to return true")
		}
	})

	t.Run("returns false for failure code", func(t *testing.T) {
		result := &RefundResult{ResponseCode: "INS-1"}
		if result.IsSuccess() {
			t.Error("expected IsSuccess to return false")
		}
	})
}

func TestSetHTTPClient(t *testing.T) {
	client := createTestClient(t, "http://localhost:8080")
	customClient := &http.Client{Timeout: 5 * time.Second}
	client.SetHTTPClient(customClient)
	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestSetBaseURL(t *testing.T) {
	client := createTestClient(t, "http://localhost:8080")
	newURL := "http://newurl.example.com"
	client.SetBaseURL(newURL)
	if client.baseURL != newURL {
		t.Errorf("expected base URL %s, got %s", newURL, client.baseURL)
	}
}

func TestEncryptAPIKey(t *testing.T) {
	// Test with valid PEM-encoded RSA public key.
	t.Run("encrypts with valid PEM key", func(t *testing.T) {
		cfg := &Config{
			APIKey:              "test-api-key",
			PublicKey:           testRSAPublicKeyPEM,
			ServiceProviderCode: "171717",
			Origin:              "developer.mpesa.vm.co.mz",
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		encrypted, err := client.encryptAPIKey()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Encrypted result should be base64 encoded and different from original.
		if encrypted == "test-api-key" {
			t.Error("expected encrypted key to be different from original")
		}
		if encrypted == "" {
			t.Error("expected non-empty encrypted key")
		}
	})

	t.Run("returns error with invalid key when AllowUnencryptedAPIKey is false", func(t *testing.T) {
		cfg := &Config{
			APIKey:                 "test-api-key",
			PublicKey:              "not-a-valid-key",
			ServiceProviderCode:    "171717",
			Origin:                 "developer.mpesa.vm.co.mz",
			AllowUnencryptedAPIKey: false,
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Should return error when encryption fails.
		_, err = client.encryptAPIKey()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrEncryptionFailed) {
			t.Errorf("expected ErrEncryptionFailed, got %v", err)
		}
	})

	t.Run("falls back to API key when AllowUnencryptedAPIKey is true", func(t *testing.T) {
		cfg := &Config{
			APIKey:                 "test-api-key",
			PublicKey:              "not-a-valid-key",
			ServiceProviderCode:    "171717",
			Origin:                 "developer.mpesa.vm.co.mz",
			Sandbox:                true, // Required for AllowUnencryptedAPIKey.
			AllowUnencryptedAPIKey: true,
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		// Should fall back to returning API key as-is.
		encrypted, err := client.encryptAPIKey()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if encrypted != "test-api-key" {
			t.Errorf("expected fallback to API key, got %s", encrypted)
		}
	})

	t.Run("encrypts with base64-encoded DER key", func(t *testing.T) {
		cfg := &Config{
			APIKey:              "test-api-key",
			PublicKey:           testRSAPublicKeyBase64,
			ServiceProviderCode: "171717",
			Origin:              "developer.mpesa.vm.co.mz",
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		encrypted, err := client.encryptAPIKey()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if encrypted == "test-api-key" {
			t.Error("expected encrypted key to be different from original")
		}
	})
}

func TestDoRequestErrors(t *testing.T) {
	t.Run("handles connection error", func(t *testing.T) {
		client := createTestClient(t, "http://localhost:59999") // port that doesn't exist
		phone := contact.MustParsePhoneNumber("841234567")
		amount := money.FromMZN(100)

		_, err := client.Initiate(context.Background(), phone, amount, "REF123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := createTestClient(t, server.URL)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		phone := contact.MustParsePhoneNumber("841234567")
		amount := money.FromMZN(100)

		_, err := client.Initiate(ctx, phone, amount, "REF123")
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

		client := createTestClient(t, server.URL)
		phone := contact.MustParsePhoneNumber("841234567")
		amount := money.FromMZN(100)

		_, err := client.Initiate(context.Background(), phone, amount, "REF123")
		if err == nil {
			t.Fatal("expected error for invalid JSON response")
		}
	})
}

func TestPublishPaymentInitiated(t *testing.T) {
	phone := contact.MustParsePhoneNumber("841234567")
	amount := money.FromMZN(100)
	logger := logging.New(logging.Config{Level: slog.LevelDebug})

	t.Run("publishes event with producer configured", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"output_TransactionID": "TXN123456",
				"output_ConversationID": "CONV123456",
				"output_ResponseCode": "INS-0",
				"output_ResponseDesc": "Request processed successfully",
				"output_ThirdPartyReference": "REF123"
			}`))
		}))
		defer server.Close()

		mock := &mockPublisher{}
		client := createTestClientWithProducer(t, server.URL, mock, logger)

		paymentID := ids.MustNewPaymentID()
		rideID := ids.MustNewRideID()

		_, err := client.InitiateWithEvent(context.Background(), paymentID, rideID, phone, amount, "REF123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(mock.publishedEvents) != 1 {
			t.Errorf("expected 1 published event, got %d", len(mock.publishedEvents))
		}
	})

	t.Run("handles publish error gracefully", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"output_TransactionID": "TXN123456",
				"output_ConversationID": "CONV123456",
				"output_ResponseCode": "INS-0",
				"output_ResponseDesc": "Request processed successfully",
				"output_ThirdPartyReference": "REF123"
			}`))
		}))
		defer server.Close()

		mock := &mockPublisher{
			publishFunc: func(_ context.Context, _ *envelope.Envelope, _ string) error {
				return errors.New("publish failed")
			},
		}
		client := createTestClientWithProducer(t, server.URL, mock, logger)

		paymentID := ids.MustNewPaymentID()
		rideID := ids.MustNewRideID()

		// Should still succeed even if publish fails (logged but not returned)
		result, err := client.InitiateWithEvent(context.Background(), paymentID, rideID, phone, amount, "REF123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
	})
}

func TestHandleCallbackWithProducer(t *testing.T) {
	logger := logging.New(logging.Config{Level: slog.LevelDebug})

	t.Run("publishes completed event on success callback", func(t *testing.T) {
		mock := &mockPublisher{}
		client := createTestClientWithProducer(t, "http://localhost:8080", mock, logger)

		paymentID := ids.MustNewPaymentID()
		callback := &Callback{
			TransactionID: "TXN123",
			ResponseCode:  ResponseCodeSuccess,
			ResponseDesc:  "Success",
		}

		err := client.HandleCallback(context.Background(), paymentID, callback)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(mock.publishedEvents) != 1 {
			t.Errorf("expected 1 published event, got %d", len(mock.publishedEvents))
		}
	})

	t.Run("publishes failed event on failure callback", func(t *testing.T) {
		mock := &mockPublisher{}
		client := createTestClientWithProducer(t, "http://localhost:8080", mock, logger)

		paymentID := ids.MustNewPaymentID()
		callback := &Callback{
			TransactionID: "TXN123",
			ResponseCode:  "INS-1",
			ResponseDesc:  "Internal error",
		}

		err := client.HandleCallback(context.Background(), paymentID, callback)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(mock.publishedEvents) != 1 {
			t.Errorf("expected 1 published event, got %d", len(mock.publishedEvents))
		}
	})

	t.Run("returns error when publish completed fails", func(t *testing.T) {
		mock := &mockPublisher{
			publishFunc: func(_ context.Context, _ *envelope.Envelope, _ string) error {
				return errors.New("publish failed")
			},
		}
		client := createTestClientWithProducer(t, "http://localhost:8080", mock, logger)

		paymentID := ids.MustNewPaymentID()
		callback := &Callback{
			TransactionID: "TXN123",
			ResponseCode:  ResponseCodeSuccess,
		}

		err := client.HandleCallback(context.Background(), paymentID, callback)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error when publish failed fails", func(t *testing.T) {
		mock := &mockPublisher{
			publishFunc: func(_ context.Context, _ *envelope.Envelope, _ string) error {
				return errors.New("publish failed")
			},
		}
		client := createTestClientWithProducer(t, "http://localhost:8080", mock, logger)

		paymentID := ids.MustNewPaymentID()
		callback := &Callback{
			TransactionID: "TXN123",
			ResponseCode:  "INS-1",
			ResponseDesc:  "Internal error",
		}

		err := client.HandleCallback(context.Background(), paymentID, callback)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestNewClientWithTimeout(t *testing.T) {
	t.Run("uses custom timeout", func(t *testing.T) {
		cfg := &Config{
			APIKey:              "test-api-key",
			PublicKey:           "test-public-key",
			ServiceProviderCode: "171717",
			Origin:              "developer.mpesa.vm.co.mz",
			Timeout:             30 * time.Second,
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.httpClient.Timeout != 30*time.Second {
			t.Errorf("expected timeout 30s, got %v", client.httpClient.Timeout)
		}
	})

	t.Run("uses default timeout when not specified", func(t *testing.T) {
		cfg := &Config{
			APIKey:              "test-api-key",
			PublicKey:           "test-public-key",
			ServiceProviderCode: "171717",
			Origin:              "developer.mpesa.vm.co.mz",
		}
		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.httpClient.Timeout != 60*time.Second {
			t.Errorf("expected default timeout 60s, got %v", client.httpClient.Timeout)
		}
	})
}

func createTestClient(t *testing.T, testServerURL string) *Client {
	t.Helper()

	cfg := &Config{
		APIKey:                 "test-api-key",
		PublicKey:              "test-public-key",
		ServiceProviderCode:    "171717",
		Origin:                 "developer.mpesa.vm.co.mz",
		Timeout:                10 * time.Second,
		Sandbox:                true, // Required for AllowUnencryptedAPIKey.
		AllowUnencryptedAPIKey: true, // Allow fallback for testing.
	}
	client, err := NewClient(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.SetBaseURL(testServerURL)
	return client
}

func createTestClientWithProducer(t *testing.T, testServerURL string, publisher EventPublisher, logger *logging.Logger) *Client {
	t.Helper()

	cfg := &Config{
		APIKey:                 "test-api-key",
		PublicKey:              "test-public-key",
		ServiceProviderCode:    "171717",
		Origin:                 "developer.mpesa.vm.co.mz",
		Timeout:                10 * time.Second,
		Producer:               publisher,
		Sandbox:                true, // Required for AllowUnencryptedAPIKey.
		AllowUnencryptedAPIKey: true, // Allow fallback for testing.
	}
	client, err := NewClient(cfg, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	client.SetBaseURL(testServerURL)
	return client
}

// testRSAPublicKeyPEM is a test RSA public key in PEM format.
const testRSAPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy0AHB7
PmOsIyJBpT0WmSqjQ/mM5fZHmqVx5b9Z0wnZz0aH8s7HH8fPzFrVsXr7O2DwZBhF+B2r2sHJr1Zm
F8sIxIKOvG0qe9Y3KTTX1y1b4THFZ3sWuE5scRhFbxex1Ybxn3DXvEfQn17a3D9a1vLV9J+JN3Ew
8e3YF5L3VZzJwJLXxJx3PXTz0wrD5PXD8r3JQ5WqV3Q3n7qJh1F7q5Z6a2B1E5e0qb/GhF/S3q0D
JmD4wHqL4fZKoA3R9U8v2e3H8BQE4sY7EqXkZfzNBAjMxEr7LCcL7qKB3qF3LFKq1H0rFAx0G+Qu
EqL4LwIDAQAB
-----END PUBLIC KEY-----`

// testRSAPublicKeyBase64 is the DER-encoded RSA public key in base64.
const testRSAPublicKeyBase64 = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy0AHB7PmOsIyJBpT0WmSqjQ/mM5fZHmqVx5b9Z0wnZz0aH8s7HH8fPzFrVsXr7O2DwZBhF+B2r2sHJr1ZmF8sIxIKOvG0qe9Y3KTTX1y1b4THFZ3sWuE5scRhFbxex1Ybxn3DXvEfQn17a3D9a1vLV9J+JN3Ew8e3YF5L3VZzJwJLXxJx3PXTz0wrD5PXD8r3JQ5WqV3Q3n7qJh1F7q5Z6a2B1E5e0qb/GhF/S3q0DJmD4wHqL4fZKoA3R9U8v2e3H8BQE4sY7EqXkZfzNBAjMxEr7LCcL7qKB3qF3LFKq1H0rFAx0G+QuEqL4LwIDAQAB"
