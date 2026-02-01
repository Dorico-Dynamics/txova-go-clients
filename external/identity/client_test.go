package identity

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
)

func newTestClient(t *testing.T) (*Client, *logging.Logger) {
	t.Helper()
	logger := logging.New(logging.Config{Level: slog.LevelDebug})
	client, err := NewClient(&Config{
		PartnerID: "test-partner",
		APIKey:    "test-api-key",
		Sandbox:   true,
	}, logger)
	require.NoError(t, err)
	return client, logger
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name: "creates client with valid config",
			cfg: &Config{
				PartnerID: "partner123",
				APIKey:    "apikey123",
			},
			wantErr: "",
		},
		{
			name: "uses sandbox URL when enabled",
			cfg: &Config{
				PartnerID: "partner123",
				APIKey:    "apikey123",
				Sandbox:   true,
			},
			wantErr: "",
		},
		{
			name:    "returns error with nil config",
			cfg:     nil,
			wantErr: "config is required",
		},
		{
			name: "returns error without partner ID",
			cfg: &Config{
				APIKey: "apikey123",
			},
			wantErr: "partner ID is required",
		},
		{
			name: "returns error without API key",
			cfg: &Config{
				PartnerID: "partner123",
			},
			wantErr: "API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.New(logging.Config{Level: slog.LevelDebug})
			client, err := NewClient(tt.cfg, logger)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestVerifyID(t *testing.T) {
	tests := []struct {
		name       string
		idNumber   string
		idType     IDType
		country    string
		response   VerificationResult
		statusCode int
		wantErr    string
	}{
		{
			name:     "successful verification",
			idNumber: "123456789",
			idType:   IDTypeNationalID,
			country:  "MZ",
			response: VerificationResult{
				JobID:           "job123",
				ResultCode:      ResultCodeApproved,
				ResultText:      "Verified",
				ConfidenceValue: 99.5,
			},
			statusCode: http.StatusOK,
		},
		{
			name:     "returns error for empty ID number",
			idNumber: "",
			idType:   IDTypeNationalID,
			wantErr:  "ID number is required",
		},
		{
			name:     "returns error for empty ID type",
			idNumber: "123456789",
			idType:   "",
			wantErr:  "ID type is required",
		},
		{
			name:       "handles API error",
			idNumber:   "123456789",
			idType:     IDTypeNationalID,
			statusCode: http.StatusBadRequest,
			wantErr:    "smile identity API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestClient(t)

			if tt.statusCode > 0 {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "/id_verification", r.URL.Path)

					w.WriteHeader(tt.statusCode)
					if tt.statusCode == http.StatusOK {
						err := json.NewEncoder(w).Encode(tt.response)
						require.NoError(t, err)
					} else {
						_, _ = w.Write([]byte(`{"error": "bad request"}`))
					}
				}))
				defer server.Close()
				client.SetBaseURL(server.URL)
			}

			result, err := client.VerifyID(context.Background(), tt.idNumber, tt.idType, tt.country)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response.JobID, result.JobID)
				assert.Equal(t, tt.response.ResultCode, result.ResultCode)
			}
		})
	}
}

func TestVerifyIDWithPhoto(t *testing.T) {
	tests := []struct {
		name       string
		idNumber   string
		idType     IDType
		selfie     []byte
		idPhoto    []byte
		response   VerificationResult
		statusCode int
		wantErr    string
	}{
		{
			name:     "successful biometric verification",
			idNumber: "123456789",
			idType:   IDTypeNationalID,
			selfie:   []byte("fake-selfie-data"),
			idPhoto:  []byte("fake-id-photo-data"),
			response: VerificationResult{
				JobID:           "job123",
				ResultCode:      ResultCodeApproved,
				ResultText:      "Face matched",
				ConfidenceValue: 95.0,
			},
			statusCode: http.StatusOK,
		},
		{
			name:    "returns error for empty ID number",
			selfie:  []byte("data"),
			idPhoto: []byte("data"),
			idType:  IDTypeNationalID,
			wantErr: "ID number is required",
		},
		{
			name:     "returns error for empty ID type",
			idNumber: "123456789",
			selfie:   []byte("data"),
			idPhoto:  []byte("data"),
			wantErr:  "ID type is required",
		},
		{
			name:     "returns error for empty selfie",
			idNumber: "123456789",
			idType:   IDTypeNationalID,
			idPhoto:  []byte("data"),
			wantErr:  "selfie is required",
		},
		{
			name:     "returns error for empty ID photo",
			idNumber: "123456789",
			idType:   IDTypeNationalID,
			selfie:   []byte("data"),
			wantErr:  "ID photo is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestClient(t)

			if tt.statusCode > 0 {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					err := json.NewEncoder(w).Encode(tt.response)
					require.NoError(t, err)
				}))
				defer server.Close()
				client.SetBaseURL(server.URL)
			}

			result, err := client.VerifyIDWithPhoto(context.Background(), tt.idNumber, tt.idType, tt.selfie, tt.idPhoto)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response.JobID, result.JobID)
			}
		})
	}
}

