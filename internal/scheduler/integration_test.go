package scheduler

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ipsix/arcsent/internal/detection"
	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/plugins/system"
	"github.com/ipsix/arcsent/internal/scanner"
	"github.com/ipsix/arcsent/internal/state"
	"github.com/ipsix/arcsent/internal/storage"
)

func TestSchedulerIntegration(t *testing.T) {
	mgr := scanner.NewManager()
	disk := &system.DiskUsage{}
	if err := disk.Init(map[string]interface{}{
		"path":         t.TempDir(),
		"warn_percent": float64(90),
		"crit_percent": float64(95),
	}); err != nil {
		t.Fatalf("disk init: %v", err)
	}
	if err := mgr.Register(disk); err != nil {
		t.Fatalf("register: %v", err)
	}

	store, err := storage.NewBadgerStore(filepath.Join(t.TempDir(), "badger"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()
	baseline := detection.NewManager(store)
	results := state.NewResultCache(10)

	s := New(logging.New("text"), mgr)
	s.SetOnResult(func(result scanner.Result) {
		results.Add(result)
		for key, raw := range result.Metadata {
			if value, ok := toFloat(raw); ok {
				_, _ = baseline.Update(result.ScannerName, key, value)
			}
		}
	})

	if err := s.AddJob(JobConfig{
		Name:       "disk",
		Plugin:     "system.disk_usage",
		Schedule:   "20ms",
		Timeout:    1 * time.Second,
		RunOnStart: true,
	}); err != nil {
		t.Fatalf("add job: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	s.Start(ctx)
	<-ctx.Done()
	s.Stop()

	if len(results.Latest()) == 0 {
		t.Fatalf("expected results to be recorded")
	}
	baselines, err := baseline.List()
	if err != nil {
		t.Fatalf("baseline list: %v", err)
	}
	if len(baselines) == 0 {
		t.Fatalf("expected baselines to be updated")
	}
}

func toFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint:
		return float64(v), true
	default:
		return 0, false
	}
}
