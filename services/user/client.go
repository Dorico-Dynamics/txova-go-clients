// Package user provides a client for the User Service API.
package user

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/contact"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/ids"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

// Client is the User Service client.
type Client struct {
	client *base.Client
}

// Config holds the configuration for the User Service client.
type Config struct {
	// BaseURL is the base URL of the User Service (required).
	BaseURL string

	// Timeout is the request timeout (default: 10s).
	Timeout time.Duration

	// RetryConfig is the retry configuration.
	Retry base.RetryConfig

	// CircuitBreaker is the circuit breaker configuration.
	CircuitBreaker *base.CircuitBreakerConfig
}

// DefaultConfig returns a default configuration for the User Service client.
func DefaultConfig(baseURL string) *Config {
	return &Config{
		BaseURL: baseURL,
		Timeout: 10 * time.Second,
		Retry:   base.DefaultRetryConfig(),
	}
}

// defaultTimeout is the default timeout for User Service requests.
const defaultTimeout = 10 * time.Second

// NewClient creates a new User Service client.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Apply default timeout when not specified.
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	baseCfg := &base.Config{
		BaseURL:        cfg.BaseURL,
		Timeout:        timeout,
		RequestTimeout: timeout,
		Retry:          cfg.Retry,
		CircuitBreaker: cfg.CircuitBreaker,
	}

	baseClient, err := base.NewClient(baseCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base client: %w", err)
	}

	return &Client{client: baseClient}, nil
}

// User represents a user in the system.
type User struct {
	ID          ids.UserID          `json:"id"`
	Phone       contact.PhoneNumber `json:"phone"`
	Email       string              `json:"email,omitempty"`
	FirstName   string              `json:"first_name"`
	LastName    string              `json:"last_name"`
	Type        enums.UserType      `json:"type"`
	Status      enums.UserStatus    `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	VerifiedAt  *time.Time          `json:"verified_at,omitempty"`
	SuspendedAt *time.Time          `json:"suspended_at,omitempty"`
}

// GetUser retrieves a user by their ID.
func (c *Client) GetUser(ctx context.Context, userID ids.UserID) (*User, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf("user ID is required")
	}

	var user User
	err := c.client.Get(ctx, fmt.Sprintf("/users/%s", userID)).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByPhone retrieves a user by their phone number.
func (c *Client) GetUserByPhone(ctx context.Context, phone contact.PhoneNumber) (*User, error) {
	if phone.IsZero() {
		return nil, fmt.Errorf("phone number is required")
	}

	var user User
	err := c.client.Get(ctx, "/users/by-phone").
		WithQuery("phone", phone.String()).
		Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// VerifyUser marks a user as verified.
func (c *Client) VerifyUser(ctx context.Context, userID ids.UserID) error {
	if userID.IsZero() {
		return fmt.Errorf("user ID is required")
	}

	resp, err := c.client.Post(ctx, fmt.Sprintf("/users/%s/verify", userID), nil).Do()
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return resp.DecodeError()
	}

	return nil
}

// SuspendUserRequest is the request body for suspending a user.
type SuspendUserRequest struct {
	Reason string `json:"reason"`
}

// SuspendUser suspends a user account.
func (c *Client) SuspendUser(ctx context.Context, userID ids.UserID, reason string) error {
	if userID.IsZero() {
		return fmt.Errorf("user ID is required")
	}
	if reason == "" {
		return fmt.Errorf("suspension reason is required")
	}

	req := SuspendUserRequest{Reason: reason}
	resp, err := c.client.Post(ctx, fmt.Sprintf("/users/%s/suspend", userID), req).Do()
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return resp.DecodeError()
	}

	return nil
}

// GetUserStatus retrieves the status of a user.
func (c *Client) GetUserStatus(ctx context.Context, userID ids.UserID) (enums.UserStatus, error) {
	if userID.IsZero() {
		return "", fmt.Errorf("user ID is required")
	}

	var response struct {
		Status enums.UserStatus `json:"status"`
	}

	err := c.client.Get(ctx, fmt.Sprintf("/users/%s/status", userID)).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Status, nil
}

// HealthCheck checks the health of the User Service.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.client.Get(ctx, "/health").Do()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("user service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
