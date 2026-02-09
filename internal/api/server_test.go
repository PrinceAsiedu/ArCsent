package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/detection"
	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
	"github.com/ipsix/arcsent/internal/scheduler"
	"github.com/ipsix/arcsent/internal/state"
	"github.com/ipsix/arcsent/internal/storage"
)

type dummyPlugin struct{}

func (d *dummyPlugin) Name() string                      { return "dummy" }
func (d *dummyPlugin) Init(map[string]interface{}) error { return nil }
func (d *dummyPlugin) Run(ctx context.Context) (*scanner.Result, error) {
	return &scanner.Result{ScannerName: "dummy", Status: scanner.StatusSuccess}, nil
}
func (d *dummyPlugin) Halt(ctx context.Context) error { return nil }

func TestAPIAuth(t *testing.T) {
	mgr := scanner.NewManager()
	if err := mgr.Register(&dummyPlugin{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	sched := scheduler.New(logging.New("text"), mgr)
	results := state.NewResultCache(10)
	store, err := storage.NewBadgerStore(filepath.Join(t.TempDir(), "badger"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()
	baseline := detection.NewManager(store)

	cfg := config.APIConfig{Enabled: true, BindAddr: "127.0.0.1:0", AuthToken: "secret"}
	server := New(cfg, logging.New("text"), mgr, sched, results, baseline, nil, nil, nil)
	handler := server.buildHandler()

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without token, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("Authorization", "secret")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected ok with token, got %d", rr.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	mgr := scanner.NewManager()
	if err := mgr.Register(&dummyPlugin{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	sched := scheduler.New(logging.New("text"), mgr)
	results := state.NewResultCache(10)
	store, err := storage.NewBadgerStore(filepath.Join(t.TempDir(), "badger"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()
	baseline := detection.NewManager(store)

	cfg := config.APIConfig{Enabled: true, BindAddr: "127.0.0.1:0", AuthToken: "secret"}
	server := New(cfg, logging.New("text"), mgr, sched, results, baseline, nil, nil, nil)
	handler := server.buildHandler()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "secret")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected ok, got %d", rr.Code)
	}
	if got := rr.Body.String(); got == "" || !strings.Contains(got, "arcsent_up") {
		t.Fatalf("expected metrics body to include arcsent_up, got: %s", got)
	}
}
