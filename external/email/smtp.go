package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/contact"
)

// SMTPClient is an SMTP email client for sending transactional emails.
type SMTPClient struct {
	host      string
	port      int
	username  string
	password  string
	fromEmail string
	fromName  string
	useTLS    bool
	tlsConfig *tls.Config
	timeout   time.Duration
	logger    *logging.Logger
}

// SMTPConfig holds the configuration for the SMTP client.
type SMTPConfig struct {
	// Host is the SMTP server host (required).
	Host string

	// Port is the SMTP server port (default: 587 for TLS, 25 for plain).
	Port int

	// Username is the SMTP authentication username (optional).
	Username string

	// Password is the SMTP authentication password (optional).
	Password string

	// FromEmail is the sender email address (required).
	FromEmail string

	// FromName is the sender display name (optional).
	FromName string

	// UseTLS enables STARTTLS (default: true).
	UseTLS bool

	// TLSConfig is the custom TLS configuration (optional).
	TLSConfig *tls.Config

	// Timeout is the connection timeout (default: 30s).
	Timeout time.Duration
}

// NewSMTPClient creates a new SMTP email client.
func NewSMTPClient(cfg *SMTPConfig, logger *logging.Logger) (*SMTPClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if cfg.FromEmail == "" {
		return nil, fmt.Errorf("from email is required")
	}

	port := cfg.Port
	if port == 0 {
		if cfg.UseTLS {
			port = 587
		} else {
			port = 25
		}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &SMTPClient{
		host:      cfg.Host,
		port:      port,
		username:  cfg.Username,
		password:  cfg.Password,
		fromEmail: cfg.FromEmail,
		fromName:  cfg.FromName,
		useTLS:    cfg.UseTLS,
		tlsConfig: cfg.TLSConfig,
		timeout:   timeout,
		logger:    logger,
	}, nil
}

// Send sends a plain text email via SMTP.
func (c *SMTPClient) Send(ctx context.Context, to contact.Email, subject, body string) error {
	return c.sendEmail(ctx, []string{to.String()}, subject, body, false)
}

// SendHTML sends an HTML email via SMTP.
func (c *SMTPClient) SendHTML(ctx context.Context, to contact.Email, subject, htmlBody string) error {
	return c.sendEmail(ctx, []string{to.String()}, subject, htmlBody, true)
}

// SendToMultiple sends an email to multiple recipients via SMTP.
func (c *SMTPClient) SendToMultiple(ctx context.Context, to []contact.Email, subject, body string) error {
	recipients := make([]string, len(to))
	for i, email := range to {
		recipients[i] = email.String()
	}
	return c.sendEmail(ctx, recipients, subject, body, false)
}

// sendEmail is the internal method that sends emails via SMTP.
func (c *SMTPClient) sendEmail(ctx context.Context, to []string, subject, body string, isHTML bool) error {
	if err := c.validateEmailParams(to, subject, body); err != nil {
		return err
	}

	// Build the email message
	msg := c.buildMessage(to, subject, body, isHTML)

	// Connect and send
	client, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := c.setupConnection(client); err != nil {
		return err
	}

	if err := c.sendMessage(client, to, msg); err != nil {
		return err
	}

	// Quit gracefully
	if quitErr := client.Quit(); quitErr != nil {
		// Log but don't fail - message was sent
		c.logger.WarnContext(ctx, "SMTP quit failed", "error", quitErr.Error())
	}

	c.logger.DebugContext(ctx, "email sent via SMTP",
		"to", strings.Join(to, ","),
		"subject", subject,
	)

	return nil
}

// validateEmailParams validates the email parameters.
func (c *SMTPClient) validateEmailParams(to []string, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if body == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

// connect establishes a connection to the SMTP server.
func (c *SMTPClient) connect(ctx context.Context) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	dialer := &net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, c.host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	return client, nil
}

// setupConnection configures TLS and authentication.
func (c *SMTPClient) setupConnection(client *smtp.Client) error {
	// STARTTLS if enabled
	if c.useTLS {
		tlsConfig := c.tlsConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{
				ServerName: c.host,
				MinVersion: tls.VersionTLS12,
			}
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate if credentials provided
	if c.username != "" && c.password != "" {
		auth := smtp.PlainAuth("", c.username, c.password, c.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	return nil
}

// sendMessage sends the email message to the recipients.
func (c *SMTPClient) sendMessage(client *smtp.Client, to []string, msg []byte) error {
	// Set sender
	if err := client.Mail(c.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send the message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

// buildMessage constructs the email message with headers.
func (c *SMTPClient) buildMessage(to []string, subject, body string, isHTML bool) []byte {
	var sb strings.Builder

	// From header
	if c.fromName != "" {
		sb.WriteString(fmt.Sprintf("From: %s <%s>\r\n", c.fromName, c.fromEmail))
	} else {
		sb.WriteString(fmt.Sprintf("From: %s\r\n", c.fromEmail))
	}

	// To header
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))

	// Subject header
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))

	// Content-Type header
	if isHTML {
		sb.WriteString("MIME-Version: 1.0\r\n")
		sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	// Date header
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z)))

	// End headers, start body
	sb.WriteString("\r\n")
	sb.WriteString(body)

	return []byte(sb.String())
}
