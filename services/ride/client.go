// Package ride provides a client for the Ride Service API.
package ride

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/geo"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
	"github.com/Dorico-Dynamics/txova-go-types/money"
	"github.com/Dorico-Dynamics/txova-go-types/pagination"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

// Client is the Ride Service client.
type Client struct {
	client *base.Client
}

// Config holds the configuration for the Ride Service client.
type Config struct {
	// BaseURL is the base URL of the Ride Service (required).
	BaseURL string

	// Timeout is the request timeout (default: 10s).
	Timeout time.Duration

	// RetryConfig is the retry configuration.
	Retry base.RetryConfig

	// CircuitBreaker is the circuit breaker configuration.
	CircuitBreaker *base.CircuitBreakerConfig
}

// DefaultConfig returns a default configuration for the Ride Service client.
func DefaultConfig(baseURL string) *Config {
	return &Config{
		BaseURL: baseURL,
		Timeout: 10 * time.Second,
		Retry:   base.DefaultRetryConfig(),
	}
}

// NewClient creates a new Ride Service client.
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

// Ride represents a ride in the system.
type Ride struct {
	ID              ids.RideID                `json:"id"`
	RiderID         ids.UserID                `json:"rider_id"`
	DriverID        *ids.DriverID             `json:"driver_id,omitempty"`
	ServiceType     enums.ServiceType         `json:"service_type"`
	Status          enums.RideStatus          `json:"status"`
	PickupLocation  geo.Location              `json:"pickup_location"`
	DropoffLocation geo.Location              `json:"dropoff_location"`
	PickupAddress   string                    `json:"pickup_address,omitempty"`
	DropoffAddress  string                    `json:"dropoff_address,omitempty"`
	EstimatedFare   money.Money               `json:"estimated_fare"`
	ActualFare      *money.Money              `json:"actual_fare,omitempty"`
	DistanceKM      float64                   `json:"distance_km"`
	DurationMinutes int                       `json:"duration_minutes"`
	RequestedAt     time.Time                 `json:"requested_at"`
	AcceptedAt      *time.Time                `json:"accepted_at,omitempty"`
	StartedAt       *time.Time                `json:"started_at,omitempty"`
	CompletedAt     *time.Time                `json:"completed_at,omitempty"`
	CancelledAt     *time.Time                `json:"cancelled_at,omitempty"`
	CancelReason    *enums.CancellationReason `json:"cancel_reason,omitempty"`
}

// GetRide retrieves a ride by its ID.
func (c *Client) GetRide(ctx context.Context, rideID ids.RideID) (*Ride, error) {
	if rideID.IsZero() {
		return nil, fmt.Errorf("ride ID is required")
	}

	var ride Ride
	err := c.client.Get(ctx, fmt.Sprintf("/rides/%s", rideID)).Decode(&ride)
	if err != nil {
		return nil, err
	}

	return &ride, nil
}

// GetActiveRide retrieves the active ride for a user.
func (c *Client) GetActiveRide(ctx context.Context, userID ids.UserID) (*Ride, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf("user ID is required")
	}

	var ride Ride
	err := c.client.Get(ctx, "/rides/active").
		WithQuery("user_id", userID.String()).
		Decode(&ride)
	if err != nil {
		return nil, err
	}

	return &ride, nil
}

// GetRideHistory retrieves the ride history for a user with pagination.
func (c *Client) GetRideHistory(ctx context.Context, userID ids.UserID, page pagination.PageRequest) (*pagination.PageResponse[Ride], error) {
	if userID.IsZero() {
		return nil, fmt.Errorf("user ID is required")
	}

	page = page.Normalize()

	var response pagination.PageResponse[Ride]
	err := c.client.Get(ctx, "/rides/history").
		WithQuery("user_id", userID.String()).
		WithQuery("limit", strconv.Itoa(page.Limit)).
		WithQuery("offset", strconv.Itoa(page.Offset)).
		Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// CancelRideRequest is the request body for cancelling a ride.
type CancelRideRequest struct {
	Reason enums.CancellationReason `json:"reason"`
}

// CancelRide cancels a ride.
func (c *Client) CancelRide(ctx context.Context, rideID ids.RideID, reason enums.CancellationReason) error {
	if rideID.IsZero() {
		return fmt.Errorf("ride ID is required")
	}
	if !reason.Valid() {
		return fmt.Errorf("invalid cancellation reason")
	}

	req := CancelRideRequest{Reason: reason}
	resp, err := c.client.Post(ctx, fmt.Sprintf("/rides/%s/cancel", rideID), req).Do()
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return resp.DecodeError()
	}

	return nil
}

// GetRideStatus retrieves the status of a ride.
func (c *Client) GetRideStatus(ctx context.Context, rideID ids.RideID) (enums.RideStatus, error) {
	if rideID.IsZero() {
		return "", fmt.Errorf("ride ID is required")
	}

	var response struct {
		Status enums.RideStatus `json:"status"`
	}

	err := c.client.Get(ctx, fmt.Sprintf("/rides/%s/status", rideID)).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Status, nil
}

// HealthCheck checks the health of the Ride Service.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.client.Get(ctx, "/health").Do()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ride service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
