// Package payment provides a client for the Payment Service API.
package payment

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
	"github.com/Dorico-Dynamics/txova-go-types/money"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

// Client is the Payment Service client.
type Client struct {
	client *base.Client
}

// Config holds the configuration for the Payment Service client.
type Config struct {
	// BaseURL is the base URL of the Payment Service (required).
	BaseURL string

	// Timeout is the request timeout (default: 15s for payment operations).
	Timeout time.Duration

	// RetryConfig is the retry configuration.
	Retry base.RetryConfig

	// CircuitBreaker is the circuit breaker configuration.
	CircuitBreaker *base.CircuitBreakerConfig
}

// DefaultConfig returns a default configuration for the Payment Service client.
func DefaultConfig(baseURL string) *Config {
	return &Config{
		BaseURL: baseURL,
		Timeout: 15 * time.Second,
		Retry:   base.DefaultRetryConfig(),
	}
}

// NewClient creates a new Payment Service client.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	baseCfg := &base.Config{
		BaseURL:        cfg.BaseURL,
		Timeout:        cfg.Timeout,
		RequestTimeout: cfg.Timeout,
		Retry:          cfg.Retry,
		CircuitBreaker: cfg.CircuitBreaker,
	}

	baseClient, err := base.NewClient(baseCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base client: %w", err)
	}

	return &Client{client: baseClient}, nil
}

// Payment represents a payment in the system.
type Payment struct {
	ID            ids.PaymentID       `json:"id"`
	RideID        ids.RideID          `json:"ride_id"`
	UserID        ids.UserID          `json:"user_id"`
	Amount        money.Money         `json:"amount"`
	Method        enums.PaymentMethod `json:"method"`
	Status        enums.PaymentStatus `json:"status"`
	TransactionID string              `json:"transaction_id,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	CompletedAt   *time.Time          `json:"completed_at,omitempty"`
	FailedAt      *time.Time          `json:"failed_at,omitempty"`
	FailureReason string              `json:"failure_reason,omitempty"`
}

// Refund represents a refund in the system.
type Refund struct {
	ID        string              `json:"id"`
	PaymentID ids.PaymentID       `json:"payment_id"`
	Amount    money.Money         `json:"amount"`
	Reason    string              `json:"reason"`
	Status    enums.PaymentStatus `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
}

// WalletBalance represents a user's wallet balance.
type WalletBalance struct {
	UserID    ids.UserID  `json:"user_id"`
	Balance   money.Money `json:"balance"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// GetPayment retrieves a payment by its ID.
func (c *Client) GetPayment(ctx context.Context, paymentID ids.PaymentID) (*Payment, error) {
	if paymentID.IsZero() {
		return nil, fmt.Errorf("payment ID is required")
	}

	var payment Payment
	err := c.client.Get(ctx, fmt.Sprintf("/payments/%s", paymentID)).Decode(&payment)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

// GetPaymentByRide retrieves the payment for a ride.
func (c *Client) GetPaymentByRide(ctx context.Context, rideID ids.RideID) (*Payment, error) {
	if rideID.IsZero() {
		return nil, fmt.Errorf("ride ID is required")
	}

	var payment Payment
	err := c.client.Get(ctx, "/payments/by-ride").
		WithQuery("ride_id", rideID.String()).
		Decode(&payment)
	if err != nil {
		return nil, err
	}

	return &payment, nil
}

// InitiateRefundRequest is the request body for initiating a refund.
type InitiateRefundRequest struct {
	Amount money.Money `json:"amount"`
	Reason string      `json:"reason"`
}

// InitiateRefund initiates a refund for a payment.
func (c *Client) InitiateRefund(ctx context.Context, paymentID ids.PaymentID, amount money.Money, reason string) (*Refund, error) {
	if paymentID.IsZero() {
		return nil, fmt.Errorf("payment ID is required")
	}
	if reason == "" {
		return nil, fmt.Errorf("refund reason is required")
	}

	req := InitiateRefundRequest{
		Amount: amount,
		Reason: reason,
	}

	var refund Refund
	err := c.client.Post(ctx, fmt.Sprintf("/payments/%s/refund", paymentID), req).Decode(&refund)
	if err != nil {
		return nil, err
	}

	return &refund, nil
}

// GetWalletBalance retrieves the wallet balance for a user.
func (c *Client) GetWalletBalance(ctx context.Context, userID ids.UserID) (*WalletBalance, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf("user ID is required")
	}

	var balance WalletBalance
	err := c.client.Get(ctx, fmt.Sprintf("/wallets/%s/balance", userID)).Decode(&balance)
	if err != nil {
		return nil, err
	}

	return &balance, nil
}

// GetPaymentStatus retrieves the status of a payment.
func (c *Client) GetPaymentStatus(ctx context.Context, paymentID ids.PaymentID) (enums.PaymentStatus, error) {
	if paymentID.IsZero() {
		return "", fmt.Errorf("payment ID is required")
	}

	var response struct {
		Status enums.PaymentStatus `json:"status"`
	}

	err := c.client.Get(ctx, fmt.Sprintf("/payments/%s/status", paymentID)).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Status, nil
}

// HealthCheck checks the health of the Payment Service.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.client.Get(ctx, "/health").Do()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("payment service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
