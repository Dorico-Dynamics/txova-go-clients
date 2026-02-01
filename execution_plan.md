# txova-go-clients Execution Plan

## Overview

Implementation plan for the HTTP and external service client library providing typed clients for internal service-to-service communication and third-party API integrations in the Txova platform.

**Target Coverage:** > 85%

---

## Internal Dependencies

### txova-go-core
| Package | Types/Functions Used | Purpose |
|---------|---------------------|---------|
| `errors` | `AppError`, `Code`, error constructors | Structured error handling for client errors |
| `errors` | `NotFound()`, `ServiceUnavailable()`, `RateLimited()` | Map HTTP responses to typed errors |
| `errors` | `IsCode()`, `IsNotFound()`, `IsServiceUnavailable()` | Error type checking |
| `logging` | `Logger`, `WithContext()` | Request/response logging with context |
| `logging` | `MaskPhone()`, `MaskEmail()`, `SafeAttr()` | PII protection in logs |
| `config` | `Load()`, config structs | Client configuration loading |
| `context` | `RequestID()`, `CorrelationID()`, `WithRequestID()` | Request correlation propagation |
| `context` | `HeaderRequestID`, `HeaderCorrelationID` | HTTP header constants |

### txova-go-types
| Package | Types Used | Purpose |
|---------|-----------|---------|
| `ids` | `UserID`, `DriverID`, `RideID`, `PaymentID`, `VehicleID`, `IncidentID` | Strongly-typed IDs for API requests/responses |
| `enums` | `UserStatus`, `DriverStatus`, `RideStatus`, `PaymentStatus` | Status enums for responses |
| `enums` | `ServiceType`, `PaymentMethod`, `IncidentSeverity` | Request parameter enums |
| `geo` | `Location`, `BoundingBox` | Geographic types for location-based queries |
| `money` | `Money` | Currency amounts in responses |
| `contact` | `PhoneNumber`, `Email` | Contact information types |
| `rating` | `Rating` | Rating values |
| `pagination` | `PageRequest`, `PageResponse`, `CursorRequest`, `CursorResponse` | Paginated API responses |

### txova-go-kafka
| Package | Types/Functions Used | Purpose |
|---------|---------------------|---------|
| `producer` | `Producer`, `Config`, `New()` | Event publishing from clients |
| `envelope` | `Envelope`, `Config`, `NewWithContext()` | Wrap events for publishing |
| `events` | Event type constants, payload structs | Strongly-typed event definitions |

---

## External Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/minio/minio-go/v7` | v7.x | MinIO/S3 storage client |
| `firebase.google.com/go/v4` | v4.x | Firebase push notifications |
| `github.com/AfricasTalkingLtd/africastalking-go` | latest | Africa's Talking SMS |
| `github.com/sendgrid/sendgrid-go` | v3.x | SendGrid email |

---

## Progress Summary

| Phase | Status | Commit | Coverage |
|-------|--------|--------|----------|
| Phase 1: Foundation & Base Client | Pending | - | - |
| Phase 2: Internal Service Clients (User, Driver) | Pending | - | - |
| Phase 3: Internal Service Clients (Ride, Payment, Pricing) | Pending | - | - |
| Phase 4: Internal Service Clients (Safety, Client Factory) | Pending | - | - |
| Phase 5: External Clients (SMS, Email) | Pending | - | - |
| Phase 6: External Clients (Storage, Push Notifications) | Pending | - | - |
| Phase 7: External Clients (M-Pesa, Identity Verification) | Pending | - | - |
| Phase 8: Integration & Documentation | Pending | - | - |

**Current Branch:** `week1`

---

## Phase 1: Foundation & Base Client

### 1.1 Project Setup
- [ ] Initialize Go module with `github.com/Dorico-Dynamics/txova-go-clients`
- [ ] Add internal dependencies:
  - `github.com/Dorico-Dynamics/txova-go-core`
  - `github.com/Dorico-Dynamics/txova-go-types`
  - `github.com/Dorico-Dynamics/txova-go-kafka`
- [ ] Add external dependencies:
  - `github.com/minio/minio-go/v7`
  - `firebase.google.com/go/v4`
  - `github.com/sendgrid/sendgrid-go`
- [ ] Create package structure: `base/`, `internal/`, `external/`, `factory/`
- [ ] Set up `.golangci.yml` for linting

