// Package sms provides a client for sending SMS messages via Africa's Talking.
package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/contact"
)

// Africa's Talking API endpoints.
const (
	productionBaseURL = "https://api.africastalking.com/version1"
	sandboxBaseURL    = "https://api.sandbox.africastalking.com/version1"
)

// Client is the SMS client for Africa's Talking.
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	apiKey     string
	senderID   string
	logger     *logging.Logger
}

// Config holds the configuration for the SMS client.
type Config struct {
	// Username is the Africa's Talking username (required).
	Username string

	// APIKey is the Africa's Talking API key (required).
	APIKey string

	// SenderID is the alphanumeric sender ID (optional, max 11 chars).
	SenderID string

	// Sandbox enables sandbox mode for testing.
	Sandbox bool

	// Timeout is the request timeout (default: 30s).
	Timeout time.Duration
}

// NewClient creates a new SMS client.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	baseURL := productionBaseURL
	if cfg.Sandbox {
		baseURL = sandboxBaseURL
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		username:   cfg.Username,
		apiKey:     cfg.APIKey,
		senderID:   cfg.SenderID,
		logger:     logger,
	}, nil
}

// SendResult represents the result of sending an SMS.
type SendResult struct {
	// MessageID is the unique identifier for the message.
	MessageID string `json:"messageId"`

	// Number is the recipient phone number.
	Number string `json:"number"`

	// Status is the delivery status.
	Status string `json:"status"`

	// StatusCode is the status code.
	StatusCode int `json:"statusCode"`

	// Cost is the cost of the message.
	Cost string `json:"cost"`
}

// SendResponse is the response from the SMS API.
type SendResponse struct {
	SMSMessageData struct {
		Message    string       `json:"Message"`
		Recipients []SendResult `json:"Recipients"`
	} `json:"SMSMessageData"`
}

// Send sends an SMS to a single recipient.
func (c *Client) Send(ctx context.Context, phone contact.PhoneNumber, message string) (*SendResult, error) {
	if phone.IsZero() {
		return nil, fmt.Errorf("phone number is required")
	}
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}

	results, err := c.sendSMS(ctx, []string{phone.String()}, message)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no result returned from API")
	}

	return &results[0], nil
}

// SendBulk sends an SMS to multiple recipients.
func (c *Client) SendBulk(ctx context.Context, phones []contact.PhoneNumber, message string) ([]SendResult, error) {
	if len(phones) == 0 {
		return nil, fmt.Errorf("at least one phone number is required")
	}
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}

	recipients := make([]string, len(phones))
	for i, p := range phones {
		if p.IsZero() {
			return nil, fmt.Errorf("phone number at index %d is invalid", i)
		}
		recipients[i] = p.String()
	}

	return c.sendSMS(ctx, recipients, message)
}

func (c *Client) sendSMS(ctx context.Context, recipients []string, message string) ([]SendResult, error) {
	data := url.Values{}
	data.Set("username", c.username)
	data.Set("to", strings.Join(recipients, ","))
	data.Set("message", message)

	if c.senderID != "" {
		data.Set("from", c.senderID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messaging", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("apiKey", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var sendResp SendResponse
	if err := json.Unmarshal(body, &sendResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return sendResp.SMSMessageData.Recipients, nil
}

// Balance represents the account balance.
type Balance struct {
	// Value is the balance amount.
	Value string `json:"balance"`
}

// GetBalance retrieves the account balance.
func (c *Client) GetBalance(ctx context.Context) (*Balance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.africastalking.com/version1/user?username="+c.username, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("apiKey", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		UserData struct {
			Balance string `json:"balance"`
		} `json:"UserData"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &Balance{Value: result.UserData.Balance}, nil
}

// DeliveryStatus represents the delivery status of a message.
type DeliveryStatus struct {
	// ID is the message ID.
	ID string `json:"id"`

	// Status is the delivery status.
	Status string `json:"status"`

	// NetworkCode is the mobile network code.
	NetworkCode string `json:"networkCode"`

	// FailureReason is the reason for failure (if any).
	FailureReason string `json:"failureReason,omitempty"`
}

// DeliveryCallback is the callback data sent by Africa's Talking for delivery reports.
type DeliveryCallback struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	PhoneNumber   string `json:"phoneNumber"`
	NetworkCode   string `json:"networkCode"`
	FailureReason string `json:"failureReason"`
	RetryCount    string `json:"retryCount"`
}

// ParseDeliveryCallback parses the delivery callback from Africa's Talking.
func ParseDeliveryCallback(body []byte) (*DeliveryCallback, error) {
	var callback DeliveryCallback
	if err := json.Unmarshal(body, &callback); err != nil {
		return nil, fmt.Errorf("failed to parse delivery callback: %w", err)
	}
	return &callback, nil
}

// ParseDeliveryCallbackForm parses the delivery callback from form data.
func ParseDeliveryCallbackForm(r *http.Request) (*DeliveryCallback, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	return &DeliveryCallback{
		ID:            r.FormValue("id"),
		Status:        r.FormValue("status"),
		PhoneNumber:   r.FormValue("phoneNumber"),
		NetworkCode:   r.FormValue("networkCode"),
		FailureReason: r.FormValue("failureReason"),
		RetryCount:    r.FormValue("retryCount"),
	}, nil
}

// HTTPClient interface for testing.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// SetHTTPClient sets a custom HTTP client (useful for testing).
func (c *Client) SetHTTPClient(client HTTPClient) {
	if hc, ok := client.(*http.Client); ok {
		c.httpClient = hc
	}
}

// NewMockResponse creates a mock HTTP response for testing.
func NewMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}
