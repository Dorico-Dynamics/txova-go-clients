// Package push provides a client for push notifications via Firebase Cloud Messaging.
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
)

// Firebase Cloud Messaging API endpoint.
const fcmAPIURL = "https://fcm.googleapis.com/v1/projects/%s/messages:send"

// Client is the push notification client for Firebase Cloud Messaging.
type Client struct {
	httpClient  *http.Client
	projectID   string
	accessToken string
	logger      *logging.Logger
}

// Config holds the configuration for the push notification client.
type Config struct {
	// ProjectID is the Firebase project ID (required).
	ProjectID string

	// AccessToken is the OAuth2 access token for FCM API (required).
	// This should be obtained from Firebase Admin SDK or service account.
	AccessToken string

	// Timeout is the request timeout (default: 30s).
	Timeout time.Duration
}

// NewClient creates a new push notification client.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}
	if cfg.AccessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		httpClient:  &http.Client{Timeout: timeout},
		projectID:   cfg.ProjectID,
		accessToken: cfg.AccessToken,
		logger:      logger,
	}, nil
}

// Notification represents a push notification.
type Notification struct {
	// Title is the notification title.
	Title string `json:"title,omitempty"`

	// Body is the notification body text.
	Body string `json:"body,omitempty"`

	// ImageURL is an optional image URL to display.
	ImageURL string `json:"image,omitempty"`
}

// AndroidConfig represents Android-specific notification options.
type AndroidConfig struct {
	// Priority is the message priority ("high" or "normal").
	Priority string `json:"priority,omitempty"`

	// TTL is the time-to-live in seconds.
	TTL string `json:"ttl,omitempty"`

	// CollapseKey is the collapse key for grouping notifications.
	CollapseKey string `json:"collapse_key,omitempty"`
}

// APNSConfig represents iOS-specific notification options.
type APNSConfig struct {
	// Headers are APNs headers.
	Headers map[string]string `json:"headers,omitempty"`
}

// Message represents a complete FCM message.
type Message struct {
	// Token is the device registration token.
	Token string `json:"token,omitempty"`

	// Topic is the topic to send to.
	Topic string `json:"topic,omitempty"`

	// Condition is the condition expression for targeting.
	Condition string `json:"condition,omitempty"`

	// Notification is the notification payload.
	Notification *Notification `json:"notification,omitempty"`

	// Data is the custom data payload.
	Data map[string]string `json:"data,omitempty"`

	// Android is Android-specific configuration.
	Android *AndroidConfig `json:"android,omitempty"`

	// APNS is iOS-specific configuration.
	APNS *APNSConfig `json:"apns,omitempty"`
}

// fcmRequest is the FCM API request body.
type fcmRequest struct {
	Message Message `json:"message"`
}

// fcmResponse is the FCM API response.
type fcmResponse struct {
	Name  string `json:"name"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// SendResult represents the result of sending a push notification.
type SendResult struct {
	// MessageID is the unique message identifier.
	MessageID string `json:"message_id"`

	// Success indicates if the send was successful.
	Success bool `json:"success"`

	// Error is the error message if failed.
	Error string `json:"error,omitempty"`
}

// BatchResult represents the result of a batch send operation.
type BatchResult struct {
	// SuccessCount is the number of successful sends.
	SuccessCount int `json:"success_count"`

	// FailureCount is the number of failed sends.
	FailureCount int `json:"failure_count"`

	// Results is the individual results for each token.
	Results []SendResult `json:"results"`
}

// SendToDevice sends a notification to a specific device token.
func (c *Client) SendToDevice(ctx context.Context, token string, notification *Notification, data map[string]string) (*SendResult, error) {
	if token == "" {
		return nil, fmt.Errorf("device token is required")
	}

	msg := Message{
		Token:        token,
		Notification: notification,
		Data:         data,
	}

	return c.sendMessage(ctx, msg)
}

// SendToTopic sends a notification to all devices subscribed to a topic.
func (c *Client) SendToTopic(ctx context.Context, topic string, notification *Notification, data map[string]string) (*SendResult, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	msg := Message{
		Topic:        topic,
		Notification: notification,
		Data:         data,
	}

	return c.sendMessage(ctx, msg)
}

// SendMessage sends a custom message.
func (c *Client) SendMessage(ctx context.Context, msg Message) (*SendResult, error) {
	if msg.Token == "" && msg.Topic == "" && msg.Condition == "" {
		return nil, fmt.Errorf("at least one of token, topic, or condition is required")
	}

	return c.sendMessage(ctx, msg)
}

// SendMulticast sends a notification to multiple device tokens.
func (c *Client) SendMulticast(ctx context.Context, tokens []string, notification *Notification, data map[string]string) (*BatchResult, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("at least one token is required")
	}

	result := &BatchResult{
		Results: make([]SendResult, len(tokens)),
	}

	for i, token := range tokens {
		msg := Message{
			Token:        token,
			Notification: notification,
			Data:         data,
		}

		sendResult, err := c.sendMessage(ctx, msg)
		if err != nil {
			result.Results[i] = SendResult{
				Success: false,
				Error:   err.Error(),
			}
			result.FailureCount++
		} else {
			result.Results[i] = *sendResult
			if sendResult.Success {
				result.SuccessCount++
			} else {
				result.FailureCount++
			}
		}
	}

	return result, nil
}

// apiURL is the URL for the FCM API (can be overridden for testing).
var apiURL = fcmAPIURL

func (c *Client) sendMessage(ctx context.Context, msg Message) (*SendResult, error) {
	reqBody := fcmRequest{Message: msg}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf(apiURL, c.projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var fcmResp fcmResponse
	if err := json.NewDecoder(resp.Body).Decode(&fcmResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if fcmResp.Error != nil {
		return &SendResult{
			Success: false,
			Error:   fcmResp.Error.Message,
		}, nil
	}

	return &SendResult{
		MessageID: fcmResp.Name,
		Success:   true,
	}, nil
}

// Common notification types.

// NewRideNotification creates a notification for a new ride request.
func NewRideNotification(pickup string) *Notification {
	return &Notification{
		Title: "New Ride Request",
		Body:  fmt.Sprintf("Pickup at %s", pickup),
	}
}

// RideAcceptedNotification creates a notification for ride acceptance.
func RideAcceptedNotification(driverName string, eta int) *Notification {
	return &Notification{
		Title: "Ride Accepted",
		Body:  fmt.Sprintf("%s is on the way. ETA: %d mins", driverName, eta),
	}
}

// DriverArrivedNotification creates a notification for driver arrival.
func DriverArrivedNotification(driverName string) *Notification {
	return &Notification{
		Title: "Driver Arrived",
		Body:  fmt.Sprintf("%s has arrived at your pickup location", driverName),
	}
}

// RideCompletedNotification creates a notification for ride completion.
func RideCompletedNotification(fare string) *Notification {
	return &Notification{
		Title: "Ride Completed",
		Body:  fmt.Sprintf("Thanks for riding with Txova! Your fare: %s", fare),
	}
}

// PaymentReceivedNotification creates a notification for payment receipt.
func PaymentReceivedNotification(amount string) *Notification {
	return &Notification{
		Title: "Payment Received",
		Body:  fmt.Sprintf("You received %s", amount),
	}
}

// SetAPIURL sets a custom API URL (useful for testing).
func SetAPIURL(url string) {
	apiURL = url
}

// ResetAPIURL resets the API URL to the default.
func ResetAPIURL() {
	apiURL = fcmAPIURL
}

// SetHTTPClient sets a custom HTTP client (useful for testing).
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}