### 1.2 Client Error Types
- [ ] Define client-specific error types using `txova-go-core/errors`:
  - `ErrServiceUnavailable` - Service not reachable
  - `ErrTimeout` - Request timed out
  - `ErrNotFound` - Resource not found (404)
  - `ErrUnauthorized` - Auth failed (401)
  - `ErrForbidden` - Permission denied (403)
  - `ErrRateLimited` - Too many requests (429)
  - `ErrBadRequest` - Invalid request (400)
  - `ErrServerError` - Server error (5xx)
- [ ] Implement `MapHTTPError(statusCode int, body []byte) error`
- [ ] Support `errors.Is()` checking for all error types

### 1.3 Base HTTP Client (`base` package)
- [ ] Define `Client` struct with configuration:
  - `httpClient *http.Client`
  - `baseURL string`
  - `logger *logging.Logger`
  - `retryConfig *RetryConfig`
  - `circuitBreaker *CircuitBreaker` (optional)
- [ ] Implement `Config` struct:
  - `BaseURL string` (required)
  - `Timeout time.Duration` (default: 10s)
  - `MaxRetries int` (default: 3)
  - `RetryWait time.Duration` (default: 100ms)
  - `MaxRetryWait time.Duration` (default: 2s)
  - `MaxIdleConns int` (default: 100)
  - `IdleConnTimeout time.Duration` (default: 90s)
- [ ] Implement `NewClient(cfg *Config, logger *logging.Logger) (*Client, error)`

### 1.4 Connection Pooling
- [ ] Configure `http.Transport` with connection pooling:
  - `MaxIdleConns`
  - `MaxIdleConnsPerHost`
  - `IdleConnTimeout`
- [ ] Support TLS configuration
- [ ] Support custom transport for testing

### 1.5 Request Building
- [ ] Implement request builder methods:
  - `(c *Client) Get(ctx context.Context, path string) (*Request, error)`
  - `(c *Client) Post(ctx context.Context, path string, body any) (*Request, error)`
  - `(c *Client) Put(ctx context.Context, path string, body any) (*Request, error)`
  - `(c *Client) Patch(ctx context.Context, path string, body any) (*Request, error)`
  - `(c *Client) Delete(ctx context.Context, path string) (*Request, error)`
- [ ] Implement `Request` struct with fluent API:
  - `WithHeader(key, value string) *Request`
  - `WithQuery(key, value string) *Request`
  - `WithQueryParams(params url.Values) *Request`
  - `WithBody(body any) *Request`
  - `Do() (*Response, error)`
  - `Decode(dest any) error`

### 1.6 Request/Response Tracing
- [ ] Propagate headers from context:
  - `X-Request-ID` (from `context.RequestID()`)
  - `X-Correlation-ID` (from `context.CorrelationID()`)
- [ ] Add headers to all outgoing requests
- [ ] Log request start at DEBUG level:
  - Method, URL, request_id, correlation_id
- [ ] Log request completion at DEBUG level:
  - Status, duration_ms, request_id
- [ ] Log failures at WARN level with error details

### 1.7 Retry Logic
- [ ] Implement `RetryConfig` struct:
  - `MaxRetries int`
  - `InitialWait time.Duration`
  - `MaxWait time.Duration`
  - `Multiplier float64` (default: 2.0)
  - `Jitter float64` (default: 0.1)
- [ ] Implement exponential backoff with jitter
- [ ] Retry on status codes: 5xx, 429, 408
- [ ] Retry on network errors (connection refused, timeout)
- [ ] Do NOT retry on: 4xx (except 429, 408)
- [ ] Respect `Retry-After` header for 429 responses
- [ ] Support context cancellation during retry

### 1.8 Circuit Breaker (P1)
- [ ] Implement `CircuitBreaker` struct:
  - `FailureThreshold int` (default: 5)
  - `SuccessThreshold int` (default: 2)
  - `Timeout time.Duration` (default: 30s)
- [ ] Implement states: Closed, Open, HalfOpen
- [ ] Track consecutive failures
- [ ] Open circuit after threshold failures
- [ ] Allow probe requests in half-open state
- [ ] Close circuit after success threshold in half-open
- [ ] Return `ErrCircuitOpen` when circuit is open

### 1.9 Response Handling
- [ ] Implement `Response` struct:
  - `StatusCode int`
  - `Headers http.Header`
  - `Body []byte`
