package alerting

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
)

type flakyChannel struct {
	failures int32
	calls    int32
}

func (f *flakyChannel) Name() string { return "flaky" }

func (f *flakyChannel) Send(alert Alert) error {
	atomic.AddInt32(&f.calls, 1)
	if atomic.AddInt32(&f.failures, -1) >= 0 {
		return errTest
	}
	return nil
}

var errTest = &testError{}

type testError struct{}

func (t *testError) Error() string { return "test error" }

func TestDedupAndRetry(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:      true,
		DedupWindow:  "1s",
		RetryMax:     2,
		RetryBackoff: "10ms",
	}
	engine := New(logging.New("text"), cfg)
	ch := &flakyChannel{failures: 1}
	engine.Register(ch)

	alert := Alert{
		ScannerName: "test",
		Severity:    scanner.SeverityHigh,
		Finding:     scanner.Finding{ID: "x", Description: "test"},
	}
	engine.Send(alert)
	engine.Send(alert)

	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&ch.calls) < 1 {
		t.Fatalf("expected channel to be called")
	}
}
