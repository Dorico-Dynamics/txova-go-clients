// Package mpesa provides a client for M-Pesa mobile money payments in Mozambique.
package mpesa

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-kafka/events"
	"github.com/Dorico-Dynamics/txova-go-types/contact"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
	"github.com/Dorico-Dynamics/txova-go-types/money"
)

// M-Pesa API environments.
const (
	SandboxBaseURL    = "https://api.sandbox.vm.co.mz"
	ProductionBaseURL = "https://api.vm.co.mz"
)

// EventPublisher defines the interface for publishing Kafka events.
// This allows for mocking in tests.
type EventPublisher interface {
	Publish(ctx context.Context, env *envelope.Envelope, partitionKey string) error
}

// Client is the M-Pesa client.
type Client struct {
	httpClient             *http.Client
	baseURL                string
	apiKey                 string
	publicKey              string
	serviceProviderCode    string
	origin                 string
	allowUnencryptedAPIKey bool
	logger                 *logging.Logger
	producer               EventPublisher
}

// Config holds the configuration for the M-Pesa client.
type Config struct {
	// APIKey is the M-Pesa API key (required).
	APIKey string

	// PublicKey is the M-Pesa public key for encryption (required).
	PublicKey string

	// ServiceProviderCode is the M-Pesa service provider code (required).
	ServiceProviderCode string

	// Origin is the registered origin for M-Pesa API requests (required).
	// This should be the domain registered with M-Pesa (e.g., "developer.mpesa.vm.co.mz").
	Origin string

	// Sandbox enables sandbox mode for testing.
	Sandbox bool

	// Timeout is the request timeout (default: 60s for payment operations).
	Timeout time.Duration

	// Producer is the Kafka producer for event publishing (optional).
	// If nil, events will not be published.
	Producer EventPublisher

	// AllowUnencryptedAPIKey allows returning the API key unencrypted when RSA
	// encryption fails. This should only be enabled for testing with mock servers.
	// In production, this MUST be false (default).
	AllowUnencryptedAPIKey bool
}

