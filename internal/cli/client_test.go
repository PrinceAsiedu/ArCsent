package cli

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestClientAddsAuthHeader(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "token" {
			t.Fatalf("expected Authorization header, got %q", got)
		}
		body := io.NopCloser(strings.NewReader(`{"ok":true}`))
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       body,
			Header:     http.Header{},
		}, nil
	})

	client := NewClient("http://127.0.0.1:8788", "token")
	client.Client = &http.Client{Transport: transport}
	raw, err := client.DoJSON(context.Background(), http.MethodGet, "/health", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if !strings.Contains(string(raw), "ok") {
		t.Fatalf("expected response body, got %s", string(raw))
	}
}

func TestClientErrorOnNon2xx(t *testing.T) {
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := io.NopCloser(strings.NewReader(`{"error":"nope"}`))
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Status:     "401 Unauthorized",
			Body:       body,
			Header:     http.Header{},
		}, nil
	})

	client := NewClient("http://127.0.0.1:8788", "token")
	client.Client = &http.Client{Transport: transport}
	_, err := client.DoJSON(context.Background(), http.MethodGet, "/status", nil)
	if err == nil {
		t.Fatalf("expected error for non-2xx response")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (rt roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt(req)
}