func TestVerifyFace(t *testing.T) {
	tests := []struct {
		name       string
		selfie     []byte
		userID     string
		response   FaceMatchResult
		statusCode int
		wantErr    string
	}{
		{
			name:   "successful face match",
			selfie: []byte("selfie-data"),
			userID: "user-123",
			response: FaceMatchResult{
				JobID:           "job123",
				ResultCode:      ResultCodeApproved,
				ConfidenceValue: 92.0,
				Matched:         true,
			},
			statusCode: http.StatusOK,
		},
		{
			name:    "returns error for empty selfie",
			userID:  "user-123",
			wantErr: "selfie is required",
		},
		{
			name:    "returns error for empty user ID",
			selfie:  []byte("data"),
			wantErr: "user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestClient(t)

			if tt.statusCode > 0 {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					err := json.NewEncoder(w).Encode(tt.response)
					require.NoError(t, err)
				}))
				defer server.Close()
				client.SetBaseURL(server.URL)
			}

			result, err := client.VerifyFace(context.Background(), tt.selfie, tt.userID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response.JobID, result.JobID)
				assert.Equal(t, tt.response.Matched, result.Matched)
			}
		})
	}
}

func TestGetVerificationStatus(t *testing.T) {
	tests := []struct {
		name       string
		jobID      string
		userID     string
		response   VerificationStatus
		statusCode int
		wantErr    string
	}{
		{
			name:   "successful status query",
			jobID:  "job123",
			userID: "user456",
			response: VerificationStatus{
				JobID:       "job123",
				JobComplete: true,
				JobSuccess:  true,
				ResultCode:  ResultCodeApproved,
				ResultText:  "Verification complete",
			},
			statusCode: http.StatusOK,
		},
		{
			name:    "returns error for empty job ID",
			userID:  "user456",
			wantErr: "job ID is required",
		},
		{
			name:    "returns error for empty user ID",
			jobID:   "job123",
			wantErr: "user ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestClient(t)

			if tt.statusCode > 0 {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					assert.Equal(t, "/job_status", r.URL.Path)

					w.WriteHeader(tt.statusCode)
					err := json.NewEncoder(w).Encode(tt.response)
					require.NoError(t, err)
				}))
				defer server.Close()
				client.SetBaseURL(server.URL)
			}

			result, err := client.GetVerificationStatus(context.Background(), tt.jobID, tt.userID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.response.JobID, result.JobID)
				assert.Equal(t, tt.response.JobComplete, result.JobComplete)
			}
		})
	}
}

func TestVerificationResultHelpers(t *testing.T) {
	t.Run("IsApproved", func(t *testing.T) {
		approved := &VerificationResult{ResultCode: ResultCodeApproved}
		rejected := &VerificationResult{ResultCode: ResultCodeRejected}

		assert.True(t, approved.IsApproved())
		assert.False(t, rejected.IsApproved())
	})

	t.Run("IsRejected", func(t *testing.T) {
		approved := &VerificationResult{ResultCode: ResultCodeApproved}
		rejected := &VerificationResult{ResultCode: ResultCodeRejected}

		assert.False(t, approved.IsRejected())
		assert.True(t, rejected.IsRejected())
	})

	t.Run("IsPending", func(t *testing.T) {
		pending := &VerificationResult{ResultCode: ResultCodePending}
		approved := &VerificationResult{ResultCode: ResultCodeApproved}

		assert.True(t, pending.IsPending())
		assert.False(t, approved.IsPending())
	})
}

func TestFaceMatchResultHelpers(t *testing.T) {
	t.Run("IsMatched with high confidence", func(t *testing.T) {
		matched := &FaceMatchResult{Matched: true, ConfidenceValue: 95.0}
		assert.True(t, matched.IsMatched())
	})

	t.Run("IsMatched with low confidence", func(t *testing.T) {
		lowConfidence := &FaceMatchResult{Matched: true, ConfidenceValue: 70.0}
		assert.False(t, lowConfidence.IsMatched())
	})

	t.Run("IsMatched when not matched", func(t *testing.T) {
		notMatched := &FaceMatchResult{Matched: false, ConfidenceValue: 95.0}
		assert.False(t, notMatched.IsMatched())
	})
}

func TestGenerateSignature(t *testing.T) {
	client, _ := newTestClient(t)

	// Signature should be deterministic for the same input
	timestamp := "2026-01-01T00:00:00Z"
	sig1 := client.generateSignature(timestamp)
	sig2 := client.generateSignature(timestamp)

	assert.Equal(t, sig1, sig2)
	assert.NotEmpty(t, sig1)
}
