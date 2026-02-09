package detection

import (
	"path/filepath"
	"testing"

	"github.com/ipsix/arcsent/internal/storage"
)

func TestBaselineUpdateAndAnomaly(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewBadgerStore(filepath.Join(dir, "badger"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()

	mgr := NewManager(store)
	for i := 0; i < 20; i++ {
		if _, err := mgr.Update("scanner", "metric", float64(10+i%3)); err != nil {
			t.Fatalf("update: %v", err)
		}
	}

	anomaly, reason, err := mgr.IsAnomaly("scanner", "metric", 1000)
	if err != nil {
		t.Fatalf("isAnomaly: %v", err)
	}
	if !anomaly {
		t.Fatalf("expected anomaly, got reason=%s", reason)
	}
}
