// Package identity provides a client for Smile Identity verification services.
// Smile Identity is Africa's leading digital identity verification solution.
package identity

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
)

// API environments.
const (
	SandboxBaseURL    = "https://testapi.smileidentity.com/v1"
	ProductionBaseURL = "https://api.smileidentity.com/v1"
)

// Client is the Smile Identity client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	partnerID  string
	apiKey     string
	logger     *logging.Logger
}

// Config holds the configuration for the Smile Identity client.
type Config struct {
	// PartnerID is the Smile Identity partner ID (required).
	PartnerID string

	// APIKey is the Smile Identity API key (required).
	APIKey string

	// Sandbox enables sandbox mode for testing.
	Sandbox bool

	// Timeout is the request timeout (default: 60s).
	Timeout time.Duration
}

// NewClient creates a new Smile Identity client.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.PartnerID == "" {
		return nil, fmt.Errorf("partner ID is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	baseURL := ProductionBaseURL
	if cfg.Sandbox {
		baseURL = SandboxBaseURL
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		partnerID:  cfg.PartnerID,
		apiKey:     cfg.APIKey,
		logger:     logger,
	}, nil
}

// IDType represents the type of ID document.
type IDType string

// Supported ID types for Mozambique.
const (
	IDTypeNationalID      IDType = "NATIONAL_ID"
	IDTypePassport        IDType = "PASSPORT"
	IDTypeDriversLicense  IDType = "DRIVERS_LICENSE"
	IDTypeBirthCertifcate IDType = "BIRTH_CERTIFICATE"
)

// JobType represents the type of verification job.
type JobType int

const (
	// JobTypeBiometricKYC verifies ID and compares selfie to ID photo.
	JobTypeBiometricKYC JobType = 1
	// JobTypeDocumentVerification verifies ID document authenticity.
	JobTypeDocumentVerification JobType = 6
	// JobTypeBasicKYC verifies ID information only.
	JobTypeBasicKYC JobType = 5
	// JobTypeEnhancedKYC queries ID authority for more information.
	JobTypeEnhancedKYC JobType = 5
	// JobTypeSmartSelfieAuthentication compares selfies for authentication.
	JobTypeSmartSelfieAuthentication JobType = 2
)

// VerificationResult represents the result of an ID verification.
type VerificationResult struct {
	// JobID is the unique job identifier.
	JobID string `json:"smile_job_id"`

	// ResultCode is the verification result code.
	ResultCode string `json:"result_code"`

	// ResultText is the human-readable result description.
	ResultText string `json:"result_text"`

	// ConfidenceValue is the confidence score (0-100).
	ConfidenceValue float64 `json:"confidence_value"`

	// PartnerParams contains partner-specific parameters.
	PartnerParams map[string]string `json:"partner_params"`

	// Actions contains recommended actions.
	Actions *Actions `json:"actions"`

	// IDInfo contains verified ID information.
	IDInfo *IDInfo `json:"id_info,omitempty"`
}

// Actions contains recommended actions for a verification result.
type Actions struct {
	// HumanReviewRequired indicates if manual review is needed.
	HumanReviewRequired bool `json:"human_review_required"`
	// ReturnPersonalInfo indicates if personal info should be returned.
	ReturnPersonalInfo bool `json:"return_personal_info"`
	// VerifyIDNumber indicates if ID number was verified.
	VerifyIDNumber bool `json:"verify_id_number"`
}

// IDInfo contains verified ID information.
type IDInfo struct {
	// Country is the ID issuing country.
	Country string `json:"country"`
	// IDType is the type of ID document.
	IDType string `json:"id_type"`
	// IDNumber is the ID number.
	IDNumber string `json:"id_number"`
	// FirstName is the first name from the ID.
	FirstName string `json:"first_name"`
	// LastName is the last name from the ID.
	LastName string `json:"last_name"`
	// MiddleName is the middle name from the ID.
	MiddleName string `json:"middle_name"`
	// DOB is the date of birth from the ID.
	DOB string `json:"dob"`
	// Gender is the gender from the ID.
	Gender string `json:"gender"`
	// Address is the address from the ID.
	Address string `json:"address"`
	// ExpirationDate is the ID expiration date.
	ExpirationDate string `json:"expiration_date"`
}