- [ ] Implement `(r *Response) Decode(dest any) error`
- [ ] Implement `(r *Response) DecodeError() (*ErrorResponse, error)`
- [ ] Parse standard Txova error envelope on non-2xx
- [ ] Map HTTP status codes to typed errors

### 1.10 Tests
- [ ] Test connection pooling reuses connections
- [ ] Test timeout handling
- [ ] Test retry logic with exponential backoff
- [ ] Test retry respects Retry-After header
- [ ] Test no retry on 4xx errors
- [ ] Test circuit breaker state transitions
- [ ] Test request ID propagation
- [ ] Test correlation ID propagation
- [ ] Test error mapping from HTTP status codes
- [ ] Test context cancellation stops retries

---

## Phase 2: Internal Service Clients (User, Driver)

### 2.1 Service Client Base
- [ ] Implement `ServiceClient` struct embedding `*base.Client`
- [ ] Add service-specific configuration:
  - Service name for logging
  - Default timeout overrides
- [ ] Implement `NewServiceClient(baseClient *base.Client, serviceName string)`

### 2.2 User Service Client (`internal/user`)
- [ ] Implement `Client` struct with methods:
  - `GetUser(ctx, userID ids.UserID) (*types.User, error)`
  - `GetUserByPhone(ctx, phone contact.PhoneNumber) (*types.User, error)`
  - `VerifyUser(ctx, userID ids.UserID) error`
  - `SuspendUser(ctx, userID ids.UserID, reason string) error`
  - `GetUserStatus(ctx, userID ids.UserID) (enums.UserStatus, error)`
- [ ] Define response types using `txova-go-types`:
  - `User` struct with `ids.UserID`, `contact.PhoneNumber`, `contact.Email`, `enums.UserStatus`
- [ ] Implement request/response mapping
- [ ] Add method-specific error handling

### 2.3 Driver Service Client (`internal/driver`)
- [ ] Implement `Client` struct with methods:
  - `GetDriver(ctx, driverID ids.DriverID) (*types.Driver, error)`
  - `GetDriverByUserID(ctx, userID ids.UserID) (*types.Driver, error)`
  - `GetActiveVehicle(ctx, driverID ids.DriverID) (*types.Vehicle, error)`
  - `RecordEarnings(ctx, driverID ids.DriverID, rideID ids.RideID, amount money.Money) error`
  - `GetDriverStatus(ctx, driverID ids.DriverID) (enums.AvailabilityStatus, error)`
  - `GetNearbyDrivers(ctx, location geo.Location, radiusKM float64) ([]*types.NearbyDriver, error)`
- [ ] Define response types:
  - `Driver` struct with `ids.DriverID`, `ids.UserID`, `enums.DriverStatus`, `rating.Rating`
  - `Vehicle` struct with `ids.VehicleID`, `vehicle.LicensePlate`
  - `NearbyDriver` struct with driver info + `geo.Location`, distance

### 2.4 Tests
- [ ] Test User client all methods with mock server
- [ ] Test Driver client all methods with mock server
- [ ] Test error handling (404, 500, timeout)
- [ ] Test request ID propagation
- [ ] Test pagination for list endpoints

---

## Phase 3: Internal Service Clients (Ride, Payment, Pricing)

### 3.1 Ride Service Client (`internal/ride`)
- [ ] Implement `Client` struct with methods:
  - `GetRide(ctx, rideID ids.RideID) (*types.Ride, error)`
  - `GetActiveRide(ctx, userID ids.UserID) (*types.Ride, error)`
  - `GetRideHistory(ctx, userID ids.UserID, page pagination.PageRequest) (*pagination.PageResponse[types.Ride], error)`
  - `CancelRide(ctx, rideID ids.RideID, reason enums.CancellationReason) error`
- [ ] Define response types:
  - `Ride` struct with full ride details using `txova-go-types`

### 3.2 Payment Service Client (`internal/payment`)
- [ ] Implement `Client` struct with methods:
  - `GetPayment(ctx, paymentID ids.PaymentID) (*types.Payment, error)`
  - `GetPaymentByRide(ctx, rideID ids.RideID) (*types.Payment, error)`
  - `InitiateRefund(ctx, paymentID ids.PaymentID, amount money.Money, reason string) (*types.Refund, error)`
  - `GetWalletBalance(ctx, userID ids.UserID) (money.Money, error)`
- [ ] Define response types:
  - `Payment` struct with `ids.PaymentID`, `money.Money`, `enums.PaymentStatus`
  - `Refund` struct

