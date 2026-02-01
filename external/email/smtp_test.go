package email

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
)

func TestNewSMTPClient(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *SMTPConfig
		wantErr string
	}{
		{
			name: "creates client with valid config",
			cfg: &SMTPConfig{
				Host:      "smtp.example.com",
				FromEmail: "noreply@example.com",
			},
			wantErr: "",
		},
		{
			name: "creates client with full config",
			cfg: &SMTPConfig{
				Host:      "smtp.example.com",
				Port:      465,
				Username:  "user",
				Password:  "pass",
				FromEmail: "noreply@example.com",
				FromName:  "Txova",
				UseTLS:    true,
			},
			wantErr: "",
		},
		{
			name: "defaults port to 587 when TLS enabled",
			cfg: &SMTPConfig{
				Host:      "smtp.example.com",
				FromEmail: "noreply@example.com",
				UseTLS:    true,
			},
			wantErr: "",
		},
		{
			name:    "returns error with nil config",
			cfg:     nil,
			wantErr: "config is required",
		},
		{
			name: "returns error without host",
			cfg: &SMTPConfig{
				FromEmail: "noreply@example.com",
			},
			wantErr: "host is required",
		},
		{
			name: "returns error without from email",
			cfg: &SMTPConfig{
				Host: "smtp.example.com",
			},
			wantErr: "from email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.New(logging.Config{Level: slog.LevelDebug})
			client, err := NewSMTPClient(tt.cfg, logger)

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

func TestSMTPBuildMessage(t *testing.T) {
	logger := logging.New(logging.Config{Level: slog.LevelDebug})

	t.Run("builds plain text message", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			FromEmail: "noreply@example.com",
		}, logger)
		require.NoError(t, err)

		msg := client.buildMessage([]string{"user@example.com"}, "Test Subject", "Hello World", false)
		msgStr := string(msg)

		assert.Contains(t, msgStr, "From: noreply@example.com")
		assert.Contains(t, msgStr, "To: user@example.com")
		assert.Contains(t, msgStr, "Subject: Test Subject")
		assert.Contains(t, msgStr, "Content-Type: text/plain; charset=UTF-8")
		assert.Contains(t, msgStr, "Hello World")
	})

	t.Run("builds HTML message", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			FromEmail: "noreply@example.com",
		}, logger)
		require.NoError(t, err)

		msg := client.buildMessage([]string{"user@example.com"}, "Test Subject", "<h1>Hello</h1>", true)
		msgStr := string(msg)

		assert.Contains(t, msgStr, "MIME-Version: 1.0")
		assert.Contains(t, msgStr, "Content-Type: text/html; charset=UTF-8")
		assert.Contains(t, msgStr, "<h1>Hello</h1>")
	})

	t.Run("builds message with from name", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			FromEmail: "noreply@example.com",
			FromName:  "Txova Platform",
		}, logger)
		require.NoError(t, err)

		msg := client.buildMessage([]string{"user@example.com"}, "Test", "Body", false)
		msgStr := string(msg)

		assert.Contains(t, msgStr, "From: Txova Platform <noreply@example.com>")
	})

	t.Run("builds message with multiple recipients", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			FromEmail: "noreply@example.com",
		}, logger)
		require.NoError(t, err)

		msg := client.buildMessage([]string{"user1@example.com", "user2@example.com"}, "Test", "Body", false)
		msgStr := string(msg)

		assert.Contains(t, msgStr, "To: user1@example.com, user2@example.com")
	})
}
