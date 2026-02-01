// Package email provides a client for sending emails via SendGrid.
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

// SendGrid API endpoint.
const sendGridAPIURL = "https://api.sendgrid.com/v3/mail/send"

// Client is the Email client for SendGrid.
type Client struct {
	httpClient *http.Client
	apiKey     string
	fromEmail  string
	fromName   string
	logger     *logging.Logger
}

// Config holds the configuration for the Email client.
type Config struct {
	// APIKey is the SendGrid API key (required).
	APIKey string

	// FromEmail is the sender email address (required).
	FromEmail string

	// FromName is the sender name (optional).
	FromName string

	// Timeout is the request timeout (default: 30s).
	Timeout time.Duration
}

// NewClient creates a new Email client.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
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

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		apiKey:     cfg.APIKey,
		fromEmail:  cfg.FromEmail,
		fromName:   cfg.FromName,
		logger:     logger,
	}, nil
}

// EmailAddress represents an email address with optional name.
type Address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Content represents email content.
type Content struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Personalization represents email personalization (recipients).
type Personalization struct {
	To      []Address         `json:"to"`
	Subject string            `json:"subject,omitempty"`
	Subs    map[string]string `json:"substitutions,omitempty"`
}

// sendGridRequest is the request body for SendGrid API.
type sendGridRequest struct {
	Personalizations []Personalization `json:"personalizations"`
	From             Address           `json:"from"`
	Subject          string            `json:"subject"`
	Content          []Content         `json:"content,omitempty"`
	TemplateID       string            `json:"template_id,omitempty"`
}

// Send sends a plain text email.
func (c *Client) Send(ctx context.Context, to contact.Email, subject, body string) error {
	if to.IsZero() {
		return fmt.Errorf("recipient email is required")
	}
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if body == "" {
		return fmt.Errorf("body is required")
	}

	req := sendGridRequest{
		Personalizations: []Personalization{{
			To: []Address{{Email: to.String()}},
		}},
		From: Address{
			Email: c.fromEmail,
			Name:  c.fromName,
		},
		Subject: subject,
		Content: []Content{{
			Type:  "text/plain",
			Value: body,
		}},
	}

	return c.sendRequest(ctx, req)
}

// SendHTML sends an HTML email.
func (c *Client) SendHTML(ctx context.Context, to contact.Email, subject, htmlBody string) error {
	if to.IsZero() {
		return fmt.Errorf("recipient email is required")
	}
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if htmlBody == "" {
		return fmt.Errorf("HTML body is required")
	}

	req := sendGridRequest{
		Personalizations: []Personalization{{
			To: []Address{{Email: to.String()}},
		}},
		From: Address{
			Email: c.fromEmail,
			Name:  c.fromName,
		},
		Subject: subject,
		Content: []Content{{
			Type:  "text/html",
			Value: htmlBody,
		}},
	}

	return c.sendRequest(ctx, req)
}

// SendTemplate sends an email using a SendGrid template.
func (c *Client) SendTemplate(ctx context.Context, to contact.Email, templateID string, data map[string]string) error {
	if to.IsZero() {
		return fmt.Errorf("recipient email is required")
	}
	if templateID == "" {
		return fmt.Errorf("template ID is required")
	}

	req := sendGridRequest{
		Personalizations: []Personalization{{
			To:   []Address{{Email: to.String()}},
			Subs: data,
		}},
		From: Address{
			Email: c.fromEmail,
			Name:  c.fromName,
		},
		TemplateID: templateID,
	}

	return c.sendRequest(ctx, req)
}

// SendToMultiple sends an email to multiple recipients.
func (c *Client) SendToMultiple(ctx context.Context, to []contact.Email, subject, body string) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient email is required")
	}
	if subject == "" {
		return fmt.Errorf("subject is required")
	}
	if body == "" {
		return fmt.Errorf("body is required")
	}

	recipients := make([]Address, len(to))
	for i, email := range to {
		if email.IsZero() {
			return fmt.Errorf("recipient email at index %d is invalid", i)
		}
		recipients[i] = Address{Email: email.String()}
	}

	req := sendGridRequest{
		Personalizations: []Personalization{{
			To: recipients,
		}},
		From: Address{
			Email: c.fromEmail,
			Name:  c.fromName,
		},
		Subject: subject,
		Content: []Content{{
			Type:  "text/plain",
			Value: body,
		}},
	}

	return c.sendRequest(ctx, req)
}

// apiURL is the URL for the SendGrid API (can be overridden for testing).
var apiURL = sendGridAPIURL

func (c *Client) sendRequest(ctx context.Context, sgReq sendGridRequest) error {
	body, err := json.Marshal(sgReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// SendGrid returns 202 Accepted for successful sends
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("SendGrid API error: status %d (failed to read body: %w)", resp.StatusCode, err)
		}
		return fmt.Errorf("SendGrid API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SetHTTPClient sets a custom HTTP client (useful for testing).
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetAPIURL sets a custom API URL (useful for testing).
func SetAPIURL(url string) {
	apiURL = url
}

// ResetAPIURL resets the API URL to the default.
func ResetAPIURL() {
	apiURL = sendGridAPIURL
}
