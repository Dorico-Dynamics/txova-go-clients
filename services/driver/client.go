// Package driver provides a client for the Driver Service API.
package driver

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
	"github.com/Dorico-Dynamics/txova-go-types/rating"
	"github.com/Dorico-Dynamics/txova-go-types/vehicle"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

// Client is the Driver Service client.
type Client struct {
	client *base.Client
}

// Config holds the configuration for the Driver Service client.
type Config struct {
	// BaseURL is the base URL of the Driver Service (required).
	BaseURL string

	// Timeout is the request timeout (default: 10s).
	Timeout time.Duration

	// RetryConfig is the retry configuration.
	Retry base.RetryConfig

	// CircuitBreaker is the circuit breaker configuration.
	CircuitBreaker *base.CircuitBreakerConfig
}

// DefaultConfig returns a default configuration for the Driver Service client.
func DefaultConfig(baseURL string) *Config {
	return &Config{
		BaseURL: baseURL,
		Timeout: 10 * time.Second,
		Retry:   base.DefaultRetryConfig(),
	}
}

// NewClient creates a new Driver Service client.
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

// Driver represents a driver in the system.
type Driver struct {
	ID                 ids.DriverID             `json:"id"`
	UserID             ids.UserID               `json:"user_id"`
	Status             enums.DriverStatus       `json:"status"`
	AvailabilityStatus enums.AvailabilityStatus `json:"availability_status"`
	Rating             rating.Rating            `json:"rating"`
	TotalRides         int                      `json:"total_rides"`
	CurrentLocation    *geo.Location            `json:"current_location,omitempty"`
	ActiveVehicleID    *ids.VehicleID           `json:"active_vehicle_id,omitempty"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
}

// Vehicle represents a vehicle in the system.
type Vehicle struct {
	ID           ids.VehicleID        `json:"id"`
	DriverID     ids.DriverID         `json:"driver_id"`
	LicensePlate vehicle.LicensePlate `json:"license_plate"`
	Make         string               `json:"make"`
	Model        string               `json:"model"`
	Year         int                  `json:"year"`
	Color        string               `json:"color"`
	Status       enums.VehicleStatus  `json:"status"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
}

// NearbyDriver represents a driver near a location.
type NearbyDriver struct {
	Driver     Driver       `json:"driver"`
	Location   geo.Location `json:"location"`
	DistanceKM float64      `json:"distance_km"`
}

// GetDriver retrieves a driver by their ID.
func (c *Client) GetDriver(ctx context.Context, driverID ids.DriverID) (*Driver, error) {
	if driverID.IsZero() {
		return nil, fmt.Errorf("driver ID is required")
	}

	var driver Driver
	err := c.client.Get(ctx, fmt.Sprintf("/drivers/%s", driverID)).Decode(&driver)
	if err != nil {
		return nil, err
	}

	return &driver, nil
}

// GetDriverByUserID retrieves a driver by their user ID.
func (c *Client) GetDriverByUserID(ctx context.Context, userID ids.UserID) (*Driver, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf("user ID is required")
	}

	var driver Driver
	err := c.client.Get(ctx, "/drivers/by-user").
		WithQuery("user_id", userID.String()).
		Decode(&driver)
	if err != nil {
		return nil, err
	}

	return &driver, nil
}

// GetActiveVehicle retrieves the active vehicle for a driver.
func (c *Client) GetActiveVehicle(ctx context.Context, driverID ids.DriverID) (*Vehicle, error) {
	if driverID.IsZero() {
		return nil, fmt.Errorf("driver ID is required")
	}

	var v Vehicle
	err := c.client.Get(ctx, fmt.Sprintf("/drivers/%s/vehicle", driverID)).Decode(&v)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// RecordEarningsRequest is the request body for recording driver earnings.
type RecordEarningsRequest struct {
	RideID ids.RideID  `json:"ride_id"`
	Amount money.Money `json:"amount"`
}

// RecordEarnings records earnings for a driver from a completed ride.
func (c *Client) RecordEarnings(ctx context.Context, driverID ids.DriverID, rideID ids.RideID, amount money.Money) error {
	if driverID.IsZero() {
		return fmt.Errorf("driver ID is required")
	}
	if rideID.IsZero() {
		return fmt.Errorf("ride ID is required")
	}

	req := RecordEarningsRequest{
		RideID: rideID,
		Amount: amount,
	}

	resp, err := c.client.Post(ctx, fmt.Sprintf("/drivers/%s/earnings", driverID), req).Do()
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return resp.DecodeError()
	}

	return nil
}

// GetDriverStatus retrieves the availability status of a driver.
func (c *Client) GetDriverStatus(ctx context.Context, driverID ids.DriverID) (enums.AvailabilityStatus, error) {
	if driverID.IsZero() {
		return "", fmt.Errorf("driver ID is required")
	}

	var response struct {
		Status enums.AvailabilityStatus `json:"status"`
	}

	err := c.client.Get(ctx, fmt.Sprintf("/drivers/%s/status", driverID)).Decode(&response)
	if err != nil {
		return "", err
	}

	return response.Status, nil
}

// GetNearbyDrivers retrieves drivers near a location within a radius.
func (c *Client) GetNearbyDrivers(ctx context.Context, location geo.Location, radiusKM float64) ([]*NearbyDriver, error) {
	var response struct {
		Drivers []*NearbyDriver `json:"drivers"`
	}

	err := c.client.Get(ctx, "/drivers/nearby").
		WithQuery("lat", fmt.Sprintf("%.6f", location.Latitude())).
		WithQuery("lon", fmt.Sprintf("%.6f", location.Longitude())).
		WithQuery("radius_km", fmt.Sprintf("%.2f", radiusKM)).
		Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Drivers, nil
}

// UpdateLocationRequest is the request body for updating driver location.
type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// UpdateLocation updates the current location of a driver.
func (c *Client) UpdateLocation(ctx context.Context, driverID ids.DriverID, location geo.Location) error {
	if driverID.IsZero() {
		return fmt.Errorf("driver ID is required")
	}

	req := UpdateLocationRequest{
		Latitude:  location.Latitude(),
		Longitude: location.Longitude(),
	}

	resp, err := c.client.Put(ctx, fmt.Sprintf("/drivers/%s/location", driverID), req).Do()
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return resp.DecodeError()
	}

	return nil
}

// SetAvailability sets the availability status of a driver.
func (c *Client) SetAvailability(ctx context.Context, driverID ids.DriverID, status enums.AvailabilityStatus) error {
	if driverID.IsZero() {
		return fmt.Errorf("driver ID is required")
	}
	if !status.Valid() {
		return fmt.Errorf("invalid availability status")
	}

	req := struct {
		Status enums.AvailabilityStatus `json:"status"`
	}{Status: status}

	resp, err := c.client.Put(ctx, fmt.Sprintf("/drivers/%s/availability", driverID), req).Do()
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return resp.DecodeError()
	}

	return nil
}

// HealthCheck checks the health of the Driver Service.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.client.Get(ctx, "/health").Do()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("driver service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
