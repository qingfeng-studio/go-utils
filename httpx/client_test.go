package httpx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type echoPayload struct {
	Method string      `json:"method"`
	Path   string      `json:"path"`
	Query  url.Values  `json:"query"`
	Header http.Header `json:"header"`
	Body   string      `json:"body"`
}

func newEchoServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		payload := echoPayload{
			Method: r.Method,
			Path:   r.URL.Path,
			Query:  r.URL.Query(),
			Header: r.Header.Clone(),
			Body:   string(body),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
	return httptest.NewServer(handler)
}

func TestClient_Get(t *testing.T) {
	srv := newEchoServer()
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithTimeout(2*time.Second), WithHeader("X-Default", "A"))
	ctx := context.Background()

	resp, body, err := c.Get(ctx, "/api/resource", map[string]string{"q": "go"}, http.Header{"X-Req": []string{"B"}})
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var payload echoPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Method != http.MethodGet {
		t.Errorf("method = %s", payload.Method)
	}
	if payload.Path != "/api/resource" {
		t.Errorf("path = %s", payload.Path)
	}
	if payload.Query.Get("q") != "go" {
		t.Errorf("query q = %s", payload.Query.Get("q"))
	}
	if payload.Header.Get("X-Default") != "A" {
		t.Errorf("missing default header")
	}
	if payload.Header.Get("X-Req") != "B" {
		t.Errorf("missing per-request header")
	}
}

func TestClient_Post(t *testing.T) {
	srv := newEchoServer()
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	ctx := context.Background()
	body := []byte(`{"name":"alice"}`)
	resp, respBody, err := c.Post(ctx, "/v1/users", body, "application/json", nil, map[string]string{"debug": "1"})
	if err != nil {
		t.Fatalf("Post error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var payload echoPayload
	if err := json.Unmarshal(respBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Method != http.MethodPost {
		t.Errorf("method = %s", payload.Method)
	}
	if payload.Path != "/v1/users" {
		t.Errorf("path = %s", payload.Path)
	}
	if payload.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content-type not set")
	}
	if payload.Body != string(body) {
		t.Errorf("body mismatch: %s", payload.Body)
	}
	if payload.Query.Get("debug") != "1" {
		t.Errorf("query missing")
	}
}

func TestClient_Put(t *testing.T) {
	srv := newEchoServer()
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	ctx := context.Background()
	body := []byte("update")
	resp, respBody, err := c.Put(ctx, "/v1/items/123", body, "text/plain", http.Header{"X-A": []string{"1"}}, nil)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var payload echoPayload
	if err := json.Unmarshal(respBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Method != http.MethodPut {
		t.Errorf("method = %s", payload.Method)
	}
	if payload.Path != "/v1/items/123" {
		t.Errorf("path = %s", payload.Path)
	}
	if payload.Header.Get("X-A") != "1" {
		t.Errorf("missing header X-A")
	}
	if payload.Body != string(body) {
		t.Errorf("body mismatch: %s", payload.Body)
	}
}

func TestClient_Delete(t *testing.T) {
	srv := newEchoServer()
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	ctx := context.Background()
	resp, respBody, err := c.Delete(ctx, "/v1/items/bye", map[string]string{"force": "true"}, nil)
	if err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var payload echoPayload
	if err := json.Unmarshal(respBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Method != http.MethodDelete {
		t.Errorf("method = %s", payload.Method)
	}
	if payload.Path != "/v1/items/bye" {
		t.Errorf("path = %s", payload.Path)
	}
	if payload.Query.Get("force") != "true" {
		t.Errorf("query missing")
	}
}

func TestClient_Head_And_Options(t *testing.T) {
	// custom server to validate methods and headers without echoing a body for HEAD
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("X-Server", "test")
			w.WriteHeader(http.StatusNoContent)
		case http.MethodOptions:
			w.Header().Set("Allow", "GET,POST,PUT,DELETE,PATCH,HEAD,OPTIONS")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	ctx := context.Background()

	// HEAD
	resp, body, err := c.Head(ctx, "/res", nil, nil)
	if err != nil {
		t.Fatalf("Head error: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if len(body) != 0 {
		t.Errorf("HEAD must not have body, got %d bytes", len(body))
	}
	if resp.Header.Get("X-Server") != "test" {
		t.Errorf("missing X-Server header")
	}

	// OPTIONS
	resp, body, err = c.Options(ctx, "/res", nil, http.Header{"X-Req": []string{"1"}})
	if err != nil {
		t.Fatalf("Options error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if resp.Header.Get("Allow") == "" {
		t.Errorf("missing Allow header")
	}
	if string(body) != "OK" {
		t.Errorf("unexpected body: %q", string(body))
	}
}

func TestClient_Patch(t *testing.T) {
	srv := newEchoServer()
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL))
	ctx := context.Background()
	body := []byte(`{"op":"replace","path":"/name","value":"bob"}`)
	resp, respBody, err := c.Patch(ctx, "/v1/users/123", body, "application/json", nil, nil)
	if err != nil {
		t.Fatalf("Patch error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var payload echoPayload
	if err := json.Unmarshal(respBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Method != http.MethodPatch {
		t.Errorf("method = %s", payload.Method)
	}
	if payload.Path != "/v1/users/123" {
		t.Errorf("path = %s", payload.Path)
	}
	if payload.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content-type not set")
	}
	if payload.Body != string(body) {
		t.Errorf("body mismatch: %s", payload.Body)
	}
}

func TestClient_ContextRequired(t *testing.T) {
	c := NewClient()
	if _, _, err := c.Get(nil, "http://example.com", nil, nil); err == nil {
		t.Fatalf("expected error when ctx is nil")
	}
}

func TestClient_AbsoluteURL(t *testing.T) {
	srv := newEchoServer()
	defer srv.Close()

	c := NewClient()
	ctx := context.Background()
	resp, body, err := c.Get(ctx, srv.URL+"/echo", map[string]string{"k": "v"}, nil)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	var payload echoPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Path != "/echo" {
		t.Errorf("path = %s", payload.Path)
	}
	if payload.Query.Get("k") != "v" {
		t.Errorf("query = %s", payload.Query.Get("k"))
	}
}