### 3.3 Pricing Service Client (`internal/pricing`)
- [ ] Implement `Client` struct with methods:
  - `GetEstimate(ctx, pickup, dropoff geo.Location, serviceType enums.ServiceType) (*types.FareEstimate, error)`
  - `GetSurgeMultiplier(ctx, location geo.Location) (float64, error)`
  - `ValidateFare(ctx, rideID ids.RideID, fare money.Money) (bool, error)`
- [ ] Define response types:
  - `FareEstimate` struct with `money.Money` range, `time.Duration` estimate

### 3.4 Tests
- [ ] Test Ride client all methods
- [ ] Test Payment client all methods
- [ ] Test Pricing client all methods
- [ ] Test paginated responses with `txova-go-types/pagination`

---

## Phase 4: Internal Service Clients (Safety, Client Factory)

### 4.1 Safety Service Client (`internal/safety`)
- [ ] Implement `Client` struct with methods:
  - `GetUserRating(ctx, userID ids.UserID) (*types.RatingAggregate, error)`
  - `GetDriverRating(ctx, driverID ids.DriverID) (*types.RatingAggregate, error)`
  - `ReportIncident(ctx, incident *types.IncidentReport) (*ids.IncidentID, error)`
  - `TriggerEmergency(ctx, rideID ids.RideID, location geo.Location) error`
- [ ] Define request/response types:
  - `RatingAggregate` struct with `rating.Rating`, count, breakdown
  - `IncidentReport` struct with `enums.IncidentSeverity`

### 4.2 Client Factory (`factory` package)
- [ ] Implement `Factory` struct:
  - Holds configuration for all services
  - Lazy initialization of clients
  - Singleton pattern per service
- [ ] Implement `Config` struct with service URLs:
  - `UserServiceURL string`
  - `DriverServiceURL string`
  - `RideServiceURL string`
  - `PaymentServiceURL string`
  - `PricingServiceURL string`
  - `SafetyServiceURL string`
- [ ] Implement `NewFactory(cfg *Config, logger *logging.Logger) *Factory`
- [ ] Implement getter methods:
  - `(f *Factory) User() *user.Client`
  - `(f *Factory) Driver() *driver.Client`
  - `(f *Factory) Ride() *ride.Client`
  - `(f *Factory) Payment() *payment.Client`
  - `(f *Factory) Pricing() *pricing.Client`
  - `(f *Factory) Safety() *safety.Client`
- [ ] Implement health check:
  - `(f *Factory) HealthCheck(ctx context.Context) map[string]error`

### 4.3 Configuration Loading
- [ ] Support loading URLs from config file using `txova-go-core/config`:
  - `services.user.url`
  - `services.driver.url`
  - `services.ride.url`
  - `services.payment.url`
  - `services.pricing.url`
  - `services.safety.url`
- [ ] Support environment variable overrides

### 4.4 Tests
- [ ] Test Safety client all methods
- [ ] Test Factory lazy initialization
- [ ] Test Factory singleton behavior
- [ ] Test Factory health check aggregation
- [ ] Test configuration loading

---

## Phase 5: External Clients (SMS, Email)

### 5.1 SMS Client - Africa's Talking (`external/sms`)
- [ ] Implement `Client` struct with configuration:
  - `Username string`
  - `APIKey string`
  - `SenderID string`
  - `Sandbox bool`
- [ ] Implement methods:
  - `Send(ctx, phone contact.PhoneNumber, message string) (*SendResult, error)`
  - `SendBulk(ctx, phones []contact.PhoneNumber, message string) ([]*SendResult, error)`
  - `GetBalance(ctx) (*Balance, error)`
  - `GetDeliveryStatus(ctx, messageID string) (*DeliveryStatus, error)`
- [ ] Validate phone format before sending (Mozambique format)
- [ ] Log all SMS sends with masked phone numbers
- [ ] Handle rate limits with retry
- [ ] Track delivery status

### 5.2 Email Client - SendGrid (`external/email`)
- [ ] Implement `Client` struct with configuration:
  - `APIKey string`
  - `FromEmail string`
  - `FromName string`
- [ ] Implement methods:
  - `Send(ctx, to contact.Email, subject, body string) error`
  - `SendHTML(ctx, to contact.Email, subject, htmlBody string) error`
  - `SendTemplate(ctx, to contact.Email, templateID string, data map[string]any) error`
