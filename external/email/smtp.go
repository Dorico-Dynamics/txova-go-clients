package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"regexp"
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

	// Validate fromEmail for CRLF injection and allowed characters.
	if err := validateEmailAddress(cfg.FromEmail); err != nil {
		return nil, fmt.Errorf("invalid from email: %w", err)
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
		fromName:  sanitizeHeaderValue(cfg.FromName),
		useTLS:    cfg.UseTLS,
		tlsConfig: cfg.TLSConfig,
		timeout:   timeout,
		logger:    logger,
	}, nil
}

// emailAddressRegex validates basic email address format.
// This is a simplified pattern; full RFC 5322 validation is complex.
var emailAddressRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// validateEmailAddress checks for CRLF injection and validates format.
func validateEmailAddress(email string) error {
	// Check for CRLF injection.
	if strings.ContainsAny(email, "\r\n") {
		return fmt.Errorf("email address contains invalid characters (CR/LF)")
	}

	// Basic format validation.
	if !emailAddressRegex.MatchString(email) {
		return fmt.Errorf("invalid email address format")
	}

	return nil
}

// sanitizeHeaderValue removes CR and LF characters to prevent header injection.
func sanitizeHeaderValue(value string) string {
	// Remove any CR or LF characters.
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return value
}

// encodeHeaderValue RFC-2047 encodes a header value if it contains non-ASCII characters.
func encodeHeaderValue(value string) string {
	// Check if encoding is needed (non-ASCII characters present).
	needsEncoding := false
	for _, r := range value {
		if r > 127 {
			needsEncoding = true
			break
		}
	}

	if !needsEncoding {
		return value
	}

	// Use MIME Q-encoding for the header value.
	return mime.QEncoding.Encode("UTF-8", value)
}

// formatEmailAddress formats an email address with optional display name.
// The display name is RFC-2047 encoded if it contains non-ASCII characters.
func formatEmailAddress(name, email string) string {
	if name == "" {
		return email
	}

	// Sanitize and encode the display name.
	name = sanitizeHeaderValue(name)
	encodedName := encodeHeaderValue(name)

	return fmt.Sprintf("%s <%s>", encodedName, email)
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

	// Build the email message.
	msg := c.buildMessage(to, subject, body, isHTML)

	// Connect and send.
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

	// Quit gracefully.
	if quitErr := client.Quit(); quitErr != nil {
		// Log but don't fail - message was sent.
		if c.logger != nil {
			c.logger.WarnContext(ctx, "SMTP quit failed", "error", quitErr.Error())
		}
	}

	if c.logger != nil {
		c.logger.DebugContext(ctx, "email sent via SMTP",
			"recipient_count", len(to),
		)
	}

	return nil
}

// validateEmailParams validates the email parameters.
func (c *SMTPClient) validateEmailParams(to []string, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	// Validate each recipient address.
	for _, recipient := range to {
		if err := validateEmailAddress(recipient); err != nil {
			return fmt.Errorf("invalid recipient %q: %w", recipient, err)
		}
	}

	// Check subject for CRLF injection.
	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("subject contains invalid characters (CR/LF)")
	}

	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if body == "" {
		return fmt.Errorf("body is required")
	}
	return nil
}

// deadlineConn wraps a net.Conn and sets read/write deadlines before each operation.
// This ensures all SMTP operations (StartTLS, Auth, Mail, Rcpt, Data, Quit) are bounded.
type deadlineConn struct {
	net.Conn
	timeout time.Duration
}

// Read sets a read deadline before reading.
func (c *deadlineConn) Read(b []byte) (int, error) {
	if err := c.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

// Write sets a write deadline before writing.
func (c *deadlineConn) Write(b []byte) (int, error) {
	if err := c.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

// connect establishes a connection to the SMTP server.
func (c *SMTPClient) connect(ctx context.Context) (*smtp.Client, error) {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	dialer := &net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	// Wrap connection with deadline enforcement for all SMTP operations.
	wrappedConn := &deadlineConn{Conn: conn, timeout: c.timeout}

	client, err := smtp.NewClient(wrappedConn, c.host)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	return client, nil
}

// setupConnection configures TLS and authentication.
func (c *SMTPClient) setupConnection(client *smtp.Client) error {
	// STARTTLS if enabled.
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

	// Authenticate if credentials provided.
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
	// Set sender.
	if err := client.Mail(c.fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients.
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send the message body.
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
// All header values are sanitized and RFC-2047 encoded as needed.
func (c *SMTPClient) buildMessage(to []string, subject, body string, isHTML bool) []byte {
	var sb strings.Builder

	// From header with properly encoded display name.
	fromHeader := formatEmailAddress(c.fromName, c.fromEmail)
	sb.WriteString(fmt.Sprintf("From: %s\r\n", fromHeader))

	// To header - recipients are plain addresses (already validated).
	sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))

	// Subject header - sanitized and RFC-2047 encoded.
	sanitizedSubject := sanitizeHeaderValue(subject)
	encodedSubject := encodeHeaderValue(sanitizedSubject)
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))

	// Content-Type header.
	if isHTML {
		sb.WriteString("MIME-Version: 1.0\r\n")
		sb.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	// Date header.
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z)))

	// End headers, start body.
	sb.WriteString("\r\n")
	sb.WriteString(body)

	return []byte(sb.String())
}
