package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/logging"
)

func TestWebUIAuth(t *testing.T) {
	cfg := config.WebUIConfig{
		Enabled:   true,
		BindAddr:  "127.0.0.1:0",
		AuthToken: "secret",
	}

	server := New(cfg, "127.0.0.1:8788", logging.New("text"))
	handler, err := server.buildHandler()
	if err != nil {
		t.Fatalf("build handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected ok without token for UI landing, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "secret")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected ok with token, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized for api without token, got %d", rr.Code)
	}
}
