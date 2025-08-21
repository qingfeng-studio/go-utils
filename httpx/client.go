package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ClientOptions holds configuration for Client construction.
type ClientOptions struct {
	BaseURL   string
	Timeout   time.Duration
	Headers   http.Header
	Transport http.RoundTripper
}

// Option configures ClientOptions.
type Option func(*ClientOptions)

// WithBaseURL sets a base URL used to resolve relative paths.
func WithBaseURL(baseURL string) Option {
	return func(o *ClientOptions) { o.BaseURL = strings.TrimRight(baseURL, "/") }
}

// WithTimeout sets the http client timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(o *ClientOptions) { o.Timeout = timeout }
}

// WithHeader adds a default header applied to every request.
func WithHeader(key, value string) Option {
	return func(o *ClientOptions) {
		if o.Headers == nil {
			o.Headers = make(http.Header)
		}
		o.Headers.Add(key, value)
	}
}

// WithTransport sets a custom transport.
func WithTransport(rt http.RoundTripper) Option {
	return func(o *ClientOptions) { o.Transport = rt }
}

// Client is a thin convenience wrapper around http.Client that provides
// ergonomic helpers for common HTTP verbs and consistent option handling.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	defaultHeaders http.Header
}

// NewClient constructs a Client with the provided options.
func NewClient(options ...Option) *Client {
	opts := &ClientOptions{
		Timeout: 30 * time.Second,
	}
	for _, o := range options {
		o(opts)
	}

	hc := &http.Client{Timeout: opts.Timeout}
	if opts.Transport != nil {
		hc.Transport = opts.Transport
	}

	return &Client{
		httpClient:     hc,
		baseURL:        opts.BaseURL,
		defaultHeaders: cloneHeader(opts.Headers),
	}
}

// Get issues a GET request.
func (c *Client) Get(ctx context.Context, path string, query map[string]string, headers http.Header) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodGet, path, nil, headers, query, "")
}

// Delete issues a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, query map[string]string, headers http.Header) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodDelete, path, nil, headers, query, "")
}

// Head issues a HEAD request.
func (c *Client) Head(ctx context.Context, path string, query map[string]string, headers http.Header) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodHead, path, nil, headers, query, "")
}

// Options issues an OPTIONS request.
func (c *Client) Options(ctx context.Context, path string, query map[string]string, headers http.Header) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodOptions, path, nil, headers, query, "")
}

// Post issues a POST request with raw body and content type.
func (c *Client) Post(ctx context.Context, path string, body []byte, contentType string, headers http.Header, query map[string]string) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodPost, path, bytes.NewReader(body), headers, query, contentType)
}

// Put issues a PUT request with raw body and content type.
func (c *Client) Put(ctx context.Context, path string, body []byte, contentType string, headers http.Header, query map[string]string) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodPut, path, bytes.NewReader(body), headers, query, contentType)
}

// Patch issues a PATCH request with raw body and content type.
func (c *Client) Patch(ctx context.Context, path string, body []byte, contentType string, headers http.Header, query map[string]string) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodPatch, path, bytes.NewReader(body), headers, query, contentType)
}

// do performs the HTTP request and returns the response and full response body.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, headers http.Header, query map[string]string, contentType string) (*http.Response, []byte, error) {
	if ctx == nil {
		return nil, nil, errors.New("context must not be nil")
	}

	fullURL, err := c.resolveURL(path, query)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, nil, err
	}

	// Merge headers: defaults first, then per-request overrides
	for k, vs := range c.defaultHeaders {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, err
	}
	if resp.StatusCode >= 400 {
		return resp, respBody, fmt.Errorf("http %s %s failed: status=%d body=%s", method, fullURL, resp.StatusCode, truncate(respBody, 512))
	}
	return resp, respBody, nil
}

func (c *Client) resolveURL(path string, query map[string]string) (string, error) {
	// If path is absolute (starts with http:// or https://), use it directly
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return addQuery(path, query)
	}

	base := strings.TrimRight(c.baseURL, "/")
	if base == "" {
		// Allow using path as absolute path without baseURL
		return addQuery(path, query)
	}

	joined := base + "/" + strings.TrimLeft(path, "/")
	return addQuery(joined, query)
}

func addQuery(rawURL string, query map[string]string) (string, error) {
	if len(query) == 0 {
		return rawURL, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	out := make(http.Header, len(h))
	for k, vs := range h {
		copied := make([]string, len(vs))
		copy(copied, vs)
		out[k] = copied
	}
	return out
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
