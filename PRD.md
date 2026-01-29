# txova-go-clients

## Overview
HTTP and external service client library providing typed clients for internal service-to-service communication and third-party API integrations.

**Module:** `github.com/txova/txova-go-clients`

---

## Packages

### `base` - Base HTTP Client

#### Features
| Feature | Priority | Description |
|---------|----------|-------------|
| Connection pooling | P0 | Reuse connections |
| Timeout handling | P0 | Configurable timeouts |
| Retry with backoff | P0 | Retry on transient failures |
| Circuit breaker | P1 | Fail fast when service down |
| Request tracing | P0 | Propagate trace context |
| Error handling | P0 | Parse error responses |

#### Configuration
| Setting | Default | Description |
|---------|---------|-------------|
| base_url | - | Service base URL |
| timeout | 10s | Request timeout |
| max_retries | 3 | Retry attempts |
| retry_wait | 100ms | Initial backoff |
| max_idle_conns | 100 | Connection pool |

#### Retry Policy
| Status | Retry? | Description |
|--------|--------|-------------|
| 5xx | Yes | Server errors |
| 429 | Yes | Rate limited |
| 408 | Yes | Timeout |
| 4xx | No | Client errors |
| Network error | Yes | Connection issues |

**Requirements:**
- Propagate X-Request-ID and trace headers
- Parse standard error envelope
- Log all requests at DEBUG level
- Log failures at WARN level
- Support context cancellation

---

### `internal` - Internal Service Clients

#### User Service Client
| Method | Description |
|--------|-------------|
| GetUser(userID) | Get user by ID |
| GetUserByPhone(phone) | Get user by phone |
| VerifyUser(userID) | Mark user verified |
| SuspendUser(userID, reason) | Suspend user |
| GetUserStatus(userID) | Get current status |

#### Driver Service Client
| Method | Description |
|--------|-------------|
| GetDriver(driverID) | Get driver by ID |
| GetDriverByUserID(userID) | Get driver by user |
| GetActiveVehicle(driverID) | Get current vehicle |
| RecordEarnings(driverID, rideID, amount) | Record ride earnings |
| GetDriverStatus(driverID) | Get availability |
| GetNearbyDrivers(location, radius) | Find available drivers |

#### Ride Service Client
| Method | Description |
|--------|-------------|
| GetRide(rideID) | Get ride by ID |
| GetActiveRide(userID) | Get user's active ride |
| GetRideHistory(userID, pagination) | Get ride history |
| CancelRide(rideID, reason) | Cancel ride |

#### Payment Service Client
| Method | Description |
|--------|-------------|
| GetPayment(paymentID) | Get payment by ID |
| GetPaymentByRide(rideID) | Get ride payment |
| InitiateRefund(paymentID, amount, reason) | Start refund |
| GetWalletBalance(userID) | Get wallet balance |

#### Pricing Service Client
| Method | Description |
|--------|-------------|
| GetEstimate(pickup, dropoff, serviceType) | Get fare estimate |
| GetSurgeMultiplier(location) | Get current surge |
| ValidateFare(rideID, fare) | Validate final fare |

#### Safety Service Client
| Method | Description |
|--------|-------------|
| GetUserRating(userID) | Get rating aggregate |
| GetDriverRating(driverID) | Get driver rating |
| ReportIncident(incident) | Create incident report |
| TriggerEmergency(rideID, location) | Activate SOS |

---

#### Client Factory
| Requirement | Description |
|-------------|-------------|
| Configuration | Load URLs from config |
| Singleton | One client per service |
| Lazy init | Initialize on first use |
| Health check | Verify service reachable |

**Service URLs (from config):**
| Service | Config Key |
|---------|------------|
| User | services.user.url |
| Driver | services.driver.url |
| Ride | services.ride.url |
| Payment | services.payment.url |
| Pricing | services.pricing.url |
| Safety | services.safety.url |
| Notification | services.notification.url |

---

### `external` - External Service Clients

