package email

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
	"github.com/Dorico-Dynamics/txova-go-types/contact"
)

func newTestResendClient(t *testing.T) (*ResendClient, *logging.Logger) {
	t.Helper()
	logger := logging.New(logging.Config{Level: slog.LevelDebug})
	client, err := NewResendClient(&ResendConfig{
		APIKey:    "re_test_key",
		FromEmail: "noreply@txova.co.mz",
		FromName:  "Txova",
	}, logger)
	require.NoError(t, err)
	return client, logger
}

func TestNewResendClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ResendConfig
		wantErr string
	}{
		{
			name: "creates client with valid config",
			cfg: &ResendConfig{
				APIKey:    "re_test_key",
				FromEmail: "noreply@example.com",
			},
			wantErr: "",
		},
		{
			name: "creates client with full config",
			cfg: &ResendConfig{
				APIKey:    "re_test_key",
				FromEmail: "noreply@example.com",
				FromName:  "Txova",
			},
			wantErr: "",
		},
		{
			name:    "returns error with nil config",
			cfg:     nil,
			wantErr: "config is required",
		},
		{
			name: "returns error without API key",
			cfg: &ResendConfig{
				FromEmail: "noreply@example.com",
			},
			wantErr: "API key is required",
		},
		{
			name: "returns error without from email",
			cfg: &ResendConfig{
				APIKey: "re_test_key",
			},
			wantErr: "from email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.New(logging.Config{Level: slog.LevelDebug})
			client, err := NewResendClient(tt.cfg, logger)

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

func TestResendSend(t *testing.T) {
	tests := []struct {
		name       string
		to         string
		subject    string
		body       string
		response   ResendSendResult
		statusCode int
		wantErr    string
	}{
		{
			name:    "successful send",
			to:      "user@example.com",
			subject: "Test Subject",
			body:    "Hello World",
			response: ResendSendResult{
				ID: "email_123",
			},
			statusCode: http.StatusOK,
		},
		{
			name:       "handles API error",
			to:         "user@example.com",
			subject:    "Test",
			body:       "Body",
			statusCode: http.StatusBadRequest,
			wantErr:    "resend API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestResendClient(t)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/emails", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")

				// Verify request body
				var req resendEmailRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, tt.subject, req.Subject)

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

			to, _ := contact.ParseEmail(tt.to)
			err := client.Send(context.Background(), to, tt.subject, tt.body)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestResendSendHTML(t *testing.T) {
	client, _ := newTestResendClient(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resendEmailRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "<h1>Hello</h1>", req.HTML)
		assert.Empty(t, req.Text)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ResendSendResult{ID: "email_123"})
	}))
	defer server.Close()
	client.SetBaseURL(server.URL)

	to, _ := contact.ParseEmail("user@example.com")
	err := client.SendHTML(context.Background(), to, "Test", "<h1>Hello</h1>")
	require.NoError(t, err)
}

func TestResendSendToMultiple(t *testing.T) {
	client, _ := newTestResendClient(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req resendEmailRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Len(t, req.To, 2)
		assert.Contains(t, req.To, "user1@example.com")
		assert.Contains(t, req.To, "user2@example.com")

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ResendSendResult{ID: "email_123"})
	}))
	defer server.Close()
	client.SetBaseURL(server.URL)

	to1, _ := contact.ParseEmail("user1@example.com")
	to2, _ := contact.ParseEmail("user2@example.com")
	err := client.SendToMultiple(context.Background(), []contact.Email{to1, to2}, "Test", "Body")
	require.NoError(t, err)
}

func TestResendSendWithOptions(t *testing.T) {
	tests := []struct {
		name       string
		opts       SendOptions
		statusCode int
		wantErr    string
	}{
		{
			name: "successful send with all options",
			opts: SendOptions{
				To:      []contact.Email{mustParseEmail("user@example.com")},
				Subject: "Test Subject",
				Text:    "Hello",
				HTML:    "<p>Hello</p>",
				Cc:      []contact.Email{mustParseEmail("cc@example.com")},
				Bcc:     []contact.Email{mustParseEmail("bcc@example.com")},
				ReplyTo: "reply@example.com",
				Tags:    map[string]string{"category": "test"},
			},
			statusCode: http.StatusOK,
		},
		{
			name: "returns error without recipients",
			opts: SendOptions{
				Subject: "Test",
				Text:    "Body",
			},
			wantErr: "at least one recipient is required",
		},
		{
			name: "returns error without subject",
			opts: SendOptions{
				To:   []contact.Email{mustParseEmail("user@example.com")},
				Text: "Body",
			},
			wantErr: "subject is required",
		},
		{
			name: "returns error without body",
			opts: SendOptions{
				To:      []contact.Email{mustParseEmail("user@example.com")},
				Subject: "Test",
			},
			wantErr: "text or HTML body is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestResendClient(t)

			if tt.statusCode > 0 {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req resendEmailRequest
					err := json.NewDecoder(r.Body).Decode(&req)
					require.NoError(t, err)

					if len(tt.opts.Cc) > 0 {
						assert.NotEmpty(t, req.Cc)
					}
					if len(tt.opts.Bcc) > 0 {
						assert.NotEmpty(t, req.Bcc)
					}
					if tt.opts.ReplyTo != "" {
						assert.Equal(t, tt.opts.ReplyTo, req.ReplyTo)
					}
					if len(tt.opts.Tags) > 0 {
						assert.NotEmpty(t, req.Tags)
					}

					w.WriteHeader(tt.statusCode)
					_ = json.NewEncoder(w).Encode(ResendSendResult{ID: "email_123"})
				}))
				defer server.Close()
				client.SetBaseURL(server.URL)
			}

			result, err := client.SendWithOptions(context.Background(), tt.opts)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result.ID)
			}
		})
	}
}

func TestResendFormatFrom(t *testing.T) {
	t.Run("with name", func(t *testing.T) {
		logger := logging.New(logging.Config{Level: slog.LevelDebug})
		client, err := NewResendClient(&ResendConfig{
			APIKey:    "key",
			FromEmail: "noreply@example.com",
			FromName:  "Txova",
		}, logger)
		require.NoError(t, err)

		from := client.formatFrom()
		assert.Equal(t, "Txova <noreply@example.com>", from)
	})

	t.Run("without name", func(t *testing.T) {
		logger := logging.New(logging.Config{Level: slog.LevelDebug})
		client, err := NewResendClient(&ResendConfig{
			APIKey:    "key",
			FromEmail: "noreply@example.com",
		}, logger)
		require.NoError(t, err)

		from := client.formatFrom()
		assert.Equal(t, "noreply@example.com", from)
	})
}

func mustParseEmail(s string) contact.Email {
	e, err := contact.ParseEmail(s)
	if err != nil {
		panic(err)
	}
	return e
}
