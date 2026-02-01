package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Dorico-Dynamics/txova-go-core/logging"
	"github.com/Dorico-Dynamics/txova-go-types/contact"
)

// Resend API base URL.
const resendBaseURL = "https://api.resend.com"

// ResendClient is a Resend email client for sending transactional emails.
// Resend is a modern email API designed for developers with excellent deliverability.
type ResendClient struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	fromEmail  string
	fromName   string
	logger     *logging.Logger
}

// ResendConfig holds the configuration for the Resend client.
type ResendConfig struct {
	// APIKey is the Resend API key (required).
	APIKey string

	// FromEmail is the sender email address (required).
	FromEmail string

	// FromName is the sender display name (optional).
	FromName string

	// Timeout is the request timeout (default: 30s).
	Timeout time.Duration
}

// NewResendClient creates a new Resend email client.
func NewResendClient(cfg *ResendConfig, logger *logging.Logger) (*ResendClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if cfg.FromEmail == "" {
		return nil, fmt.Errorf("from email is required")
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &ResendClient{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    resendBaseURL,
		apiKey:     cfg.APIKey,
		fromEmail:  cfg.FromEmail,
		fromName:   cfg.FromName,
		logger:     logger,
	}, nil
}

// resendEmailRequest is the request body for sending emails via Resend.
type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Text    string   `json:"text,omitempty"`
	HTML    string   `json:"html,omitempty"`
	Cc      []string `json:"cc,omitempty"`
	Bcc     []string `json:"bcc,omitempty"`
	ReplyTo string   `json:"reply_to,omitempty"`
	Tags    []Tag    `json:"tags,omitempty"`
}

// Tag represents an email tag for tracking.
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ResendSendResult represents the result of sending an email via Resend.
type ResendSendResult struct {
	// ID is the unique email ID.
	ID string `json:"id"`
}

// Send sends a plain text email via Resend.
func (c *ResendClient) Send(ctx context.Context, to contact.Email, subject, body string) error {
	req := resendEmailRequest{
		From:    c.formatFrom(),
		To:      []string{to.String()},
		Subject: subject,
		Text:    body,
	}
	_, err := c.sendEmail(ctx, req)
	return err
}

// SendHTML sends an HTML email via Resend.
func (c *ResendClient) SendHTML(ctx context.Context, to contact.Email, subject, htmlBody string) error {
	req := resendEmailRequest{
		From:    c.formatFrom(),
		To:      []string{to.String()},
		Subject: subject,
		HTML:    htmlBody,
	}
	_, err := c.sendEmail(ctx, req)
	return err
}

// SendToMultiple sends an email to multiple recipients via Resend.
func (c *ResendClient) SendToMultiple(ctx context.Context, to []contact.Email, subject, body string) error {
	recipients := make([]string, len(to))
	for i, email := range to {
		recipients[i] = email.String()
	}
	req := resendEmailRequest{
		From:    c.formatFrom(),
		To:      recipients,
		Subject: subject,
		Text:    body,
	}
	_, err := c.sendEmail(ctx, req)
	return err
}

// SendWithOptions sends an email with additional options like CC, BCC, and tags.
type SendOptions struct {
	To      []contact.Email
	Subject string
	Text    string
	HTML    string
	Cc      []contact.Email
	Bcc     []contact.Email
	ReplyTo string
	Tags    map[string]string
}

// SendWithOptions sends an email with advanced options via Resend.
func (c *ResendClient) SendWithOptions(ctx context.Context, opts SendOptions) (*ResendSendResult, error) {
	if len(opts.To) == 0 {
		return nil, fmt.Errorf("at least one recipient is required")
	}
	if opts.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if opts.Text == "" && opts.HTML == "" {
		return nil, fmt.Errorf("text or HTML body is required")
	}

	recipients := make([]string, len(opts.To))
	for i, email := range opts.To {
		recipients[i] = email.String()
	}

	var cc []string
	if len(opts.Cc) > 0 {
		cc = make([]string, len(opts.Cc))
		for i, email := range opts.Cc {
			cc[i] = email.String()
		}
	}

	var bcc []string
	if len(opts.Bcc) > 0 {
		bcc = make([]string, len(opts.Bcc))
		for i, email := range opts.Bcc {
			bcc[i] = email.String()
		}
	}

	var tags []Tag
	if len(opts.Tags) > 0 {
		tags = make([]Tag, 0, len(opts.Tags))
		for name, value := range opts.Tags {
			tags = append(tags, Tag{Name: name, Value: value})
		}
	}

	req := resendEmailRequest{
		From:    c.formatFrom(),
		To:      recipients,
		Subject: opts.Subject,
		Text:    opts.Text,
		HTML:    opts.HTML,
		Cc:      cc,
		Bcc:     bcc,
		ReplyTo: opts.ReplyTo,
		Tags:    tags,
	}

	return c.sendEmail(ctx, req)
}

// sendEmail is the internal method that sends emails via the Resend API.
func (c *ResendClient) sendEmail(ctx context.Context, req resendEmailRequest) (*ResendSendResult, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/emails", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("resend API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result ResendSendResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if c.logger != nil {
		c.logger.DebugContext(ctx, "email sent via Resend",
			"email_id", result.ID,
			"to", req.To,
			"subject", req.Subject,
		)
	}

	return &result, nil
}

// formatFrom formats the sender address with optional display name.
func (c *ResendClient) formatFrom() string {
	if c.fromName != "" {
		return fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail)
	}
	return c.fromEmail
}

// SetBaseURL sets a custom base URL (useful for testing).
func (c *ResendClient) SetBaseURL(url string) {
	c.baseURL = url
}

// SetHTTPClient sets a custom HTTP client (useful for testing).
func (c *ResendClient) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}
