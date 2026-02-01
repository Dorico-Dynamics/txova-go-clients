// Package factory provides a client factory for creating service clients.
package factory

import (
	"context"
	"fmt"
	"sync"

	"github.com/Dorico-Dynamics/txova-go-core/logging"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
	"github.com/Dorico-Dynamics/txova-go-clients/services/driver"
	"github.com/Dorico-Dynamics/txova-go-clients/services/payment"
	"github.com/Dorico-Dynamics/txova-go-clients/services/pricing"
	"github.com/Dorico-Dynamics/txova-go-clients/services/ride"
	"github.com/Dorico-Dynamics/txova-go-clients/services/safety"
	"github.com/Dorico-Dynamics/txova-go-clients/services/user"
)

// Config holds the configuration for the client factory.
type Config struct {
	// UserServiceURL is the base URL of the User Service.
	UserServiceURL string

	// DriverServiceURL is the base URL of the Driver Service.
	DriverServiceURL string

	// RideServiceURL is the base URL of the Ride Service.
	RideServiceURL string

	// PaymentServiceURL is the base URL of the Payment Service.
	PaymentServiceURL string

	// PricingServiceURL is the base URL of the Pricing Service.
	PricingServiceURL string

	// SafetyServiceURL is the base URL of the Safety Service.
	SafetyServiceURL string

	// Retry is the default retry configuration for all clients.
	Retry base.RetryConfig

	// CircuitBreaker is the default circuit breaker configuration for all clients.
	CircuitBreaker *base.CircuitBreakerConfig
}

// Factory is a client factory that creates and manages service clients.
// It provides lazy initialization and singleton pattern for each service client.
type Factory struct {
	cfg    *Config
	logger *logging.Logger

	mu      sync.RWMutex
	user    *user.Client
	driver  *driver.Client
	ride    *ride.Client
	payment *payment.Client
	pricing *pricing.Client
	safety  *safety.Client
}

// New creates a new client factory.
func New(cfg *Config, logger *logging.Logger) (*Factory, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	return &Factory{
		cfg:    cfg,
		logger: logger,
	}, nil
}

