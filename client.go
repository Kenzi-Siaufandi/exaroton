package exaroton

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RawResponse is the exaroton API envelope wrapping every REST payload.
type RawResponse[T any] struct {
	Success bool    `json:"success"`
	Error   *string `json:"error,omitempty"`
	Data    *T      `json:"data,omitempty"`
}

// ApiError is a typed error for non-200 REST responses.
// It covers API error responses (400/403/404/500) and
// 208 AlreadyStarting/AlreadyStopping for server actions.
// Use errors.As to inspect StatusCode without parsing strings.
type ApiError struct {
	StatusCode int
	// Message is the API envelope error text, if present.
	Message string
	// Body holds a capped copy of the raw response body for debugging.
	Body []byte
}

// Error returns the API error message, preserving the legacy text format.
func (e *ApiError) Error() string {
	if e == nil {
		return "api error: <nil>"
	}
	if e.Message == "" {
		return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
	}
	if strings.HasPrefix(e.Message, "unexpected status code") {
		return e.Message
	}
	return fmt.Sprintf("api error (%d): %s", e.StatusCode, e.Message)
}

// newApiError builds an *ApiError preserving the pre-codegen message format
// so existing callers matching on text keep working.
func newApiError(status int, msg string, body []byte) *ApiError {
	const maxBody = 8 * 1024
	if len(body) > maxBody {
		body = body[:maxBody]
	}
	return &ApiError{StatusCode: status, Message: msg, Body: body}
}

// Client is a REST client for the exaroton API. Construct with NewClient.
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
	userAgent  string
}

// DefaultUserAgent is sent on every REST and websocket handshake unless
// overridden with WithUserAgent.
const DefaultUserAgent = "go-exaroton-api@1.0"

// Option configures a Client.
type Option func(*Client) error

// WithHTTPClient sets a custom HTTP client for REST calls. Its Timeout
// applies to JSON calls; file downloads (Read) use the request context
// deadline instead so large files are not cut off.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		if client == nil {
			return fmt.Errorf("httpClient cannot be nil")
		}
		c.httpClient = client
		return nil
	}
}

// WithBaseURL overrides the API base URL. A trailing slash is added when missing.
func WithBaseURL(url string) Option {
	return func(c *Client) error {
		if url == "" {
			return fmt.Errorf("baseURL cannot be empty")
		}
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		c.baseURL = url
		return nil
	}
}

// WithUserAgent overrides the User-Agent sent on REST and websocket handshakes.
func WithUserAgent(ua string) Option {
	return func(c *Client) error {
		if ua == "" {
			return fmt.Errorf("userAgent cannot be empty")
		}
		c.userAgent = ua
		return nil
	}
}

// NewClient creates a Client with the given API token and options.
func NewClient(token string, options ...Option) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	client := &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:   "https://api.exaroton.com/v1/",
		userAgent: DefaultUserAgent,
	}

	for _, option := range options {
		if err := option(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *Client) buildURL(path string) string {
	base := strings.TrimSuffix(c.baseURL, "/")
	p := strings.Trim(path, "/")
	if p == "" {
		return base + "/"
	}
	return base + "/" + p + "/"
}

func (c *Client) userAgentOrDefault() string {
	if c.userAgent == "" {
		return DefaultUserAgent
	}
	return c.userAgent
}

func (c *Client) newJSONRequest(ctx context.Context, method string, path string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.userAgentOrDefault())
	return req, nil
}

func (c *Client) newRawRequest(ctx context.Context, method string, path string, body any, contentType string) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = strings.NewReader(v)
		case io.Reader:
			bodyReader = v
		default:
			return nil, fmt.Errorf("unsupported body type %T: expected []byte, string or io.Reader", body)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("User-Agent", c.userAgentOrDefault())
	return req, nil
}

