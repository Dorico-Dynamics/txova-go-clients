// Package safety provides a client for the Safety Service API.
package safety

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/geo"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
	"github.com/Dorico-Dynamics/txova-go-types/rating"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

// Client is the Safety Service client.
type Client struct {
	client *base.Client
}

// Config holds the configuration for the Safety Service client.
type Config struct {
	// BaseURL is the base URL of the Safety Service (required).
	BaseURL string

	// Timeout is the request timeout (default: 10s).
	Timeout time.Duration

	// RetryConfig is the retry configuration.
	Retry base.RetryConfig

	// CircuitBreaker is the circuit breaker configuration.
	CircuitBreaker *base.CircuitBreakerConfig
}

// DefaultConfig returns a default configuration for the Safety Service client.
func DefaultConfig(baseURL string) *Config {
	return &Config{
		BaseURL: baseURL,
		Timeout: 10 * time.Second,
		Retry:   base.DefaultRetryConfig(),
	}
}

// NewClient creates a new Safety Service client.
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

// RatingAggregate represents aggregated rating information.
type RatingAggregate struct {
	AverageRating rating.Rating `json:"average_rating"`
	TotalRatings  int           `json:"total_ratings"`
	FiveStars     int           `json:"five_stars"`
	FourStars     int           `json:"four_stars"`
	ThreeStars    int           `json:"three_stars"`
	TwoStars      int           `json:"two_stars"`
	OneStar       int           `json:"one_star"`
}

// IncidentReport represents a safety incident report.
type IncidentReport struct {
	RideID      ids.RideID             `json:"ride_id"`
	ReporterID  ids.UserID             `json:"reporter_id"`
	ReportedID  *ids.UserID            `json:"reported_id,omitempty"`
	Severity    enums.IncidentSeverity `json:"severity"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Location    *geo.Location          `json:"location,omitempty"`
}

// Incident represents a safety incident in the system.
type Incident struct {
	ID          ids.IncidentID         `json:"id"`
	RideID      ids.RideID             `json:"ride_id"`
	ReporterID  ids.UserID             `json:"reporter_id"`
	ReportedID  *ids.UserID            `json:"reported_id,omitempty"`
	Severity    enums.IncidentSeverity `json:"severity"`
	Status      enums.IncidentStatus   `json:"status"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Location    *geo.Location          `json:"location,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
}

// GetUserRating retrieves the aggregated rating for a user.
func (c *Client) GetUserRating(ctx context.Context, userID ids.UserID) (*RatingAggregate, error) {
	if userID.IsZero() {
		return nil, fmt.Errorf("user ID is required")
	}

	var ra RatingAggregate
	err := c.client.Get(ctx, fmt.Sprintf("/users/%s/rating", userID)).Decode(&ra)
	if err != nil {
		return nil, err
	}

	return &ra, nil
}

// GetDriverRating retrieves the aggregated rating for a driver.
func (c *Client) GetDriverRating(ctx context.Context, driverID ids.DriverID) (*RatingAggregate, error) {
	if driverID.IsZero() {
		return nil, fmt.Errorf("driver ID is required")
	}

	var ra RatingAggregate
	err := c.client.Get(ctx, fmt.Sprintf("/drivers/%s/rating", driverID)).Decode(&ra)
	if err != nil {
		return nil, err
	}

	return &ra, nil
}

// ReportIncident reports a safety incident.
func (c *Client) ReportIncident(ctx context.Context, report *IncidentReport) (*Incident, error) {
	if report == nil {
		return nil, fmt.Errorf("incident report is required")
	}
	if report.RideID.IsZero() {
		return nil, fmt.Errorf("ride ID is required")
	}
	if report.ReporterID.IsZero() {
		return nil, fmt.Errorf("reporter ID is required")
	}
	if !report.Severity.Valid() {
		return nil, fmt.Errorf("invalid incident severity")
	}
	if report.Description == "" {
		return nil, fmt.Errorf("description is required")
	}

	var incident Incident
	err := c.client.Post(ctx, "/incidents", report).Decode(&incident)
	if err != nil {
		return nil, err
	}

	return &incident, nil
}

// GetIncident retrieves an incident by its ID.
func (c *Client) GetIncident(ctx context.Context, incidentID ids.IncidentID) (*Incident, error) {
	if incidentID.IsZero() {
		return nil, fmt.Errorf("incident ID is required")
	}

	var incident Incident
	err := c.client.Get(ctx, fmt.Sprintf("/incidents/%s", incidentID)).Decode(&incident)
	if err != nil {
		return nil, err
	}

	return &incident, nil
}

// TriggerEmergencyRequest is the request body for triggering an emergency.
type TriggerEmergencyRequest struct {
	Location geo.Location `json:"location"`
}

// TriggerEmergency triggers an emergency alert for a ride.
func (c *Client) TriggerEmergency(ctx context.Context, rideID ids.RideID, location geo.Location) error {
	if rideID.IsZero() {
		return fmt.Errorf("ride ID is required")
	}

	req := TriggerEmergencyRequest{Location: location}
	resp, err := c.client.Post(ctx, fmt.Sprintf("/rides/%s/emergency", rideID), req).Do()
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return resp.DecodeError()
	}

	return nil
}

// HealthCheck checks the health of the Safety Service.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.client.Get(ctx, "/health").Do()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("safety service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