// FaceMatchResult represents the result of a face comparison.
type FaceMatchResult struct {
	// JobID is the unique job identifier.
	JobID string `json:"smile_job_id"`

	// ResultCode is the verification result code.
	ResultCode string `json:"result_code"`

	// ResultText is the human-readable result description.
	ResultText string `json:"result_text"`

	// ConfidenceValue is the face match confidence score (0-100).
	ConfidenceValue float64 `json:"confidence_value"`

	// Matched indicates if the faces match.
	Matched bool `json:"matched"`
}

// VerificationStatus represents the status of a verification job.
type VerificationStatus struct {
	// JobID is the unique job identifier.
	JobID string `json:"smile_job_id"`

	// JobComplete indicates if the job is finished.
	JobComplete bool `json:"job_complete"`

	// JobSuccess indicates if the job was successful.
	JobSuccess bool `json:"job_success"`

	// ResultCode is the verification result code.
	ResultCode string `json:"result_code"`

	// ResultText is the human-readable result description.
	ResultText string `json:"result_text"`

	// Timestamp is when the status was retrieved.
	Timestamp time.Time `json:"timestamp"`
}

// verifyIDRequest is the request body for ID verification.
type verifyIDRequest struct {
	PartnerID     string            `json:"partner_id"`
	Timestamp     string            `json:"timestamp"`
	Signature     string            `json:"signature"`
	Country       string            `json:"country"`
	IDType        string            `json:"id_type"`
	IDNumber      string            `json:"id_number"`
	FirstName     string            `json:"first_name,omitempty"`
	LastName      string            `json:"last_name,omitempty"`
	DOB           string            `json:"dob,omitempty"`
	PartnerParams map[string]string `json:"partner_params"`
}