- [ ] Support multiple recipients
- [ ] Log all email sends with masked addresses

### 5.3 Tests
- [ ] Test SMS client with mock AT API
- [ ] Test phone validation
- [ ] Test bulk SMS
- [ ] Test Email client with mock SendGrid API
- [ ] Test template sending

---

## Phase 6: External Clients (Storage, Push Notifications)

### 6.1 Storage Client - MinIO/S3 (`external/storage`)
- [ ] Implement `Client` struct with configuration:
  - `Endpoint string`
  - `AccessKey string`
  - `SecretKey string`
  - `Bucket string`
  - `UseSSL bool`
  - `Region string`
- [ ] Implement methods:
  - `Upload(ctx, key string, reader io.Reader, contentType string) error`
  - `Download(ctx, key string) (io.ReadCloser, error)`
  - `GetPresignedURL(ctx, key string, expiry time.Duration) (string, error)`
  - `Delete(ctx, key string) error`
  - `Exists(ctx, key string) (bool, error)`
  - `List(ctx, prefix string) ([]ObjectInfo, error)`
- [ ] Implement key naming helpers:
  - `ProfilePhotoKey(userID ids.UserID) string` → `users/{user_id}/profile.jpg`
  - `DriverDocumentKey(driverID ids.DriverID, docType string) string` → `drivers/{driver_id}/documents/{type}.pdf`
  - `VehiclePhotoKey(vehicleID ids.VehicleID, index int) string` → `vehicles/{vehicle_id}/photos/{n}.jpg`

### 6.2 Push Notification Client - Firebase (`external/push`)
- [ ] Implement `Client` struct with configuration:
  - `CredentialsFile string` (path to service account JSON)
  - `ProjectID string`
- [ ] Implement methods:
  - `SendToDevice(ctx, token string, notification *Notification) error`
  - `SendToTopic(ctx, topic string, notification *Notification) error`
  - `SendToUser(ctx, userID ids.UserID, notification *Notification) error` (requires token lookup)
  - `SendMulticast(ctx, tokens []string, notification *Notification) (*BatchResult, error)`
- [ ] Define `Notification` struct:
  - `Title string`
  - `Body string`
  - `Data map[string]string`
  - `ImageURL string`
- [ ] Handle token expiration/invalidation

### 6.3 Tests
- [ ] Test Storage client upload/download
- [ ] Test presigned URL generation
- [ ] Test key naming conventions
- [ ] Test Push client with mock Firebase
- [ ] Test multicast delivery

---

## Phase 7: External Clients (M-Pesa, Identity Verification)

### 7.1 M-Pesa Client (`external/mpesa`)
- [ ] Implement `Client` struct with configuration:
  - `APIKey string`
  - `PublicKey string`
  - `ServiceProviderCode string`
  - `Environment string` (sandbox/production)
  - `BaseURL string`
- [ ] Implement methods:
  - `Initiate(ctx, phone contact.PhoneNumber, amount money.Money, reference string) (*InitiateResult, error)`
  - `Query(ctx, transactionID string) (*TransactionStatus, error)`
  - `Refund(ctx, transactionID string, amount money.Money) (*RefundResult, error)`
- [ ] Handle M-Pesa specific error codes
- [ ] Implement callback handling for async notifications
- [ ] Log all transactions (masked sensitive data)

### 7.2 Identity Verification Client - Smile Identity (`external/identity`)
- [ ] Implement `Client` struct with configuration:
  - `PartnerID string`
  - `APIKey string`
  - `Environment string` (sandbox/production)
- [ ] Implement methods:
  - `VerifyID(ctx, idNumber, idType string, photo []byte) (*VerificationResult, error)`
  - `VerifyFace(ctx, selfie, idPhoto []byte) (*FaceMatchResult, error)`
  - `GetVerificationStatus(ctx, jobID string) (*VerificationStatus, error)`
- [ ] Handle async verification flow
- [ ] Support Mozambique ID types

### 7.3 Event Publishing Integration
- [ ] Integrate `txova-go-kafka/producer` for publishing events:
  - Publish `PaymentInitiated` on M-Pesa initiation
  - Publish `PaymentCompleted` / `PaymentFailed` on callback
- [ ] Use `envelope.NewWithContext()` to propagate correlation IDs
- [ ] Use `events` package for strongly-typed event payloads

