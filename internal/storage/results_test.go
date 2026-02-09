package storage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

func TestResultsStoreSaveListPrune(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBadgerStore(filepath.Join(dir, "badger"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	results := NewResultsStore(store)
	now := time.Now().UTC()
	if err := results.Save(scanner.Result{
		ScannerName: "test",
		Status:      scanner.StatusSuccess,
		FinishedAt:  now.Add(-48 * time.Hour),
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := results.Save(scanner.Result{
		ScannerName: "test",
		Status:      scanner.StatusSuccess,
		FinishedAt:  now,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	list, err := results.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 results, got %d", len(list))
	}

	if err := results.PruneOlderThan(now.Add(-24 * time.Hour)); err != nil {
		t.Fatalf("prune: %v", err)
	}
	list, err = results.List()
	if err != nil {
		t.Fatalf("list after prune: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 result after prune, got %d", len(list))
	}
}