// VerifyID verifies an ID document against the ID authority database.
// This is a Basic/Enhanced KYC operation.
func (c *Client) VerifyID(ctx context.Context, idNumber string, idType IDType, country string) (*VerificationResult, error) {
	if idNumber == "" {
		return nil, fmt.Errorf("ID number is required")
	}
	if idType == "" {
		return nil, fmt.Errorf("ID type is required")
	}
	if country == "" {
		country = "MZ" // Default to Mozambique
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := c.generateSignature(timestamp)

	req := verifyIDRequest{
		PartnerID: c.partnerID,
		Timestamp: timestamp,
		Signature: signature,
		Country:   country,
		IDType:    string(idType),
		IDNumber:  idNumber,
		PartnerParams: map[string]string{
			"job_type": "5", // Basic KYC
		},
	}

	var result VerificationResult
	if err := c.doRequest(ctx, "/id_verification", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// biometricRequest is the request body for biometric KYC.
type biometricRequest struct {
	PartnerID     string            `json:"partner_id"`
	Timestamp     string            `json:"timestamp"`
	Signature     string            `json:"signature"`
	Country       string            `json:"country"`
	IDType        string            `json:"id_type"`
	IDNumber      string            `json:"id_number"`
	Images        []imageEntry      `json:"images"`
	PartnerParams map[string]string `json:"partner_params"`
}

// imageEntry represents an image in a verification request.
type imageEntry struct {
	ImageTypeID int    `json:"image_type_id"`
	Image       string `json:"image"` // Base64 encoded
}

// Image type IDs for Smile Identity.
const (
	ImageTypeSelfie      = 0
	ImageTypeIDCardFront = 1
	ImageTypeIDCardBack  = 2
)

// VerifyIDWithPhoto verifies an ID and compares the selfie to the ID photo.
// This is a Biometric KYC operation.
func (c *Client) VerifyIDWithPhoto(ctx context.Context, idNumber string, idType IDType, selfie, idPhoto []byte) (*VerificationResult, error) {
	if idNumber == "" {
		return nil, fmt.Errorf("ID number is required")
	}
	if idType == "" {
		return nil, fmt.Errorf("ID type is required")
	}
	if len(selfie) == 0 {
		return nil, fmt.Errorf("selfie is required")
	}
	if len(idPhoto) == 0 {
		return nil, fmt.Errorf("ID photo is required")
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := c.generateSignature(timestamp)

	req := biometricRequest{
		PartnerID: c.partnerID,
		Timestamp: timestamp,
		Signature: signature,
		Country:   "MZ",
		IDType:    string(idType),
		IDNumber:  idNumber,
		Images: []imageEntry{
			{ImageTypeID: ImageTypeSelfie, Image: base64.StdEncoding.EncodeToString(selfie)},
			{ImageTypeID: ImageTypeIDCardFront, Image: base64.StdEncoding.EncodeToString(idPhoto)},
		},
		PartnerParams: map[string]string{
			"job_type": "1", // Biometric KYC
		},
	}

	var result VerificationResult
	if err := c.doRequest(ctx, "/id_verification", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// faceCompareRequest is the request body for face comparison.
type faceCompareRequest struct {
	PartnerID     string            `json:"partner_id"`
	Timestamp     string            `json:"timestamp"`
	Signature     string            `json:"signature"`
	Images        []imageEntry      `json:"images"`
	PartnerParams map[string]string `json:"partner_params"`
}

// VerifyFace compares two photos to determine if they are the same person.
// This is a SmartSelfie Authentication operation.
func (c *Client) VerifyFace(ctx context.Context, selfie, referencePhoto []byte) (*FaceMatchResult, error) {
	if len(selfie) == 0 {
		return nil, fmt.Errorf("selfie is required")
	}
	if len(referencePhoto) == 0 {
		return nil, fmt.Errorf("reference photo is required")
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := c.generateSignature(timestamp)

	req := faceCompareRequest{
		PartnerID: c.partnerID,
		Timestamp: timestamp,
		Signature: signature,
		Images: []imageEntry{
			{ImageTypeID: ImageTypeSelfie, Image: base64.StdEncoding.EncodeToString(selfie)},
			{ImageTypeID: ImageTypeIDCardFront, Image: base64.StdEncoding.EncodeToString(referencePhoto)},
		},
		PartnerParams: map[string]string{
			"job_type": "2", // SmartSelfie Authentication
		},
	}

	var result FaceMatchResult
	if err := c.doRequest(ctx, "/id_verification", req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// statusRequest is the request body for job status queries.
type statusRequest struct {
	PartnerID string `json:"partner_id"`
	Timestamp string `json:"timestamp"`
	Signature string `json:"signature"`
	JobID     string `json:"job_id"`
	UserID    string `json:"user_id"`
}

// GetVerificationStatus retrieves the status of a verification job.
func (c *Client) GetVerificationStatus(ctx context.Context, jobID, userID string) (*VerificationStatus, error) {
	if jobID == "" {
		return nil, fmt.Errorf("job ID is required")
	}
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	signature := c.generateSignature(timestamp)

	req := statusRequest{
		PartnerID: c.partnerID,
		Timestamp: timestamp,
		Signature: signature,
		JobID:     jobID,
		UserID:    userID,
	}

	var result VerificationStatus
	if err := c.doRequest(ctx, "/job_status", req, &result); err != nil {
		return nil, err
	}

	result.Timestamp = time.Now().UTC()
	return &result, nil
}

// doRequest performs an HTTP POST request to the Smile Identity API.
func (c *Client) doRequest(ctx context.Context, path string, body, result any) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("smile identity API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// generateSignature generates an HMAC-SHA256 signature for API authentication.
func (c *Client) generateSignature(timestamp string) string {
	h := hmac.New(sha256.New, []byte(c.apiKey))
	h.Write([]byte(timestamp + c.partnerID + "sid_request"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Result codes for verification.
const (
	// ResultCodeApproved indicates the verification was approved.
	ResultCodeApproved = "0810"
	// ResultCodeRejected indicates the verification was rejected.
	ResultCodeRejected = "0820"
	// ResultCodePending indicates the verification is still pending.
	ResultCodePending = "0830"
	// ResultCodeIDNotFound indicates the ID was not found.
	ResultCodeIDNotFound = "0840"
	// ResultCodeFaceNotMatched indicates the face did not match.
	ResultCodeFaceNotMatched = "0911"
)

// IsApproved returns true if the verification was approved.
func (r *VerificationResult) IsApproved() bool {
	return r.ResultCode == ResultCodeApproved
}

// IsRejected returns true if the verification was rejected.
func (r *VerificationResult) IsRejected() bool {
	return r.ResultCode == ResultCodeRejected
}

// IsPending returns true if the verification is still pending.
func (r *VerificationResult) IsPending() bool {
	return r.ResultCode == ResultCodePending
}

// IsMatched returns true if the face match was successful.
func (r *FaceMatchResult) IsMatched() bool {
	return r.Matched && r.ConfidenceValue >= 80.0
}

// SetHTTPClient sets a custom HTTP client (useful for testing).
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetBaseURL sets a custom base URL (useful for testing).
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}