#### SMS Client (Africa's Talking)
| Method | Description |
|--------|-------------|
| Send(phone, message) | Send single SMS |
| SendBulk(phones[], message) | Send bulk SMS |
| GetBalance() | Check account balance |
| GetDeliveryStatus(messageID) | Check delivery |

**Configuration:**
| Setting | Description |
|---------|-------------|
| username | AT username |
| api_key | AT API key |
| sender_id | Sender name |
| sandbox | Use sandbox env |

**Requirements:**
- Validate phone format before sending
- Log all SMS sends
- Handle rate limits
- Track delivery status

---

#### Email Client (SendGrid)
| Method | Description |
|--------|-------------|
| Send(to, subject, body) | Send single email |
| SendTemplate(to, templateID, data) | Send templated email |

**Configuration:**
| Setting | Description |
|---------|-------------|
| api_key | SendGrid API key |
| from_email | Sender email |
| from_name | Sender name |

---

#### Storage Client (MinIO/S3)
| Method | Description |
|--------|-------------|
| Upload(key, reader, contentType) | Upload file |
| Download(key) | Get file |
| GetPresignedURL(key, expiry) | Get temporary URL |
| Delete(key) | Remove file |
| Exists(key) | Check if exists |

**Configuration:**
| Setting | Description |
|---------|-------------|
| endpoint | MinIO endpoint |
| access_key | Access key ID |
| secret_key | Secret access key |
| bucket | Default bucket |
| use_ssl | Enable SSL |

**Key Naming Convention:**
| Type | Pattern |
|------|---------|
| Profile photo | users/{user_id}/profile.jpg |
| Driver document | drivers/{driver_id}/documents/{type}.pdf |
| Vehicle photo | vehicles/{vehicle_id}/photos/{n}.jpg |

---

#### Identity Verification Client (Smile Identity)
| Method | Description |
|--------|-------------|
| VerifyID(idNumber, idType, photo) | Verify ID document |
| VerifyFace(selfie, idPhoto) | Compare faces |
| GetVerificationStatus(jobID) | Check status |

**Configuration:**
| Setting | Description |
|---------|-------------|
| partner_id | Partner ID |
| api_key | API key |
| environment | sandbox/production |

---

#### M-Pesa Client
| Method | Description |
|--------|-------------|
| Initiate(phone, amount, reference) | Start payment |
| Query(transactionID) | Check status |
| Refund(transactionID, amount) | Initiate refund |

**Configuration:**
| Setting | Description |
|---------|-------------|
| api_key | M-Pesa API key |
| public_key | Public key for encryption |
| service_provider | Service provider code |
| environment | sandbox/production |

---

#### Push Notification Client (Firebase)
| Method | Description |
|--------|-------------|
| SendToDevice(token, notification) | Send to device |
| SendToTopic(topic, notification) | Send to topic |
| SendToUser(userID, notification) | Send to all user devices |

**Configuration:**
| Setting | Description |
|---------|-------------|
| credentials_file | Service account JSON |
| project_id | Firebase project |

---

## Error Handling

| Error Type | Description |
|------------|-------------|
| ErrServiceUnavailable | Service not reachable |
| ErrTimeout | Request timed out |
| ErrNotFound | Resource not found (404) |
| ErrUnauthorized | Auth failed (401) |
| ErrForbidden | Permission denied (403) |
| ErrRateLimited | Too many requests (429) |
| ErrBadRequest | Invalid request (400) |
| ErrServerError | Server error (5xx) |

**Requirements:**
- Map HTTP status to typed errors
- Include response body in error
- Support errors.Is() checking
- Preserve original error for debugging

---

## Dependencies

**Internal:**
- `txova-go-types`
- `txova-go-core`
- `txova-go-kafka`

**External:**
- `github.com/minio/minio-go/v7` — MinIO client
- `firebase.google.com/go/v4` — Firebase SDK
- Africa's Talking Go SDK
- Smile Identity Go SDK

---

## Success Metrics
| Metric | Target |
|--------|--------|
| Test coverage | > 85% |
| Request latency P99 | < 200ms |
| Retry success rate | > 90% |
| Circuit breaker trips | < 1/hour |
