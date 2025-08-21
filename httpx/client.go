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

// ClientOptions 客户端构建时的可选配置
// 实用场景: 当你需要统一设置 BaseURL、超时、默认 Header 或自定义 Transport
// 来初始化 HTTP 客户端时使用
type ClientOptions struct {
	BaseURL   string            // 基础 URL，用于拼接相对路径，适用于服务地址固定场景
	Timeout   time.Duration     // 请求超时时间，用于控制长请求或防止阻塞
	Headers   http.Header       // 默认请求头，每次请求都会附加，可用于统一添加认证、User-Agent 等
	Transport http.RoundTripper // 自定义 HTTP Transport，用于代理、TLS 配置、连接复用等
}

// Option 用于配置 ClientOptions 的函数式选项
type Option func(*ClientOptions)

// WithBaseURL 设置基础地址（结尾多余的 / 会被移除）
func WithBaseURL(baseURL string) Option {
	return func(o *ClientOptions) { o.BaseURL = strings.TrimRight(baseURL, "/") }
}

// WithTimeout 设置 http.Client 的超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(o *ClientOptions) { o.Timeout = timeout }
}

// WithHeader 增加默认请求头（可多次调用累加）
func WithHeader(key, value string) Option {
	return func(o *ClientOptions) {
		if o.Headers == nil {
			o.Headers = make(http.Header)
		}
		o.Headers.Add(key, value)
	}
}

// WithTransport 设置自定义 Transport（例如代理、TLS、连接复用等）
func WithTransport(rt http.RoundTripper) Option {
	return func(o *ClientOptions) { o.Transport = rt }
}

// Client 对 http.Client 的轻量封装
// 实用场景: 当你希望在项目中统一处理 BaseURL、默认请求头、查询参数、Content-Type 并
// 提供便捷的 GET/POST/PUT/PATCH/DELETE 方法时使用
type Client struct {
	httpClient     *http.Client // 内部 http.Client 实例，用于发送请求
	baseURL        string       // 基础 URL，用于拼接相对路径
	defaultHeaders http.Header  // 默认请求头，供每次请求使用，可被 per-request headers 覆盖
}

// NewClient 根据可选项创建 Client 实例
func NewClient(options ...Option) *Client {
	opts := &ClientOptions{
		Timeout: 30 * time.Second, // 默认超时 30 秒
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

// Get 发送 GET 请求，用于获取资源，支持 query 参数和自定义 headers
func (c *Client) Get(ctx context.Context, path string, query map[string]string, headers http.Header) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodGet, path, nil, headers, query, "")
}

// Delete 发送 DELETE 请求，用于删除资源，支持 query 参数和自定义 headers
func (c *Client) Delete(ctx context.Context, path string, query map[string]string, headers http.Header) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodDelete, path, nil, headers, query, "")
}

// Head 发送 HEAD 请求（响应通常无 Body），用于检查资源存在性或获取元信息
func (c *Client) Head(ctx context.Context, path string, query map[string]string, headers http.Header) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodHead, path, nil, headers, query, "")
}

// Options 发送 OPTIONS 请求，用于探测服务支持的 HTTP 方法
func (c *Client) Options(ctx context.Context, path string, query map[string]string, headers http.Header) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodOptions, path, nil, headers, query, "")
}

// Post 发送 POST 请求，用于创建资源或提交数据
func (c *Client) Post(ctx context.Context, path string, body []byte, contentType string, headers http.Header, query map[string]string) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodPost, path, bytes.NewReader(body), headers, query, contentType)
}

// Put 发送 PUT 请求，用于更新资源的全部字段
func (c *Client) Put(ctx context.Context, path string, body []byte, contentType string, headers http.Header, query map[string]string) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodPut, path, bytes.NewReader(body), headers, query, contentType)
}

// Patch 发送 PATCH 请求，用于更新资源的部分字段
func (c *Client) Patch(ctx context.Context, path string, body []byte, contentType string, headers http.Header, query map[string]string) (*http.Response, []byte, error) {
	return c.do(ctx, http.MethodPatch, path, bytes.NewReader(body), headers, query, contentType)
}

// do 执行 HTTP 请求核心逻辑，内部方法
// 实用场景: 所有 HTTP 方法均调用此方法，实现统一的请求逻辑和错误处理
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, headers http.Header, query map[string]string, contentType string) (*http.Response, []byte, error) {
	if ctx == nil {
		return nil, nil, errors.New("context must not be nil")
	}

	fullURL, err := c.resolveURL(path, query) // 拼接完整 URL
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body) // 创建请求
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

	resp, err := c.httpClient.Do(req) // 执行请求
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }() // 确保关闭 Body

	respBody, err := io.ReadAll(resp.Body) // 读取响应体
	if err != nil {
		return resp, nil, err
	}
	if resp.StatusCode >= 400 {
		return resp, respBody, fmt.Errorf("http %s %s failed: status=%d body=%s", method, fullURL, resp.StatusCode, truncate(respBody, 512))
	}
	return resp, respBody, nil
}

// resolveURL 解析相对路径或绝对 URL，并拼接 query 参数
func (c *Client) resolveURL(path string, query map[string]string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return addQuery(path, query)
	}
	base := strings.TrimRight(c.baseURL, "/")
	if base == "" {
		return addQuery(path, query)
	}
	joined := base + "/" + strings.TrimLeft(path, "/")
	return addQuery(joined, query)
}

// addQuery 给 URL 拼接 query 参数
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

// cloneHeader 克隆 Header，防止修改默认 Header
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

// truncate 截断字节数组为字符串，用于日志输出
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