// NewClient creates a new M-Pesa client.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if cfg.PublicKey == "" {
		return nil, fmt.Errorf("public key is required")
	}
	if cfg.ServiceProviderCode == "" {
		return nil, fmt.Errorf("service provider code is required")
	}
	if cfg.Origin == "" {
		return nil, fmt.Errorf("origin is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	baseURL := ProductionBaseURL
	if cfg.Sandbox {
		baseURL = SandboxBaseURL
	}

	return &Client{
		httpClient:             &http.Client{Timeout: timeout},
		baseURL:                baseURL,
		apiKey:                 cfg.APIKey,
		publicKey:              cfg.PublicKey,
		serviceProviderCode:    cfg.ServiceProviderCode,
		origin:                 cfg.Origin,
		allowUnencryptedAPIKey: cfg.AllowUnencryptedAPIKey,
		logger:                 logger,
		producer:               cfg.Producer,
	}, nil
}

// serviceName is used as the source in Kafka envelopes.
const serviceName = "mpesa-client"

// InitiateResult represents the result of a payment initiation.
type InitiateResult struct {
	// TransactionID is the M-Pesa transaction ID.
	TransactionID string `json:"output_TransactionID"`

	// ConversationID is the conversation ID for tracking.
	ConversationID string `json:"output_ConversationID"`

	// ResponseDesc is the response description.
	ResponseDesc string `json:"output_ResponseDesc"`

	// ResponseCode is the response code.
	ResponseCode string `json:"output_ResponseCode"`

	// ThirdPartyReference is the third party reference.
	ThirdPartyReference string `json:"output_ThirdPartyReference"`
}

// TransactionStatus represents the status of a transaction.
type TransactionStatus struct {
	// TransactionID is the M-Pesa transaction ID.
	TransactionID string `json:"output_TransactionID"`

	// ConversationID is the conversation ID.
	ConversationID string `json:"output_ConversationID"`

	// ResponseCode is the response code.
	ResponseCode string `json:"output_ResponseCode"`

	// ResponseDesc is the response description.
	ResponseDesc string `json:"output_ResponseDesc"`

	// ThirdPartyReference is the third party reference.
	ThirdPartyReference string `json:"output_ThirdPartyReference"`
}

// RefundResult represents the result of a refund.
type RefundResult struct {
	// TransactionID is the M-Pesa transaction ID for the refund.
	TransactionID string `json:"output_TransactionID"`

	// ConversationID is the conversation ID.
	ConversationID string `json:"output_ConversationID"`

	// ResponseCode is the response code.
	ResponseCode string `json:"output_ResponseCode"`

	// ResponseDesc is the response description.
	ResponseDesc string `json:"output_ResponseDesc"`
}

// c2bRequest is the request body for C2B (Customer to Business) transactions.
type c2bRequest struct {
	InputTransactionReference string `json:"input_TransactionReference"`
	InputCustomerMSISDN       string `json:"input_CustomerMSISDN"`
	InputAmount               string `json:"input_Amount"`
	InputThirdPartyReference  string `json:"input_ThirdPartyReference"`
	InputServiceProviderCode  string `json:"input_ServiceProviderCode"`
}

// Initiate initiates a C2B (Customer to Business) payment.
// This sends a payment request to the customer's M-Pesa wallet.
func (c *Client) Initiate(ctx context.Context, phone contact.PhoneNumber, amount money.Money, reference string) (*InitiateResult, error) {
	if phone.IsZero() {
		return nil, fmt.Errorf("phone number is required")
	}
	if amount.IsZero() || amount.IsNegative() {
		return nil, fmt.Errorf("valid positive amount is required")
	}
	if reference == "" {
		return nil, fmt.Errorf("reference is required")
	}

	// Generate unique transaction reference
	transactionRef := generateTransactionRef()

	req := c2bRequest{
		InputTransactionReference: transactionRef,
		InputCustomerMSISDN:       formatPhoneForMPesa(phone),
		InputAmount:               fmt.Sprintf("%.2f", amount.MZN()),
		InputThirdPartyReference:  reference,
		InputServiceProviderCode:  c.serviceProviderCode,
	}

	var result InitiateResult
	err := c.doRequest(ctx, http.MethodPost, "/ipg/v1x/c2bPayment/singleStage/", req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// InitiateWithEvent initiates a payment and publishes a PaymentInitiated event to Kafka.
// This is the preferred method when event publishing is required.
func (c *Client) InitiateWithEvent(ctx context.Context, paymentID ids.PaymentID, rideID ids.RideID, phone contact.PhoneNumber, amount money.Money, reference string) (*InitiateResult, error) {
	result, err := c.Initiate(ctx, phone, amount, reference)
	if err != nil {
		return nil, err
	}

	// Publish PaymentInitiated event if producer is configured
	c.publishPaymentInitiated(ctx, paymentID, rideID, amount)

	return result, nil
}

// publishPaymentInitiated publishes a PaymentInitiated event to Kafka.
func (c *Client) publishPaymentInitiated(ctx context.Context, paymentID ids.PaymentID, rideID ids.RideID, amount money.Money) {
	if c.producer == nil {
		return
	}

	payload := events.PaymentInitiated{
		PaymentID: paymentID,
		RideID:    rideID,
		Amount:    amount.Centavos(),
		Method:    enums.PaymentMethodMPesa,
	}

	env, err := envelope.NewWithContext(ctx, &envelope.Config{
		Type:    string(events.EventTypePaymentInitiated),
		Version: events.VersionLatest,
		Source:  serviceName,
		Payload: payload,
	})
	if err != nil {
		c.logger.WarnContext(ctx, "failed to create payment initiated envelope",
			"error", err.Error(),
			"payment_id", paymentID.String(),
		)
		return
	}

	if err := c.producer.Publish(ctx, env, paymentID.String()); err != nil {
		c.logger.WarnContext(ctx, "failed to publish payment initiated event",
			"error", err.Error(),
			"payment_id", paymentID.String(),
		)
	}
}

// queryRequest is the request body for transaction queries.
type queryRequest struct {
	InputQueryReference      string `json:"input_QueryReference"`
	InputServiceProviderCode string `json:"input_ServiceProviderCode"`
	InputThirdPartyReference string `json:"input_ThirdPartyReference"`
}

// Query queries the status of a transaction.
func (c *Client) Query(ctx context.Context, transactionID, thirdPartyRef string) (*TransactionStatus, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("transaction ID is required")
	}
	if thirdPartyRef == "" {
		return nil, fmt.Errorf("third party reference is required")
	}

	req := queryRequest{
		InputQueryReference:      transactionID,
		InputServiceProviderCode: c.serviceProviderCode,
		InputThirdPartyReference: thirdPartyRef,
	}

	var result TransactionStatus
	err := c.doRequest(ctx, http.MethodGet, "/ipg/v1x/queryTransactionStatus/", req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// reversalRequest is the request body for reversals/refunds.
type reversalRequest struct {
	InputTransactionID       string `json:"input_TransactionID"`
	InputSecurityCredential  string `json:"input_SecurityCredential"`
	InputInitiatorIdentifier string `json:"input_InitiatorIdentifier"`
	InputThirdPartyReference string `json:"input_ThirdPartyReference"`
	InputServiceProviderCode string `json:"input_ServiceProviderCode"`
	InputReversalAmount      string `json:"input_ReversalAmount"`
}

// Refund initiates a refund/reversal for a transaction.
func (c *Client) Refund(ctx context.Context, transactionID string, amount money.Money, thirdPartyRef string) (*RefundResult, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("transaction ID is required")
	}
	if amount.IsZero() || amount.IsNegative() {
		return nil, fmt.Errorf("valid positive amount is required")
	}
	if thirdPartyRef == "" {
		return nil, fmt.Errorf("third party reference is required")
	}

	// Encrypt the API key for security credential
	securityCredential, err := c.encryptAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt security credential: %w", err)
	}

	req := reversalRequest{
		InputTransactionID:       transactionID,
		InputSecurityCredential:  securityCredential,
		InputInitiatorIdentifier: c.serviceProviderCode,
		InputThirdPartyReference: thirdPartyRef,
		InputServiceProviderCode: c.serviceProviderCode,
		InputReversalAmount:      fmt.Sprintf("%.2f", amount.MZN()),
	}

	var result RefundResult
	err = c.doRequest(ctx, http.MethodPut, "/ipg/v1x/reversal/", req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// doRequest performs an HTTP request to the M-Pesa API.
func (c *Client) doRequest(ctx context.Context, method, path string, body, result any) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Generate bearer token by encrypting API key
	bearerToken, err := c.encryptAPIKey()
	if err != nil {
		return fmt.Errorf("failed to generate bearer token: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Origin", c.origin)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("M-Pesa API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// encryptAPIKey encrypts the API key using the public key.
func (c *Client) encryptAPIKey() (string, error) {
	// Decode PEM public key
	block, _ := pem.Decode([]byte(c.publicKey))
	if block == nil {
		// Try without PEM wrapper
		return c.encryptWithRawKey()
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("not an RSA public key")
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, []byte(c.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// ErrEncryptionFailed is returned when API key encryption fails.
var ErrEncryptionFailed = fmt.Errorf("failed to encrypt API key")

// encryptWithRawKey tries to encrypt with a raw base64 encoded key.
// Returns an error if encryption fails, unless AllowUnencryptedAPIKey is enabled.
func (c *Client) encryptWithRawKey() (string, error) {
	// Decode base64 public key.
	keyBytes, err := base64.StdEncoding.DecodeString(c.publicKey)
	if err != nil {
		if c.allowUnencryptedAPIKey {
			return c.apiKey, nil
		}
		return "", fmt.Errorf("%w: failed to decode public key: %w", ErrEncryptionFailed, err)
	}

	pub, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		if c.allowUnencryptedAPIKey {
			return c.apiKey, nil
		}
		return "", fmt.Errorf("%w: failed to parse public key: %w", ErrEncryptionFailed, err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		if c.allowUnencryptedAPIKey {
			return c.apiKey, nil
		}
		return "", fmt.Errorf("%w: not an RSA public key", ErrEncryptionFailed)
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPub, []byte(c.apiKey))
	if err != nil {
		if c.allowUnencryptedAPIKey {
			return c.apiKey, nil
		}
		return "", fmt.Errorf("%w: RSA encryption failed: %w", ErrEncryptionFailed, err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// formatPhoneForMPesa formats a phone number for M-Pesa API.
func formatPhoneForMPesa(phone contact.PhoneNumber) string {
	// M-Pesa Mozambique expects format: 258XXXXXXXXX (no + prefix)
	// phone.LocalNumber() returns 9-digit local number (e.g., "841234567")
	return contact.MozambiqueCountryCode + phone.LocalNumber()
}

// generateTransactionRef generates a unique transaction reference.
func generateTransactionRef() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based reference if crypto/rand fails
		return fmt.Sprintf("TXV%X", time.Now().UnixNano())
	}
	return fmt.Sprintf("TXV%X", b)
}

// Callback represents an M-Pesa callback notification.
type Callback struct {
	TransactionID       string `json:"output_TransactionID"`
	ConversationID      string `json:"output_ConversationID"`
	ThirdPartyReference string `json:"output_ThirdPartyReference"`
	ResponseCode        string `json:"output_ResponseCode"`
	ResponseDesc        string `json:"output_ResponseDesc"`
}

// ParseCallback parses an M-Pesa callback from JSON.
func ParseCallback(body []byte) (*Callback, error) {
	var callback Callback
	if err := json.Unmarshal(body, &callback); err != nil {
		return nil, fmt.Errorf("failed to parse callback: %w", err)
	}
	return &callback, nil
}

// HandleCallback processes an M-Pesa callback and publishes the appropriate event.
// It publishes PaymentCompleted on success or PaymentFailed on failure.
func (c *Client) HandleCallback(ctx context.Context, paymentID ids.PaymentID, callback *Callback) error {
	if callback == nil {
		return fmt.Errorf("callback is required")
	}

	if c.producer == nil {
		// No producer configured, nothing to publish
		return nil
	}

	if callback.ResponseCode == ResponseCodeSuccess {
		return c.publishPaymentCompleted(ctx, paymentID, callback.TransactionID)
	}
	return c.publishPaymentFailed(ctx, paymentID, callback.ResponseCode, callback.ResponseDesc)
}

// publishPaymentCompleted publishes a PaymentCompleted event to Kafka.
func (c *Client) publishPaymentCompleted(ctx context.Context, paymentID ids.PaymentID, transactionRef string) error {
	payload := events.PaymentCompleted{
		PaymentID:      paymentID,
		TransactionRef: transactionRef,
	}

	env, err := envelope.NewWithContext(ctx, &envelope.Config{
		Type:    string(events.EventTypePaymentCompleted),
		Version: events.VersionLatest,
		Source:  serviceName,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to create payment completed envelope: %w", err)
	}

	if err := c.producer.Publish(ctx, env, paymentID.String()); err != nil {
		return fmt.Errorf("failed to publish payment completed event: %w", err)
	}

	return nil
}

// publishPaymentFailed publishes a PaymentFailed event to Kafka.
func (c *Client) publishPaymentFailed(ctx context.Context, paymentID ids.PaymentID, errorCode, errorMessage string) error {
	payload := events.PaymentFailed{
		PaymentID:    paymentID,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	}

	env, err := envelope.NewWithContext(ctx, &envelope.Config{
		Type:    string(events.EventTypePaymentFailed),
		Version: events.VersionLatest,
		Source:  serviceName,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to create payment failed envelope: %w", err)
	}

	if err := c.producer.Publish(ctx, env, paymentID.String()); err != nil {
		return fmt.Errorf("failed to publish payment failed event: %w", err)
	}

	return nil
}

// IsSuccess returns true if the response indicates success.
func (r *InitiateResult) IsSuccess() bool {
	return r.ResponseCode == ResponseCodeSuccess
}

// IsSuccess returns true if the status indicates success.
func (s *TransactionStatus) IsSuccess() bool {
	return s.ResponseCode == ResponseCodeSuccess
}

// IsSuccess returns true if the refund was successful.
func (r *RefundResult) IsSuccess() bool {
	return r.ResponseCode == ResponseCodeSuccess
}

// Common M-Pesa response codes.
const (
	ResponseCodeSuccess           = "INS-0"
	ResponseCodeInsufficientFunds = "INS-1"
	ResponseCodeInvalidMSISDN     = "INS-5"
	ResponseCodeTimeout           = "INS-9"
	ResponseCodeDuplicateRequest  = "INS-10"
	ResponseCodeInvalidAmount     = "INS-13"
)

// ResponseCodeDescription returns a human-readable description for a response code.
func ResponseCodeDescription(code string) string {
	descriptions := map[string]string{
		"INS-0":  "Request processed successfully",
		"INS-1":  "Internal error",
		"INS-5":  "Transaction cancelled by customer",
		"INS-6":  "Transaction failed",
		"INS-9":  "Request timeout",
		"INS-10": "Duplicate transaction",
		"INS-13": "Invalid amount",
		"INS-14": "Invalid transaction reference",
		"INS-15": "Invalid transaction ID",
		"INS-16": "Customer MSISDN not found",
		"INS-17": "Transaction limit exceeded",
		"INS-18": "Customer account status problem",
		"INS-19": "Receiver account not found",
		"INS-20": "Service unavailable",
		"INS-21": "Reversal window expired",
		"INS-22": "Transaction already reversed",
		"INS-23": "Invalid security credential",
		"INS-24": "Invalid initiator identifier",
		"INS-25": "Invalid reversal amount",
	}

	if desc, ok := descriptions[code]; ok {
		return desc
	}
	return "Unknown error code: " + code
}

// SetHTTPClient sets a custom HTTP client (useful for testing).
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetBaseURL sets a custom base URL (useful for testing).
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}
