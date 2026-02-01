package email

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/contact"
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

func TestSMTPValidateEmailParams(t *testing.T) {
	logger := logging.New(logging.Config{Level: slog.LevelDebug})

	client, err := NewSMTPClient(&SMTPConfig{
		Host:      "smtp.example.com",
		FromEmail: "noreply@example.com",
	}, logger)
	require.NoError(t, err)

	t.Run("returns error for empty recipients", func(t *testing.T) {
		err := client.validateEmailParams([]string{}, "Subject", "Body")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one recipient is required")
	})

	t.Run("returns error for empty subject", func(t *testing.T) {
		err := client.validateEmailParams([]string{"user@example.com"}, "", "Body")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subject is required")
	})

	t.Run("returns error for empty body", func(t *testing.T) {
		err := client.validateEmailParams([]string{"user@example.com"}, "Subject", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "body is required")
	})

	t.Run("returns nil for valid params", func(t *testing.T) {
		err := client.validateEmailParams([]string{"user@example.com"}, "Subject", "Body")
		assert.NoError(t, err)
	})
}

func TestSMTPSendMethods(t *testing.T) {
	logger := logging.New(logging.Config{Level: slog.LevelDebug})

	// Create client with short timeout for faster test failure
	client, err := NewSMTPClient(&SMTPConfig{
		Host:      "localhost",
		Port:      59999, // Use a port that's unlikely to have a server
		FromEmail: "noreply@example.com",
		Timeout:   100 * time.Millisecond,
	}, logger)
	require.NoError(t, err)

	t.Run("Send returns connection error", func(t *testing.T) {
		email := contact.MustParseEmail("user@example.com")
		err := client.Send(context.Background(), email, "Test Subject", "Test Body")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
	})

	t.Run("SendHTML returns connection error", func(t *testing.T) {
		email := contact.MustParseEmail("user@example.com")
		err := client.SendHTML(context.Background(), email, "Test Subject", "<h1>Test</h1>")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
	})

	t.Run("SendToMultiple returns connection error", func(t *testing.T) {
		emails := []contact.Email{
			contact.MustParseEmail("user1@example.com"),
			contact.MustParseEmail("user2@example.com"),
		}
		err := client.SendToMultiple(context.Background(), emails, "Test Subject", "Test Body")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect")
	})
}

func TestSMTPSendEmailValidationErrors(t *testing.T) {
	logger := logging.New(logging.Config{Level: slog.LevelDebug})

	client, err := NewSMTPClient(&SMTPConfig{
		Host:      "smtp.example.com",
		FromEmail: "noreply@example.com",
	}, logger)
	require.NoError(t, err)

	t.Run("returns error for validation failure", func(t *testing.T) {
		// Empty email should fail validation before connection attempt
		err := client.sendEmail(context.Background(), []string{}, "Subject", "Body", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one recipient is required")
	})

	t.Run("returns error for empty subject", func(t *testing.T) {
		err := client.sendEmail(context.Background(), []string{"user@example.com"}, "", "Body", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "subject is required")
	})

	t.Run("returns error for empty body", func(t *testing.T) {
		err := client.sendEmail(context.Background(), []string{"user@example.com"}, "Subject", "", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "body is required")
	})
}

func TestSMTPClientPortDefaults(t *testing.T) {
	logger := logging.New(logging.Config{Level: slog.LevelDebug})

	t.Run("defaults to port 587 with TLS", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			FromEmail: "noreply@example.com",
			UseTLS:    true,
		}, logger)
		require.NoError(t, err)
		assert.Equal(t, 587, client.port)
	})

	t.Run("defaults to port 25 without TLS", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			FromEmail: "noreply@example.com",
			UseTLS:    false,
		}, logger)
		require.NoError(t, err)
		assert.Equal(t, 25, client.port)
	})

	t.Run("uses custom port", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			Port:      465,
			FromEmail: "noreply@example.com",
		}, logger)
		require.NoError(t, err)
		assert.Equal(t, 465, client.port)
	})
}

func TestSMTPClientTimeoutDefault(t *testing.T) {
	logger := logging.New(logging.Config{Level: slog.LevelDebug})

	t.Run("defaults to 30 second timeout", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			FromEmail: "noreply@example.com",
		}, logger)
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, client.timeout)
	})

	t.Run("uses custom timeout", func(t *testing.T) {
		client, err := NewSMTPClient(&SMTPConfig{
			Host:      "smtp.example.com",
			FromEmail: "noreply@example.com",
			Timeout:   60 * time.Second,
		}, logger)
		require.NoError(t, err)
		assert.Equal(t, 60*time.Second, client.timeout)
	})
}
