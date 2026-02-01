package factory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Dorico-Dynamics/txova-go-clients/base"
)

func TestNew(t *testing.T) {
	t.Run("creates factory with valid config", func(t *testing.T) {
		cfg := &Config{
			UserServiceURL: "http://user-service:8080",
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f == nil {
			t.Fatal("expected factory, got nil")
		}
	})

	t.Run("returns error with nil config", func(t *testing.T) {
		_, err := New(nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFactory_User(t *testing.T) {
	t.Run("creates user client", func(t *testing.T) {
		cfg := &Config{
			UserServiceURL: "http://user-service:8080",
			Retry:          base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		client, err := f.User()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
	})

	t.Run("returns same instance on repeated calls", func(t *testing.T) {
		cfg := &Config{
			UserServiceURL: "http://user-service:8080",
			Retry:          base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		client1, err := f.User()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		client2, err := f.User()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client1 != client2 {
			t.Error("expected same instance, got different instances")
		}
	})

	t.Run("returns error when URL not configured", func(t *testing.T) {
		cfg := &Config{}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		_, err = f.User()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFactory_Driver(t *testing.T) {
	t.Run("creates driver client", func(t *testing.T) {
		cfg := &Config{
			DriverServiceURL: "http://driver-service:8080",
			Retry:            base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		client, err := f.Driver()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
	})

	t.Run("returns error when URL not configured", func(t *testing.T) {
		cfg := &Config{}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		_, err = f.Driver()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFactory_Ride(t *testing.T) {
	t.Run("creates ride client", func(t *testing.T) {
		cfg := &Config{
			RideServiceURL: "http://ride-service:8080",
			Retry:          base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		client, err := f.Ride()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
	})

	t.Run("returns error when URL not configured", func(t *testing.T) {
		cfg := &Config{}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		_, err = f.Ride()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFactory_Payment(t *testing.T) {
	t.Run("creates payment client", func(t *testing.T) {
		cfg := &Config{
			PaymentServiceURL: "http://payment-service:8080",
			Retry:             base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		client, err := f.Payment()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
	})

	t.Run("returns error when URL not configured", func(t *testing.T) {
		cfg := &Config{}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		_, err = f.Payment()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFactory_Pricing(t *testing.T) {
	t.Run("creates pricing client", func(t *testing.T) {
		cfg := &Config{
			PricingServiceURL: "http://pricing-service:8080",
			Retry:             base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		client, err := f.Pricing()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
	})

	t.Run("returns error when URL not configured", func(t *testing.T) {
		cfg := &Config{}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		_, err = f.Pricing()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFactory_Safety(t *testing.T) {
	t.Run("creates safety client", func(t *testing.T) {
		cfg := &Config{
			SafetyServiceURL: "http://safety-service:8080",
			Retry:            base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		client, err := f.Safety()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client == nil {
			t.Fatal("expected client, got nil")
		}
	})

	t.Run("returns error when URL not configured", func(t *testing.T) {
		cfg := &Config{}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		_, err = f.Safety()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestFactory_ConcurrentAccess(t *testing.T) {
	cfg := &Config{
		UserServiceURL:    "http://user-service:8080",
		DriverServiceURL:  "http://driver-service:8080",
		RideServiceURL:    "http://ride-service:8080",
		PaymentServiceURL: "http://payment-service:8080",
		PricingServiceURL: "http://pricing-service:8080",
		SafetyServiceURL:  "http://safety-service:8080",
		Retry:             base.RetryConfig{MaxRetries: 0},
	}
	f, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent access to each client type
	for i := 0; i < numGoroutines; i++ {
		wg.Add(6)
		go func() {
			defer wg.Done()
			_, _ = f.User()
		}()
		go func() {
			defer wg.Done()
			_, _ = f.Driver()
		}()
		go func() {
			defer wg.Done()
			_, _ = f.Ride()
		}()
		go func() {
			defer wg.Done()
			_, _ = f.Payment()
		}()
		go func() {
			defer wg.Done()
			_, _ = f.Pricing()
		}()
		go func() {
			defer wg.Done()
			_, _ = f.Safety()
		}()
	}

	wg.Wait()
}

func TestFactory_HealthCheck(t *testing.T) {
	// Create healthy mock servers
	userServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer userServer.Close()

	driverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer driverServer.Close()

	t.Run("returns health status for configured services", func(t *testing.T) {
		cfg := &Config{
			UserServiceURL:   userServer.URL,
			DriverServiceURL: driverServer.URL,
			Retry:            base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		results := f.HealthCheck(context.Background())
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}

		// Check that both services are healthy
		for _, r := range results {
			if !r.Healthy {
				t.Errorf("expected %s to be healthy, got error: %s", r.Name, r.Error)
			}
		}
	})

	t.Run("returns unhealthy for failing services", func(t *testing.T) {
		unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer unhealthyServer.Close()

		cfg := &Config{
			UserServiceURL: unhealthyServer.URL,
			Retry:          base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		results := f.HealthCheck(context.Background())
		if len(results) != 1 {
			t.Errorf("expected 1 result, got %d", len(results))
		}

		if results[0].Healthy {
			t.Error("expected service to be unhealthy")
		}
		if results[0].Error == "" {
			t.Error("expected error message")
		}
	})

	t.Run("returns empty results when no services configured", func(t *testing.T) {
		cfg := &Config{}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		results := f.HealthCheck(context.Background())
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}

func TestFactory_HealthCheck_AllServices(t *testing.T) {
	// Create mock servers for all services
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &Config{
		UserServiceURL:    server.URL,
		DriverServiceURL:  server.URL,
		RideServiceURL:    server.URL,
		PaymentServiceURL: server.URL,
		PricingServiceURL: server.URL,
		SafetyServiceURL:  server.URL,
		Retry:             base.RetryConfig{MaxRetries: 0},
	}
	f, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("failed to create factory: %v", err)
	}

	results := f.HealthCheck(context.Background())
	if len(results) != 6 {
		t.Errorf("expected 6 results, got %d", len(results))
	}

	// All should be healthy
	for _, r := range results {
		if !r.Healthy {
			t.Errorf("expected %s to be healthy, got error: %s", r.Name, r.Error)
		}
	}
}

func TestFactory_AllHealthy(t *testing.T) {
	t.Run("returns true when all services healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		cfg := &Config{
			UserServiceURL:   server.URL,
			DriverServiceURL: server.URL,
			Retry:            base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		if !f.AllHealthy(context.Background()) {
			t.Error("expected all services to be healthy")
		}
	})

	t.Run("returns false when any service unhealthy", func(t *testing.T) {
		healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer healthyServer.Close()

		unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer unhealthyServer.Close()

		cfg := &Config{
			UserServiceURL:   healthyServer.URL,
			DriverServiceURL: unhealthyServer.URL,
			Retry:            base.RetryConfig{MaxRetries: 0},
		}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		if f.AllHealthy(context.Background()) {
			t.Error("expected not all services to be healthy")
		}
	})

	t.Run("returns true when no services configured", func(t *testing.T) {
		cfg := &Config{}
		f, err := New(cfg, nil)
		if err != nil {
			t.Fatalf("failed to create factory: %v", err)
		}

		if !f.AllHealthy(context.Background()) {
			t.Error("expected all (zero) services to be healthy")
		}
	})
}
