# txova-go-clients

HTTP and external service client library providing typed clients for internal service-to-service communication and third-party API integrations.

## Overview

`txova-go-clients` provides robust HTTP clients for both internal Txova service communication and external third-party integrations, including retry logic with exponential backoff, circuit breakers, and proper error handling.

**Module:** `github.com/Dorico-Dynamics/txova-go-clients`

## Features

- **Base HTTP Client** - Connection pooling, exponential backoff retries, circuit breaker pattern
- **Service Clients** - Typed clients for all Txova microservices (User, Driver, Ride, Payment, Pricing, Safety)
- **External Integrations** - SMS (Africa's Talking), Email (SMTP/Resend), Storage (MinIO/S3), M-Pesa, Push Notifications (Firebase), Identity Verification (Smile Identity)
- **Factory Pattern** - Thread-safe lazy initialization with dependency injection
- **Structured Logging** - Integration with `slog` via txova-go-core

## Package Structure

```
txova-go-clients/
├── base/           # Base HTTP client with retry/circuit breaker
├── services/       # Internal service clients
│   ├── driver/     # Driver service client
│   ├── payment/    # Payment service client
│   ├── pricing/    # Pricing service client
│   ├── ride/       # Ride service client
│   ├── safety/     # Safety service client
│   └── user/       # User service client
├── external/       # Third-party API clients
│   ├── email/      # Email (SMTP + Resend)
│   ├── identity/   # Smile Identity verification
│   ├── mpesa/      # M-Pesa payments
│   ├── push/       # Firebase push notifications
│   ├── sms/        # Africa's Talking SMS
│   └── storage/    # MinIO/S3 storage
└── factory/        # Service client factory
```

## Installation

```bash
go get github.com/Dorico-Dynamics/txova-go-clients
```

## Quick Start

### Using the Factory Pattern (Recommended)

```go
import "github.com/Dorico-Dynamics/txova-go-clients/factory"

f := factory.New(factory.Config{
    UserServiceURL:    "http://user-service:8080",
    DriverServiceURL:  "http://driver-service:8080",
    RideServiceURL:    "http://ride-service:8080",
    PaymentServiceURL: "http://payment-service:8080",
    PricingServiceURL: "http://pricing-service:8080",
    SafetyServiceURL:  "http://safety-service:8080",
}, logger)

// Clients are lazily initialized on first use
user, err := f.User().GetByID(ctx, userID)
driver, err := f.Driver().GetByID(ctx, driverID)
```

### Direct Client Usage

```go
import (
    "github.com/Dorico-Dynamics/txova-go-clients/base"
    "github.com/Dorico-Dynamics/txova-go-clients/services/user"
)

baseClient := base.New(base.Config{
    BaseURL:    "http://user-service:8080",
    Timeout:    10 * time.Second,
    MaxRetries: 3,
}, logger)

userClient := user.New(baseClient, logger)
u, err := userClient.GetByID(ctx, userID)
```

### External Integrations

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/sms"

client := sms.New(sms.Config{
    Username: "txova",
    APIKey:   apiKey,
    SenderID: "TXOVA",
}, logger)

err := client.Send(ctx, "+258841234567", "Your code is 123456")
```

## Documentation

For comprehensive usage examples and API documentation, see **[USAGE.md](USAGE.md)**.

## Error Handling

All clients use typed errors from the base package:

| Error | HTTP Status | Description |
|-------|-------------|-------------|
| `ErrNotFound` | 404 | Resource not found |
| `ErrUnauthorized` | 401 | Authentication failed |
| `ErrForbidden` | 403 | Permission denied |
| `ErrBadRequest` | 400 | Invalid request |
| `ErrRateLimited` | 429 | Too many requests |
| `ErrServerError` | 5xx | Server error |
| `ErrServiceUnavailable` | 503 | Service unavailable |
| `ErrTimeout` | - | Request timed out |
| `ErrCircuitOpen` | - | Circuit breaker open |

```go
import "github.com/Dorico-Dynamics/txova-go-clients/base"

user, err := userClient.GetByID(ctx, userID)
if errors.Is(err, base.ErrNotFound) {
    // Handle user not found
}
```

## Circuit Breaker

The base client includes a circuit breaker with three states:

- **Closed** - Normal operation, requests flow through
- **Open** - After threshold failures, requests fail fast
- **Half-Open** - After timeout, allows test request

Default configuration: 5 failures to open, 30 second timeout.

## Dependencies

**Internal:**
- `github.com/Dorico-Dynamics/txova-go-types` - Shared type definitions
- `github.com/Dorico-Dynamics/txova-go-core` - Core utilities and logging
- `github.com/Dorico-Dynamics/txova-go-kafka` - Kafka event publishing

**External:**
- `github.com/minio/minio-go/v7` - MinIO/S3 client
- `firebase.google.com/go/v4` - Firebase SDK
- Africa's Talking Go SDK
- Smile Identity Go SDK

## Development

### Requirements

- Go 1.23+

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

### Linting

```bash
golangci-lint run
```

### Test Coverage Target

> 85%

Current coverage: 87.5%

## License

Proprietary - Dorico Dynamics