// User returns the User Service client, creating it if necessary.
func (f *Factory) User() (*user.Client, error) {
	f.mu.RLock()
	if f.user != nil {
		defer f.mu.RUnlock()
		return f.user, nil
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if f.user != nil {
		return f.user, nil
	}

	if f.cfg.UserServiceURL == "" {
		return nil, fmt.Errorf("user service URL is not configured")
	}

	cfg := &user.Config{
		BaseURL:        f.cfg.UserServiceURL,
		Retry:          f.cfg.Retry,
		CircuitBreaker: f.cfg.CircuitBreaker,
	}

	client, err := user.NewClient(cfg, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create user client: %w", err)
	}

	f.user = client
	return f.user, nil
}

// Driver returns the Driver Service client, creating it if necessary.
func (f *Factory) Driver() (*driver.Client, error) {
	f.mu.RLock()
	if f.driver != nil {
		defer f.mu.RUnlock()
		return f.driver, nil
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if f.driver != nil {
		return f.driver, nil
	}

	if f.cfg.DriverServiceURL == "" {
		return nil, fmt.Errorf("driver service URL is not configured")
	}

	cfg := &driver.Config{
		BaseURL:        f.cfg.DriverServiceURL,
		Retry:          f.cfg.Retry,
		CircuitBreaker: f.cfg.CircuitBreaker,
	}

	client, err := driver.NewClient(cfg, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver client: %w", err)
	}

	f.driver = client
	return f.driver, nil
}

// Ride returns the Ride Service client, creating it if necessary.
func (f *Factory) Ride() (*ride.Client, error) {
	f.mu.RLock()
	if f.ride != nil {
		defer f.mu.RUnlock()
		return f.ride, nil
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if f.ride != nil {
		return f.ride, nil
	}

	if f.cfg.RideServiceURL == "" {
		return nil, fmt.Errorf("ride service URL is not configured")
	}

	cfg := &ride.Config{
		BaseURL:        f.cfg.RideServiceURL,
		Retry:          f.cfg.Retry,
		CircuitBreaker: f.cfg.CircuitBreaker,
	}

	client, err := ride.NewClient(cfg, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ride client: %w", err)
	}

	f.ride = client
	return f.ride, nil
}

// Payment returns the Payment Service client, creating it if necessary.
func (f *Factory) Payment() (*payment.Client, error) {
	f.mu.RLock()
	if f.payment != nil {
		defer f.mu.RUnlock()
		return f.payment, nil
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if f.payment != nil {
		return f.payment, nil
	}

	if f.cfg.PaymentServiceURL == "" {
		return nil, fmt.Errorf("payment service URL is not configured")
	}

	cfg := &payment.Config{
		BaseURL:        f.cfg.PaymentServiceURL,
		Retry:          f.cfg.Retry,
		CircuitBreaker: f.cfg.CircuitBreaker,
	}

	client, err := payment.NewClient(cfg, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment client: %w", err)
	}

	f.payment = client
	return f.payment, nil
}

// Pricing returns the Pricing Service client, creating it if necessary.
func (f *Factory) Pricing() (*pricing.Client, error) {
	f.mu.RLock()
	if f.pricing != nil {
		defer f.mu.RUnlock()
		return f.pricing, nil
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if f.pricing != nil {
		return f.pricing, nil
	}

	if f.cfg.PricingServiceURL == "" {
		return nil, fmt.Errorf("pricing service URL is not configured")
	}

	cfg := &pricing.Config{
		BaseURL:        f.cfg.PricingServiceURL,
		Retry:          f.cfg.Retry,
		CircuitBreaker: f.cfg.CircuitBreaker,
	}

	client, err := pricing.NewClient(cfg, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create pricing client: %w", err)
	}

	f.pricing = client
	return f.pricing, nil
}

// Safety returns the Safety Service client, creating it if necessary.
func (f *Factory) Safety() (*safety.Client, error) {
	f.mu.RLock()
	if f.safety != nil {
		defer f.mu.RUnlock()
		return f.safety, nil
	}
	f.mu.RUnlock()

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if f.safety != nil {
		return f.safety, nil
	}

	if f.cfg.SafetyServiceURL == "" {
		return nil, fmt.Errorf("safety service URL is not configured")
	}

	cfg := &safety.Config{
		BaseURL:        f.cfg.SafetyServiceURL,
		Retry:          f.cfg.Retry,
		CircuitBreaker: f.cfg.CircuitBreaker,
	}

	client, err := safety.NewClient(cfg, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create safety client: %w", err)
	}

	f.safety = client
	return f.safety, nil
}

// ServiceHealth represents the health status of a service.
type ServiceHealth struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

// HealthCheck checks the health of all configured services.
// It returns a map of service names to their health check errors (nil if healthy).
func (f *Factory) HealthCheck(ctx context.Context) []ServiceHealth {
	var results []ServiceHealth
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Helper to add a health check result
	addResult := func(name string, healthy bool, err error) {
		mu.Lock()
		defer mu.Unlock()
		result := ServiceHealth{Name: name, Healthy: healthy}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}

	// Check each configured service
	if f.cfg.UserServiceURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := f.User()
			if err != nil {
				addResult("user", false, err)
				return
			}
			err = client.HealthCheck(ctx)
			addResult("user", err == nil, err)
		}()
	}

	if f.cfg.DriverServiceURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := f.Driver()
			if err != nil {
				addResult("driver", false, err)
				return
			}
			err = client.HealthCheck(ctx)
			addResult("driver", err == nil, err)
		}()
	}

	if f.cfg.RideServiceURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := f.Ride()
			if err != nil {
				addResult("ride", false, err)
				return
			}
			err = client.HealthCheck(ctx)
			addResult("ride", err == nil, err)
		}()
	}

	if f.cfg.PaymentServiceURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := f.Payment()
			if err != nil {
				addResult("payment", false, err)
				return
			}
			err = client.HealthCheck(ctx)
			addResult("payment", err == nil, err)
		}()
	}

	if f.cfg.PricingServiceURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := f.Pricing()
			if err != nil {
				addResult("pricing", false, err)
				return
			}
			err = client.HealthCheck(ctx)
			addResult("pricing", err == nil, err)
		}()
	}

	if f.cfg.SafetyServiceURL != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client, err := f.Safety()
			if err != nil {
				addResult("safety", false, err)
				return
			}
			err = client.HealthCheck(ctx)
			addResult("safety", err == nil, err)
		}()
	}

	wg.Wait()
	return results
}

// AllHealthy returns true if all configured services are healthy.
func (f *Factory) AllHealthy(ctx context.Context) bool {
	results := f.HealthCheck(ctx)
	for _, r := range results {
		if !r.Healthy {
			return false
		}
	}
	return true
}
