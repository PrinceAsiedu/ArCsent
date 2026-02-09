package daemon

import (
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/ipsix/arcsent/internal/logging"
)

func TestHandleSignalsCallsReloadOnSIGHUP(t *testing.T) {
	runner := &Runner{logger: logging.New("text")}
	sigCh := make(chan os.Signal, 1)
	var reloadCalled atomic.Bool

	done := make(chan struct{})
	go func() {
		runner.handleSignals(sigCh, func() {}, func() { reloadCalled.Store(true) })
		close(done)
	}()

	sigCh <- syscall.SIGHUP
	close(sigCh)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("handleSignals did not return after closing channel")
	}

	if !reloadCalled.Load() {
		t.Fatalf("expected reload to be called on SIGHUP")
	}
}
