# txova-go-clients

HTTP and external service client library providing typed clients for internal service-to-service communication and third-party API integrations.

## Overview

`txova-go-clients` provides robust HTTP clients for both internal Txova service communication and external third-party integrations, including retry logic, circuit breakers, and proper error handling.

**Module:** `github.com/txova/txova-go-clients`

## Features

- **Base HTTP Client** - Connection pooling, retries, circuit breaker
- **Internal Clients** - Typed clients for all Txova services
- **External Clients** - SMS, email, storage, payments, push notifications
- **Error Handling** - Typed errors with proper status code mapping

## Packages

| Package | Description |
|---------|-------------|
| `base` | Base HTTP client with retry/circuit breaker |
| `internal` | Internal service clients |
| `external` | Third-party API clients |

## Installation

```bash
go get github.com/txova/txova-go-clients
```

## Usage

### Internal Service Clients

```go
import "github.com/txova/txova-go-clients/internal"

// User Service
userClient := internal.NewUserClient(baseURL)
user, err := userClient.GetUser(ctx, userID)
user, err := userClient.GetUserByPhone(ctx, phone)

// Driver Service
driverClient := internal.NewDriverClient(baseURL)
driver, err := driverClient.GetDriver(ctx, driverID)
drivers, err := driverClient.GetNearbyDrivers(ctx, location, 5.0) // 5km radius

// Ride Service
rideClient := internal.NewRideClient(baseURL)
ride, err := rideClient.GetRide(ctx, rideID)
rides, err := rideClient.GetRideHistory(ctx, userID, pagination)

// Payment Service
paymentClient := internal.NewPaymentClient(baseURL)
balance, err := paymentClient.GetWalletBalance(ctx, userID)
err := paymentClient.InitiateRefund(ctx, paymentID, amount, reason)

// Pricing Service
pricingClient := internal.NewPricingClient(baseURL)
estimate, err := pricingClient.GetEstimate(ctx, pickup, dropoff, serviceType)
surge, err := pricingClient.GetSurgeMultiplier(ctx, location)

// Safety Service
safetyClient := internal.NewSafetyClient(baseURL)
rating, err := safetyClient.GetUserRating(ctx, userID)
err := safetyClient.TriggerEmergency(ctx, rideID, location)
```

### External Service Clients

#### SMS (Africa's Talking)

```go
import "github.com/txova/txova-go-clients/external"

sms := external.NewSMSClient(external.SMSConfig{
    Username: "txova",
    APIKey:   apiKey,
    SenderID: "TXOVA",
})

// Send single SMS
err := sms.Send(ctx, "+258841234567", "Your verification code is 123456")

// Send bulk SMS
err := sms.SendBulk(ctx, phones, "Service announcement")

// Check balance
balance, err := sms.GetBalance(ctx)
```

#### Storage (MinIO/S3)

```go
import "github.com/txova/txova-go-clients/external"

storage := external.NewStorageClient(external.StorageConfig{
    Endpoint:  "minio:9000",
    AccessKey: accessKey,
    SecretKey: secretKey,
    Bucket:    "txova-uploads",
})

// Upload file
err := storage.Upload(ctx, "users/123/profile.jpg", reader, "image/jpeg")

// Get presigned URL (1 hour expiry)
url, err := storage.GetPresignedURL(ctx, "users/123/profile.jpg", time.Hour)

// Download file
reader, err := storage.Download(ctx, "users/123/profile.jpg")
```

#### M-Pesa

```go
import "github.com/txova/txova-go-clients/external"

mpesa := external.NewMPesaClient(external.MPesaConfig{
    APIKey:          apiKey,
    PublicKey:       publicKey,
    ServiceProvider: "123456",
    Environment:     "production",
})

// Initiate payment
result, err := mpesa.Initiate(ctx, "+258841234567", money.NewMZN(250, 0), "RIDE-123")

// Query status
status, err := mpesa.Query(ctx, transactionID)

// Initiate refund
err := mpesa.Refund(ctx, transactionID, money.NewMZN(250, 0))
```

#### Push Notifications (Firebase)

```go
import "github.com/txova/txova-go-clients/external"

push := external.NewPushClient(external.PushConfig{
    CredentialsFile: "/secrets/firebase.json",
    ProjectID:       "txova-app",
})

// Send to device
err := push.SendToDevice(ctx, deviceToken, external.Notification{
    Title: "Your ride is arriving",
    Body:  "Driver João is 2 minutes away",
    Data:  map[string]string{"ride_id": rideID.String()},
})

// Send to topic
err := push.SendToTopic(ctx, "maputo-drivers", notification)
```

### Base Client Configuration

```go
import "github.com/txova/txova-go-clients/base"

client := base.New(base.Config{
    BaseURL:      "http://user-service:8080",
    Timeout:      10 * time.Second,
    MaxRetries:   3,
    RetryWait:    100 * time.Millisecond,
    MaxIdleConns: 100,
})

// Retry policy: retries on 5xx, 429, 408, and network errors
// Does not retry on 4xx client errors
```

## Error Types

| Error | Description |
|-------|-------------|
| `ErrServiceUnavailable` | Service not reachable |
| `ErrTimeout` | Request timed out |
| `ErrNotFound` | Resource not found (404) |
| `ErrUnauthorized` | Auth failed (401) |
| `ErrForbidden` | Permission denied (403) |
| `ErrRateLimited` | Too many requests (429) |
| `ErrBadRequest` | Invalid request (400) |
| `ErrServerError` | Server error (5xx) |

```go
import "github.com/txova/txova-go-clients/base"

user, err := userClient.GetUser(ctx, userID)
if errors.Is(err, base.ErrNotFound) {
    // Handle user not found
}
```

## Dependencies

**Internal:**
- `txova-go-types`
- `txova-go-core`
- `txova-go-kafka`

**External:**
- `github.com/minio/minio-go/v7` - MinIO client
- `firebase.google.com/go/v4` - Firebase SDK
- Africa's Talking Go SDK
- Smile Identity Go SDK

## Development

### Requirements

- Go 1.25+

### Testing

```bash
go test ./...
```

### Test Coverage Target

> 85%

## License

Proprietary - Dorico Dynamics