### 7.4 Tests
- [ ] Test M-Pesa client with mock API
- [ ] Test M-Pesa callback handling
- [ ] Test Identity verification flow
- [ ] Test event publishing integration

---

## Phase 8: Integration & Documentation

### 8.1 Integration Tests
- [ ] Test full flow: User lookup → Ride creation → Payment
- [ ] Test retry behavior across services
- [ ] Test circuit breaker integration
- [ ] Test correlation ID propagation across service calls

### 8.2 Benchmarks
- [ ] Benchmark connection pooling efficiency
- [ ] Benchmark retry overhead
- [ ] Benchmark circuit breaker overhead

### 8.3 Documentation
- [ ] Create comprehensive USAGE.md with examples:
  - Base client configuration
  - Internal service client usage
  - External client setup (SMS, Email, Storage, Push)
  - Payment integration (M-Pesa)
  - Error handling patterns
  - Testing with mocks
- [ ] Update README.md with overview and quick start
- [ ] Document all configuration options

### 8.4 Final Validation
- [ ] Run full test suite: `go test -race -cover ./...`
- [ ] Verify > 85% coverage target
- [ ] Run linting: `golangci-lint run ./...`
- [ ] Run security scan: `gosec ./...`
- [ ] Zero critical issues

---

## Success Criteria

| Metric | Target |
|--------|--------|
| Test coverage | > 85% |
| Request latency P99 | < 200ms (excluding external APIs) |
| Retry success rate | > 90% |
| Circuit breaker trips | < 1/hour |
| Zero critical linting issues | Required |
| All gosec issues resolved | Required |
| All exports documented | Required |

---

## Package Structure

```
txova-go-clients/
├── base/
│   ├── client.go           # Base HTTP client
│   ├── config.go           # Client configuration
│   ├── request.go          # Request builder
│   ├── response.go         # Response handling
│   ├── retry.go            # Retry logic with backoff
│   ├── circuit.go          # Circuit breaker (P1)
│   └── errors.go           # Client error types
├── internal/
│   ├── user/
│   │   └── client.go       # User service client
│   ├── driver/
│   │   └── client.go       # Driver service client
│   ├── ride/
│   │   └── client.go       # Ride service client
│   ├── payment/
│   │   └── client.go       # Payment service client
│   ├── pricing/
│   │   └── client.go       # Pricing service client
│   └── safety/
│       └── client.go       # Safety service client
├── external/
│   ├── sms/
│   │   └── client.go       # Africa's Talking SMS
│   ├── email/
│   │   └── client.go       # SendGrid email
│   ├── storage/
│   │   └── client.go       # MinIO/S3 storage
│   ├── push/
│   │   └── client.go       # Firebase push notifications
│   ├── mpesa/
│   │   └── client.go       # M-Pesa payments
│   └── identity/
│       └── client.go       # Smile Identity verification
├── factory/
│   ├── factory.go          # Client factory
│   └── config.go           # Factory configuration
├── go.mod
├── go.sum
├── README.md
├── USAGE.md
└── execution_plan.md
```

---

## Internal Dependency Usage Map

### txova-go-core Usage

| Core Package | Used In | Purpose |
|--------------|---------|---------|
| `errors` | `base/errors.go` | Define client error types, map HTTP status |
| `errors` | All service clients | Return typed errors from API calls |
| `logging` | `base/client.go` | Request/response logging |
| `logging` | External clients | Log sensitive operations with masking |
| `config` | `factory/config.go` | Load service URLs from config |
| `context` | `base/request.go` | Extract and propagate request/correlation IDs |

### txova-go-types Usage

| Types Package | Used In | Purpose |
|---------------|---------|---------|
| `ids` | All internal clients | Strongly-typed entity IDs |
| `enums` | All internal clients | Status and type enums |
| `geo` | `internal/driver`, `internal/pricing`, `internal/safety` | Location types |
| `money` | `internal/payment`, `internal/pricing`, `external/mpesa` | Currency amounts |
| `contact` | `internal/user`, `external/sms`, `external/email` | Phone/email types |
| `rating` | `internal/safety` | Rating values |
| `pagination` | `internal/ride` | Paginated responses |

### txova-go-kafka Usage

| Kafka Package | Used In | Purpose |
|---------------|---------|---------|
| `producer` | `external/mpesa` | Publish payment events |
| `envelope` | `external/mpesa` | Wrap events with metadata |
| `events` | `external/mpesa` | Payment event types |
