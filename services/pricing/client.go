// Package pricing provides a client for the Pricing Service API.
package pricing

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/geo"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
	"github.com/Dorico-Dynamics/txova-go-types/money"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

// Client is the Pricing Service client.
type Client struct {
	client *base.Client
}

// Config holds the configuration for the Pricing Service client.
type Config struct {
	// BaseURL is the base URL of the Pricing Service (required).
	BaseURL string

	// Timeout is the request timeout (default: 5s for fast pricing lookups).
	Timeout time.Duration

	// RetryConfig is the retry configuration.
	Retry base.RetryConfig

	// CircuitBreaker is the circuit breaker configuration.
	CircuitBreaker *base.CircuitBreakerConfig
}

// DefaultConfig returns a default configuration for the Pricing Service client.
func DefaultConfig(baseURL string) *Config {
	return &Config{
		BaseURL: baseURL,
		Timeout: 5 * time.Second,
		Retry:   base.DefaultRetryConfig(),
	}
}

// NewClient creates a new Pricing Service client.
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

// FareEstimate represents a fare estimate for a ride.
type FareEstimate struct {
	MinFare         money.Money       `json:"min_fare"`
	MaxFare         money.Money       `json:"max_fare"`
	EstimatedFare   money.Money       `json:"estimated_fare"`
	DistanceKM      float64           `json:"distance_km"`
	DurationMinutes int               `json:"duration_minutes"`
	SurgeMultiplier float64           `json:"surge_multiplier"`
	ServiceType     enums.ServiceType `json:"service_type"`
	BaseFare        money.Money       `json:"base_fare"`
	PerKMRate       money.Money       `json:"per_km_rate"`
	PerMinuteRate   money.Money       `json:"per_minute_rate"`
	BookingFee      money.Money       `json:"booking_fee"`
	ValidUntil      time.Time         `json:"valid_until"`
}

// SurgeInfo represents surge pricing information for a location.
type SurgeInfo struct {
	Multiplier float64      `json:"multiplier"`
	Location   geo.Location `json:"location"`
	Reason     string       `json:"reason,omitempty"`
	ExpiresAt  time.Time    `json:"expires_at"`
}

// FareValidation represents the result of fare validation.
type FareValidation struct {
	Valid             bool        `json:"valid"`
	ExpectedFare      money.Money `json:"expected_fare"`
	ActualFare        money.Money `json:"actual_fare"`
	Difference        money.Money `json:"difference"`
	DifferencePercent float64     `json:"difference_percent"`
	Reason            string      `json:"reason,omitempty"`
}

// GetEstimateRequest is the request body for getting a fare estimate.
type GetEstimateRequest struct {
	PickupLat   float64           `json:"pickup_lat"`
	PickupLon   float64           `json:"pickup_lon"`
	DropoffLat  float64           `json:"dropoff_lat"`
	DropoffLon  float64           `json:"dropoff_lon"`
	ServiceType enums.ServiceType `json:"service_type"`
}

// GetEstimate retrieves a fare estimate for a ride.
func (c *Client) GetEstimate(ctx context.Context, pickup, dropoff geo.Location, serviceType enums.ServiceType) (*FareEstimate, error) {
	if !serviceType.Valid() {
		return nil, fmt.Errorf("invalid service type")
	}

	req := GetEstimateRequest{
		PickupLat:   pickup.Latitude(),
		PickupLon:   pickup.Longitude(),
		DropoffLat:  dropoff.Latitude(),
		DropoffLon:  dropoff.Longitude(),
		ServiceType: serviceType,
	}

	var estimate FareEstimate
	err := c.client.Post(ctx, "/pricing/estimate", req).Decode(&estimate)
	if err != nil {
		return nil, err
	}

	return &estimate, nil
}

// GetSurgeMultiplier retrieves the current surge multiplier for a location.
func (c *Client) GetSurgeMultiplier(ctx context.Context, location geo.Location) (*SurgeInfo, error) {
	var surge SurgeInfo
	err := c.client.Get(ctx, "/pricing/surge").
		WithQuery("lat", fmt.Sprintf("%.6f", location.Latitude())).
		WithQuery("lon", fmt.Sprintf("%.6f", location.Longitude())).
		Decode(&surge)
	if err != nil {
		return nil, err
	}

	return &surge, nil
}

// ValidateFareRequest is the request body for validating a fare.
type ValidateFareRequest struct {
	Fare money.Money `json:"fare"`
}

// ValidateFare validates the fare for a completed ride.
func (c *Client) ValidateFare(ctx context.Context, rideID ids.RideID, fare money.Money) (*FareValidation, error) {
	if rideID.IsZero() {
		return nil, fmt.Errorf("ride ID is required")
	}

	req := ValidateFareRequest{Fare: fare}

	var validation FareValidation
	err := c.client.Post(ctx, fmt.Sprintf("/pricing/validate/%s", rideID), req).Decode(&validation)
	if err != nil {
		return nil, err
	}

	return &validation, nil
}

// GetServiceTypes retrieves all available service types with their base pricing.
func (c *Client) GetServiceTypes(ctx context.Context) ([]ServiceTypePricing, error) {
	var response struct {
		ServiceTypes []ServiceTypePricing `json:"service_types"`
	}

	err := c.client.Get(ctx, "/pricing/service-types").Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.ServiceTypes, nil
}

// ServiceTypePricing represents the pricing configuration for a service type.
type ServiceTypePricing struct {
	ServiceType   enums.ServiceType `json:"service_type"`
	DisplayName   string            `json:"display_name"`
	Description   string            `json:"description"`
	BaseFare      money.Money       `json:"base_fare"`
	PerKMRate     money.Money       `json:"per_km_rate"`
	PerMinuteRate money.Money       `json:"per_minute_rate"`
	MinimumFare   money.Money       `json:"minimum_fare"`
	BookingFee    money.Money       `json:"booking_fee"`
	Active        bool              `json:"active"`
}

// HealthCheck checks the health of the Pricing Service.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.client.Get(ctx, "/health").Do()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pricing service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
