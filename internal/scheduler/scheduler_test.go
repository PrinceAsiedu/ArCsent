package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
	"github.com/ipsix/arcsent/internal/storage"
)

type testPlugin struct {
	name   string
	delay  time.Duration
	panic  bool
	fail   bool
	called int
}

func (t *testPlugin) Name() string                        { return t.name }
func (t *testPlugin) Init(_ map[string]interface{}) error { return nil }
func (t *testPlugin) Halt(_ context.Context) error        { return nil }
func (t *testPlugin) Run(ctx context.Context) (*scanner.Result, error) {
	t.called++
	if t.panic {
		panic("boom")
	}
	if t.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(t.delay):
		}
	}
	if t.fail {
		return nil, errors.New("failed")
	}
	return &scanner.Result{ScannerName: t.name, Status: scanner.StatusSuccess}, nil
}

func TestParseSchedule(t *testing.T) {
	if _, err := parseSchedule("@every 1s"); err != nil {
		t.Fatalf("expected schedule to parse: %v", err)
	}
	if _, err := parseSchedule("1s"); err != nil {
		t.Fatalf("expected schedule to parse: %v", err)
	}
	if _, err := parseSchedule("*/5 * * * *"); err != nil {
		t.Fatalf("expected cron expression to parse: %v", err)
	}
}

func TestOverlapPrevention(t *testing.T) {
	mgr := scanner.NewManager()
	p := &testPlugin{name: "test", delay: 50 * time.Millisecond}
	if err := mgr.Register(p); err != nil {
		t.Fatalf("register: %v", err)
	}
	s := New(logging.New("text"), mgr)
	if err := s.AddJob(JobConfig{
		Name:       "job",
		Plugin:     "test",
		Schedule:   "10ms",
		Timeout:    200 * time.Millisecond,
		RunOnStart: true,
	}); err != nil {
		t.Fatalf("add job: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	s.Start(ctx)
	<-ctx.Done()
	s.Stop()

	if p.called < 1 {
		t.Fatalf("expected job to run at least once")
	}
}

func TestPanicRecovery(t *testing.T) {
	mgr := scanner.NewManager()
	p := &testPlugin{name: "panic", panic: true}
	if err := mgr.Register(p); err != nil {
		t.Fatalf("register: %v", err)
	}
	s := New(logging.New("text"), mgr)
	if err := s.AddJob(JobConfig{
		Name:       "job",
		Plugin:     "panic",
		Schedule:   "50ms",
		Timeout:    50 * time.Millisecond,
		RunOnStart: true,
	}); err != nil {
		t.Fatalf("add job: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	s.Start(ctx)
	<-ctx.Done()
	s.Stop()
}

func TestRetryBackoff(t *testing.T) {
	mgr := scanner.NewManager()
	p := &testPlugin{name: "retry", fail: true}
	if err := mgr.Register(p); err != nil {
		t.Fatalf("register: %v", err)
	}
	s := New(logging.New("text"), mgr)
	if err := s.AddJob(JobConfig{
		Name:         "job",
		Plugin:       "retry",
		Schedule:     "20ms",
		Timeout:      20 * time.Millisecond,
		MaxRetries:   2,
		RetryBackoff: 5 * time.Millisecond,
		RetryMax:     10 * time.Millisecond,
		RunOnStart:   true,
	}); err != nil {
		t.Fatalf("add job: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	s.Start(ctx)
	<-ctx.Done()
	s.Stop()

	if p.called < 2 {
		t.Fatalf("expected retries, got %d calls", p.called)
	}
}

func TestPersistentStateRespectsInterval(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewBadgerStore(filepath.Join(dir, "badger"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer store.Close()

	mgr := scanner.NewManager()
	p := &testPlugin{name: "persist"}
	if err := mgr.Register(p); err != nil {
		t.Fatalf("register: %v", err)
	}

	s := New(logging.New("text"), mgr)
	s.WithStateStore(store)

	state := JobState{
		LastRun: time.Now().Add(-30 * time.Second),
	}
	raw, _ := json.Marshal(state)
	if err := store.Put("scheduler_state", "job", raw); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	if err := s.AddJob(JobConfig{
		Name:     "job",
		Plugin:   "persist",
		Schedule: "2m",
	}); err != nil {
		t.Fatalf("add job: %v", err)
	}

	next, ok := s.NextRun("job")
	if !ok {
		t.Fatalf("expected next run")
	}
	if time.Until(next) < time.Minute {
		t.Fatalf("expected next run to respect last run interval, got %v", next)
	}
}