// parseApiError builds the typed error for a non-200 response. Bodies are
// JSON envelopes per the spec (400/403/404/500, plus 208 for server actions).
func parseApiError(resp *http.Response, raw []byte) *ApiError {
	var errorResp RawResponse[any]
	if err := json.Unmarshal(raw, &errorResp); err == nil && errorResp.Error != nil {
		return newApiError(resp.StatusCode, *errorResp.Error, raw)
	}
	return newApiError(resp.StatusCode, fmt.Sprintf("unexpected status code: %d", resp.StatusCode), raw)
}

func (c *Client) sendRequest(ctx context.Context, method string, path string, body any, result any) error {
	return c.doJSONRequest(ctx, method, path, body, result, false)
}

// sendActionRequest sends a server action (start/stop/restart/command,
// player-list add/remove) and validates only the success flag. Server action
// responses carry nullable data, so data:null is accepted.
func (c *Client) sendActionRequest(ctx context.Context, method string, path string, body any) error {
	var resp RawResponse[json.RawMessage]
	return c.doJSONRequest(ctx, method, path, body, &resp, true)
}

func (c *Client) doJSONRequest(ctx context.Context, method string, path string, body any, result any, allowNullData bool) error {
	req, err := c.newJSONRequest(ctx, method, path, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return parseApiError(resp, raw)
	}

	// Buffer body so we can decode into result and still validate success/data.
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return checkEnvelope(result, allowNullData)
}

// envelopeChecker is implemented by *RawResponse[T] for any T. It lets the
// client validate the success flag (and data presence) without reflection.
type envelopeChecker interface {
	checkEnvelope(allowNullData bool) error
}

// checkEnvelope validates the exaroton envelope: success must be true, and
// data must be non-nil unless allowNullData is set (server actions and empty
// responses declare data as nullable in the spec).
func (r *RawResponse[T]) checkEnvelope(allowNullData bool) error {
	if r == nil {
		return nil
	}
	if !r.Success {
		if r.Error != nil && *r.Error != "" {
			return fmt.Errorf("api error: %s", *r.Error)
		}
		return fmt.Errorf("api error: request was not successful")
	}
	if !allowNullData && r.Data == nil {
		return fmt.Errorf("api error: missing data in response")
	}
	return nil
}

func checkEnvelope(result any, allowNullData bool) error {
	if v, ok := result.(envelopeChecker); ok {
		return v.checkEnvelope(allowNullData)
	}
	return nil
}

// sendEmptyRequest sends a mutation (file PUT/DELETE) with an explicit
// content type and validates only the success flag. Empty responses carry
// nullable data, so data:null is accepted.
func (c *Client) sendEmptyRequest(ctx context.Context, method string, path string, body any, contentType string) error {
	if contentType == "" {
		contentType = "application/json"
	}
	req, err := c.newRawRequest(ctx, method, path, body, contentType)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return parseApiError(resp, raw)
	}

	// PUT/DELETE file endpoints return EmptyResponse JSON. Validate success
	// leniently: empty body is OK, non-JSON body is OK (forward compat).
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	var env RawResponse[json.RawMessage]
	if err := json.Unmarshal(data, &env); err != nil {
		return nil
	}
	if !env.Success {
		if env.Error != nil && *env.Error != "" {
			return fmt.Errorf("api error: %s", *env.Error)
		}
		return fmt.Errorf("api error: request was not successful")
	}
	return nil
}

// downloadClient returns a copy of the client's HTTP client without the
// whole-request Timeout, sharing the transport. Byte downloads are bounded
// by the caller's context instead.
func (c *Client) downloadClient() *http.Client {
	if c.httpClient == nil {
		return &http.Client{}
	}
	clone := *c.httpClient
	clone.Timeout = 0
	return &clone
}

func (c *Client) sendByteRequest(ctx context.Context, method string, path string, body any, contentType string) ([]byte, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req, err := c.newRawRequest(ctx, method, path, body, contentType)
	if err != nil {
		return nil, err
	}

	// Downloads run under the caller's context deadline only, not the
	// client's whole-request Timeout: a large world folder legitimately takes
	// longer than 10s. Pass a context with a timeout or cancel to bound the
	// download.
	resp, err := c.downloadClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, parseApiError(resp, raw)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}
