# txova-go-clients Usage Guide

This guide provides comprehensive examples and patterns for using the txova-go-clients library.

## Table of Contents

- [Getting Started](#getting-started)
- [Base HTTP Client](#base-http-client)
- [Service Clients](#service-clients)
- [External Integrations](#external-integrations)
- [Factory Pattern](#factory-pattern)
- [Error Handling](#error-handling)
- [Configuration Reference](#configuration-reference)

---

## Getting Started

### Installation

```bash
go get github.com/Dorico-Dynamics/txova-go-clients
```

### Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/Dorico-Dynamics/txova-go-clients/factory"
    "github.com/Dorico-Dynamics/txova-go-clients/base"
    "github.com/Dorico-Dynamics/txova-go-core/logging"
)

func main() {
    logger := logging.New(logging.Config{Level: slog.LevelInfo})

    // Create factory with service URLs
    f, err := factory.New(&factory.Config{
        UserServiceURL:   "http://user-service:3000",
        DriverServiceURL: "http://driver-service:3001",
        RideServiceURL:   "http://ride-service:3002",
        Retry:            base.DefaultRetryConfig(),
    }, logger)
    if err != nil {
        log.Fatal(err)
    }

    // Get a service client
    userClient, err := f.User()
    if err != nil {
        log.Fatal(err)
    }

    // Make API calls
    ctx := context.Background()
    user, err := userClient.GetUser(ctx, userID)
    if err != nil {
        log.Fatal(err)
    }
}
```

---

## Base HTTP Client

The base client provides enterprise-grade HTTP communication with retry logic, circuit breakers, and request tracing.

### Creating a Base Client

```go
import (
    "github.com/Dorico-Dynamics/txova-go-clients/base"
    "github.com/Dorico-Dynamics/txova-go-core/logging"
)

cfg := &base.Config{
    BaseURL:        "http://api.example.com",
    Timeout:        30 * time.Second,
    RequestTimeout: 10 * time.Second,
    MaxIdleConns:   100,
    Retry: base.RetryConfig{
        MaxRetries:  3,
        InitialWait: 100 * time.Millisecond,
        MaxWait:     2 * time.Second,
        Multiplier:  2.0,
        Jitter:      0.1,
    },
    CircuitBreaker: &base.CircuitBreakerConfig{
        FailureThreshold: 5,
        SuccessThreshold: 2,
        Timeout:          30 * time.Second,
        Name:             "my-service",
    },
}

client, err := base.NewClient(cfg, logger)
```

### Making Requests

#### GET Request

```go
var user User
err := client.Get(ctx, "/users/123").Decode(&user)
```

#### GET with Query Parameters

```go
var rides []Ride
err := client.Get(ctx, "/rides").
    WithQuery("status", "completed").
    WithQuery("limit", "20").
    WithQuery("offset", "0").
    Decode(&rides)
```

#### POST Request

```go
payload := CreateUserRequest{
    Phone:     "+258841234567",
    FirstName: "João",
    LastName:  "Silva",
}

var user User
err := client.Post(ctx, "/users", payload).Decode(&user)
```

#### PUT/PATCH Request

```go
update := UpdateUserRequest{
    FirstName: "João",
    LastName:  "Santos",
}

err := client.Put(ctx, "/users/123", update).Do()
// or
err := client.Patch(ctx, "/users/123", update).Do()
```

#### DELETE Request

```go
_, err := client.Delete(ctx, "/users/123").Do()
```

#### Custom Headers

```go
resp, err := client.Post(ctx, "/payments", payment).
    WithHeader("X-Idempotency-Key", uuid.New().String()).
    WithHeader("X-Request-Source", "mobile-app").
    Do()
```

### Response Handling

```go
resp, err := client.Get(ctx, "/users/123").Do()
if err != nil {
    return err // Network error, timeout, etc.
}

// Check response status
if resp.IsSuccess() {
    var user User
    err := resp.Decode(&user)
    return user, err
}

if resp.IsClientError() {
    // 4xx error - client issue
    return nil, resp.DecodeError()
}

if resp.IsServerError() {
    // 5xx error - server issue
    return nil, resp.DecodeError()
}
```

### Circuit Breaker Monitoring

```go
stats := client.CircuitBreakerStats()
if stats != nil {
    log.Printf("Circuit Breaker State: %s", stats.State.String())
    log.Printf("Consecutive Failures: %d", stats.ConsecutiveFailures)
    log.Printf("Consecutive Successes: %d", stats.ConsecutiveSuccesses)
    log.Printf("Total Requests: %d", stats.TotalRequests)
    log.Printf("Failed Requests: %d", stats.FailedRequests)
}
```

---

## Service Clients

All internal service clients follow a consistent pattern with typed methods and responses.

### User Service

```go
import "github.com/Dorico-Dynamics/txova-go-clients/services/user"

cfg := &user.Config{
    BaseURL: "http://user-service:3000",
    Timeout: 10 * time.Second,
    Retry:   base.DefaultRetryConfig(),
}

client, err := user.NewClient(cfg, logger)

// Get user by ID
user, err := client.GetUser(ctx, userID)

// Get user by phone number
user, err := client.GetUserByPhone(ctx, phone)

// Verify user (KYC completed)
err := client.VerifyUser(ctx, userID)

// Suspend user
err := client.SuspendUser(ctx, userID, "Fraudulent activity detected")

// Get user status
status, err := client.GetUserStatus(ctx, userID)

// Health check
err := client.HealthCheck(ctx)
```

### Driver Service

```go
import "github.com/Dorico-Dynamics/txova-go-clients/services/driver"

client, err := driver.NewClient(&driver.Config{
    BaseURL: "http://driver-service:3001",
}, logger)

// Get driver by ID
driver, err := client.GetDriver(ctx, driverID)

// Get driver by user ID
driver, err := client.GetDriverByUserID(ctx, userID)

// Get active vehicle
vehicle, err := client.GetActiveVehicle(ctx, driverID)

// Record earnings
err := client.RecordEarnings(ctx, driverID, rideID, amount)

// Get driver availability status
status, err := client.GetDriverStatus(ctx, driverID)

// Find nearby drivers (within 5km)
drivers, err := client.GetNearbyDrivers(ctx, location, 5.0)

// Update driver location
err := client.UpdateLocation(ctx, driverID, location)

// Set availability
err := client.SetAvailability(ctx, driverID, enums.AvailabilityStatusOnline)
```

### Ride Service

```go
import "github.com/Dorico-Dynamics/txova-go-clients/services/ride"

client, err := ride.NewClient(&ride.Config{
    BaseURL: "http://ride-service:3002",
}, logger)

// Get ride by ID
ride, err := client.GetRide(ctx, rideID)

// Get user's active ride
ride, err := client.GetActiveRide(ctx, userID)

// Get ride history with pagination
page := pagination.PageRequest{Limit: 20, Offset: 0}
history, err := client.GetRideHistory(ctx, userID, page)

for _, ride := range history.Data {
    fmt.Printf("Ride: %s, Status: %s\n", ride.ID, ride.Status)
}

// Check if more pages available
if history.HasMore {
    // Fetch next page
    page.Offset += page.Limit
    nextPage, _ := client.GetRideHistory(ctx, userID, page)
}

// Cancel ride
err := client.CancelRide(ctx, rideID, enums.CancellationReasonRiderCancelled)

// Get ride status
status, err := client.GetRideStatus(ctx, rideID)
```

### Payment Service

```go
import "github.com/Dorico-Dynamics/txova-go-clients/services/payment"

client, err := payment.NewClient(&payment.Config{
    BaseURL: "http://payment-service:3003",
    Timeout: 15 * time.Second, // Longer timeout for payments
}, logger)

// Get payment by ID
payment, err := client.GetPayment(ctx, paymentID)

// Get payment by ride
payment, err := client.GetPaymentByRide(ctx, rideID)

// Initiate refund
refund, err := client.InitiateRefund(ctx, paymentID, amount, "Customer request")

// Get wallet balance
balance, err := client.GetWalletBalance(ctx, userID)

// Get payment status
status, err := client.GetPaymentStatus(ctx, paymentID)
```

### Pricing Service

```go
import "github.com/Dorico-Dynamics/txova-go-clients/services/pricing"

client, err := pricing.NewClient(&pricing.Config{
    BaseURL: "http://pricing-service:3004",
    Timeout: 5 * time.Second, // Fast lookups
}, logger)

// Get fare estimate
estimate, err := client.GetEstimate(ctx, pickupLocation, dropoffLocation, enums.ServiceTypeStandard)

fmt.Printf("Estimated fare: %s - %s\n", estimate.MinFare, estimate.MaxFare)
fmt.Printf("Distance: %.2f km\n", estimate.DistanceKM)
fmt.Printf("Duration: %d minutes\n", estimate.DurationMinutes)
fmt.Printf("Surge: %.2fx\n", estimate.SurgeMultiplier)

// Get surge multiplier for a location
surge, err := client.GetSurgeMultiplier(ctx, location)
if surge.Multiplier > 1.0 {
    fmt.Printf("Surge active: %.2fx - %s\n", surge.Multiplier, surge.Reason)
}

// Validate final fare
validation, err := client.ValidateFare(ctx, rideID, actualFare)
if !validation.Valid {
    fmt.Printf("Fare validation failed: %s\n", validation.Reason)
}

// Get all service types with pricing
serviceTypes, err := client.GetServiceTypes(ctx)
for _, st := range serviceTypes {
    fmt.Printf("%s: Base %s, Per KM %s\n", st.DisplayName, st.BaseFare, st.PerKMRate)
}
```

### Safety Service

```go
import "github.com/Dorico-Dynamics/txova-go-clients/services/safety"

client, err := safety.NewClient(&safety.Config{
    BaseURL: "http://safety-service:3005",
}, logger)

// Get user rating
rating, err := client.GetUserRating(ctx, userID)
fmt.Printf("Average: %.1f (%d ratings)\n", rating.AverageRating, rating.TotalRatings)

// Get driver rating
rating, err := client.GetDriverRating(ctx, driverID)

// Report incident
report := &safety.IncidentReport{
    RideID:      rideID,
    ReporterID:  userID,
    Severity:    enums.IncidentSeverityHigh,
    Type:        "unsafe_driving",
    Description: "Driver was speeding and running red lights",
    Location:    &location,
}
incident, err := client.ReportIncident(ctx, report)

// Get incident details
incident, err := client.GetIncident(ctx, incidentID)

// Trigger emergency (SOS)
err := client.TriggerEmergency(ctx, rideID, currentLocation)
```

---

## External Integrations

### SMS (Africa's Talking)

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/sms"

client, err := sms.NewClient(&sms.Config{
    Username: "txova",
    APIKey:   os.Getenv("AFRICASTALKING_API_KEY"),
    SenderID: "TXOVA",
    Sandbox:  false,
}, logger)

// Send single SMS
result, err := client.Send(ctx, phone, "Your verification code is 123456")
fmt.Printf("Message ID: %s, Status: %s, Cost: %s\n", 
    result.MessageID, result.Status, result.Cost)

// Send bulk SMS
phones := []contact.PhoneNumber{phone1, phone2, phone3}
results, err := client.SendBulk(ctx, phones, "Service announcement: Maintenance at 2am")

// Check account balance
balance, err := client.GetBalance(ctx)
fmt.Printf("Balance: %s\n", balance.Value)

// Parse delivery callback (webhook)
callback, err := sms.ParseDeliveryCallback(requestBody)
// or from form data
callback, err := sms.ParseDeliveryCallbackForm(httpRequest)
```

### Email (SendGrid)

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/email"

client, err := email.NewClient(&email.Config{
    APIKey:    os.Getenv("SENDGRID_API_KEY"),
    FromEmail: "noreply@txova.co.mz",
    FromName:  "Txova",
}, logger)

// Send plain text email
err := client.Send(ctx, recipientEmail, "Welcome to Txova", "Thank you for joining!")

// Send HTML email
htmlBody := "<h1>Welcome!</h1><p>Thank you for joining Txova.</p>"
err := client.SendHTML(ctx, recipientEmail, "Welcome to Txova", htmlBody)

// Send templated email
data := map[string]string{
    "first_name": "João",
    "ride_id":    "RIDE-123",
    "fare":       "250 MZN",
}
err := client.SendTemplate(ctx, recipientEmail, "d-abc123template", data)

// Send to multiple recipients
recipients := []contact.Email{email1, email2, email3}
err := client.SendToMultiple(ctx, recipients, "Announcement", "Important update...")
```

### Email (Resend) - Alternative Provider

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/email"

client, err := email.NewResendClient(&email.ResendConfig{
    APIKey:    os.Getenv("RESEND_API_KEY"),
    FromEmail: "noreply@txova.co.mz",
    FromName:  "Txova",
}, logger)

// Basic send
err := client.Send(ctx, recipientEmail, "Subject", "Body")

// Send with all options
result, err := client.SendWithOptions(ctx, email.SendOptions{
    To:      []contact.Email{email1, email2},
    Subject: "Important Update",
    HTML:    "<h1>Hello</h1>",
    Text:    "Hello (plain text fallback)",
    Cc:      []contact.Email{ccEmail},
    Bcc:     []contact.Email{bccEmail},
    ReplyTo: "support@txova.co.mz",
    Tags:    map[string]string{"category": "transactional"},
})
```

### Email (SMTP)

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/email"

client, err := email.NewSMTPClient(&email.SMTPConfig{
    Host:      "smtp.example.com",
    Port:      587,
    Username:  "user@example.com",
    Password:  os.Getenv("SMTP_PASSWORD"),
    FromEmail: "noreply@txova.co.mz",
    FromName:  "Txova",
    UseTLS:    true,
}, logger)

err := client.Send(ctx, recipientEmail, "Subject", "Body")
err := client.SendHTML(ctx, recipientEmail, "Subject", "<h1>Hello</h1>")
```

### M-Pesa Payments

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/mpesa"

client, err := mpesa.NewClient(&mpesa.Config{
    APIKey:              os.Getenv("MPESA_API_KEY"),
    PublicKey:           os.Getenv("MPESA_PUBLIC_KEY"),
    ServiceProviderCode: "171717",
    Sandbox:             false,
    Timeout:             60 * time.Second,
}, logger)

// Initiate C2B payment
result, err := client.Initiate(ctx, phone, amount, "RIDE-123")
if result.IsSuccess() {
    fmt.Printf("Transaction ID: %s\n", result.TransactionID)
}

// Query transaction status
status, err := client.Query(ctx, result.TransactionID, "RIDE-123")

// Initiate refund
refund, err := client.Refund(ctx, transactionID, amount, "RIDE-123")

// Parse callback (webhook)
callback, err := mpesa.ParseCallback(requestBody)

// Response code descriptions
desc := mpesa.ResponseCodeDescription("INS-0") // "Request processed successfully"
desc := mpesa.ResponseCodeDescription("INS-9") // "Request timeout"
```

### M-Pesa with Kafka Event Publishing

```go
import (
    "github.com/Dorico-Dynamics/txova-go-clients/external/mpesa"
    "github.com/Dorico-Dynamics/txova-go-kafka/producer"
)

// Create Kafka producer
kafkaProducer, err := producer.New(&producer.Config{
    Brokers: []string{"kafka:9092"},
}, logger)

// Create M-Pesa client with event publishing
client, err := mpesa.NewClient(&mpesa.Config{
    APIKey:              apiKey,
    PublicKey:           publicKey,
    ServiceProviderCode: "171717",
    Producer:            kafkaProducer, // Enable event publishing
}, logger)

// Initiate with automatic event publishing
// Publishes PaymentInitiated event
result, err := client.InitiateWithEvent(ctx, paymentID, rideID, phone, amount, "REF-123")

// Handle callback with automatic event publishing
// Publishes PaymentCompleted or PaymentFailed event
callback, _ := mpesa.ParseCallback(body)
err := client.HandleCallback(ctx, paymentID, callback)
```

### Identity Verification (Smile Identity)

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/identity"

client, err := identity.NewClient(&identity.Config{
    PartnerID: os.Getenv("SMILE_PARTNER_ID"),
    APIKey:    os.Getenv("SMILE_API_KEY"),
    Sandbox:   false,
}, logger)

// Basic ID verification
result, err := client.VerifyID(ctx, "12345678", identity.IDTypeNationalID, "MZ")
if result.IsApproved() {
    fmt.Printf("ID verified for: %s %s\n", result.IDInfo.FirstName, result.IDInfo.LastName)
}

// Biometric verification with photo
selfieBytes, _ := os.ReadFile("selfie.jpg")
idPhotoBytes, _ := os.ReadFile("id_photo.jpg")

result, err := client.VerifyIDWithPhoto(ctx, 
    "12345678",
    identity.IDTypeNationalID,
    selfieBytes,
    idPhotoBytes,
)

// Face matching only (SmartSelfie)
match, err := client.VerifyFace(ctx, selfieBytes, referencePhotoBytes)
if match.Matched {
    fmt.Printf("Face matched with %.2f%% confidence\n", match.ConfidenceValue)
}

// Check verification status
status, err := client.GetVerificationStatus(ctx, jobID, userID)
```

### Push Notifications (Firebase)

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/push"

client, err := push.NewClient(&push.Config{
    ProjectID:   "txova-app",
    AccessToken: firebaseToken,
}, logger)

// Send to single device
result, err := client.SendToDevice(ctx, deviceToken, 
    &push.Notification{
        Title: "Your ride is arriving",
        Body:  "Driver João is 2 minutes away",
    },
    map[string]string{
        "ride_id": rideID.String(),
        "action":  "driver_arriving",
    },
)

// Send to topic
result, err := client.SendToTopic(ctx, "maputo-promotions",
    &push.Notification{
        Title: "50% off your next ride!",
        Body:  "Use code MAPUTO50",
    },
    nil,
)

// Send to multiple devices
results, err := client.SendMulticast(ctx, 
    []string{token1, token2, token3},
    notification,
    data,
)
fmt.Printf("Sent: %d, Failed: %d\n", results.SuccessCount, results.FailureCount)

// Convenience notification builders
notification := push.NewRideNotification("123 Main Street")
notification := push.RideAcceptedNotification("João", 5)
notification := push.DriverArrivedNotification("João")
notification := push.RideCompletedNotification("250 MZN")
notification := push.PaymentReceivedNotification("250 MZN")
```

### Object Storage (MinIO/S3)

```go
import "github.com/Dorico-Dynamics/txova-go-clients/external/storage"

client, err := storage.NewClient(&storage.Config{
    Endpoint:  "minio.txova.local:9000",
    AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
    SecretKey: os.Getenv("MINIO_SECRET_KEY"),
    Bucket:    "txova-uploads",
    UseSSL:    true,
}, logger)

// Upload file
file, _ := os.Open("photo.jpg")
defer file.Close()
stat, _ := file.Stat()

key := storage.ProfilePhotoKey(userID) // "users/{userID}/profile.jpg"
err := client.Upload(ctx, key, file, stat.Size(), "image/jpeg")

// Download file
reader, err := client.Download(ctx, key)
defer reader.Close()
data, _ := io.ReadAll(reader)

// Generate presigned URL (valid for 1 hour)
url, err := client.GetPresignedURL(ctx, key, 1*time.Hour)

// Check if file exists
exists, err := client.Exists(ctx, key)

// Delete file
err := client.Delete(ctx, key)

// List files with prefix
objects, err := client.List(ctx, "users/123/")
for _, obj := range objects {
    fmt.Printf("%s (%d bytes)\n", obj.Key, obj.Size)
}

// Key builders for consistent naming
profileKey := storage.ProfilePhotoKey(userID)              // users/{id}/profile.jpg
docKey := storage.DriverDocumentKey(driverID, "license")   // drivers/{id}/documents/license.pdf
vehicleKey := storage.VehiclePhotoKey(vehicleID, 0)        // vehicles/{id}/photos/0.jpg
receiptKey := storage.RideReceiptKey(rideID)               // rides/{id}/receipt.pdf
```

---

## Factory Pattern

The factory provides centralized configuration and lazy initialization of service clients.

### Basic Usage

```go
import "github.com/Dorico-Dynamics/txova-go-clients/factory"

cfg := &factory.Config{
    UserServiceURL:    "http://user-service:3000",
    DriverServiceURL:  "http://driver-service:3001",
    RideServiceURL:    "http://ride-service:3002",
    PaymentServiceURL: "http://payment-service:3003",
    PricingServiceURL: "http://pricing-service:3004",
    SafetyServiceURL:  "http://safety-service:3005",
    
    // Shared configuration
    Retry: base.RetryConfig{
        MaxRetries:  3,
        InitialWait: 100 * time.Millisecond,
        MaxWait:     2 * time.Second,
    },
    CircuitBreaker: &base.CircuitBreakerConfig{
        FailureThreshold: 5,
        SuccessThreshold: 2,
        Timeout:          30 * time.Second,
    },
}

f, err := factory.New(cfg, logger)
if err != nil {
    log.Fatal(err)
}

// Get clients (created lazily on first access)
userClient, err := f.User()
driverClient, err := f.Driver()
rideClient, err := f.Ride()
paymentClient, err := f.Payment()
pricingClient, err := f.Pricing()
safetyClient, err := f.Safety()
```

### Health Checking

```go
// Check all services
health := f.HealthCheck(ctx)
for _, svc := range health {
    if svc.Healthy {
        fmt.Printf("%s: OK\n", svc.Name)
    } else {
        fmt.Printf("%s: FAILED - %s\n", svc.Name, svc.Error)
    }
}

// Quick check if all healthy
if f.AllHealthy(ctx) {
    fmt.Println("All services healthy")
} else {
    fmt.Println("Some services unhealthy")
}
```

### Optional Services

Services with empty URLs are skipped:

```go
cfg := &factory.Config{
    UserServiceURL:   "http://user-service:3000",
    // DriverServiceURL not set - driver client will return error
}

f, _ := factory.New(cfg, logger)
userClient, _ := f.User()      // Works
driverClient, err := f.Driver() // Returns error: "driver service URL not configured"
```

---

## Error Handling

### Error Types

```go
import "github.com/Dorico-Dynamics/txova-go-clients/base"

// Check specific error types
if base.IsTimeout(err) {
    // Request timed out
}

if base.IsCircuitOpen(err) {
    // Circuit breaker is open
}

if base.IsBadGateway(err) {
    // Invalid response from upstream
}

if base.IsRetryable(err) {
    // Error is retryable (but retries exhausted)
}
```

### HTTP Status Code Handling

Errors from HTTP responses are automatically mapped:

```go
user, err := userClient.GetUser(ctx, userID)
if err != nil {
    var appErr *errors.AppError
    if errors.As(err, &appErr) {
        switch appErr.HTTPCode() {
        case 404:
            // User not found
        case 401:
            // Unauthorized
        case 403:
            // Forbidden
        case 429:
            // Rate limited
        case 500, 502, 503, 504:
            // Server error
        }
    }
}
```

### Error Wrapping Pattern

```go
user, err := userClient.GetUser(ctx, userID)
if err != nil {
    return nil, fmt.Errorf("failed to get user %s: %w", userID, err)
}
```

---

## Configuration Reference

### Base Client Defaults

| Setting | Default | Description |
|---------|---------|-------------|
| Timeout | 30s | Overall connection timeout |
| RequestTimeout | 10s | Per-request timeout |
| MaxIdleConns | 100 | Max idle connections |
| MaxIdleConnsPerHost | 10 | Max idle connections per host |
| IdleConnTimeout | 90s | Idle connection timeout |

### Retry Defaults

| Setting | Default | Description |
|---------|---------|-------------|
| MaxRetries | 3 | Maximum retry attempts |
| InitialWait | 100ms | Initial backoff wait |
| MaxWait | 2s | Maximum backoff wait |
| Multiplier | 2.0 | Backoff multiplier |
| Jitter | 0.1 | Jitter factor (10%) |

### Circuit Breaker Defaults

| Setting | Default | Description |
|---------|---------|-------------|
| FailureThreshold | 5 | Failures to open circuit |
| SuccessThreshold | 2 | Successes to close circuit |
| Timeout | 30s | Time before half-open |

### Service-Specific Timeouts

| Service | Timeout | Reason |
|---------|---------|--------|
| User, Driver, Ride, Safety | 10s | Standard operations |
| Payment | 15s | Payment processing |
| Pricing | 5s | Fast lookups |
| Email, SMS | 30s | External API calls |
| M-Pesa, Identity | 60s | Slow external services |

### Retryable Status Codes

The following HTTP status codes trigger automatic retries:

- `408` Request Timeout
- `429` Too Many Requests
- `500` Internal Server Error
- `502` Bad Gateway
- `503` Service Unavailable
- `504` Gateway Timeout

---

## Best Practices

1. **Use the Factory** for service clients to ensure consistent configuration
2. **Set Appropriate Timeouts** based on expected operation duration
3. **Enable Circuit Breakers** in production to prevent cascade failures
4. **Handle Errors Properly** using the typed error checking functions
5. **Use Context** for cancellation and deadline propagation
6. **Monitor Circuit Breaker Stats** to detect service degradation
7. **Use Key Builders** for storage to ensure consistent naming conventions
8. **Configure Retries Carefully** to balance reliability with latency
