package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
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
	if _, err := parseSchedule("*/5 * * * *"); err == nil {
		t.Fatalf("expected cron expression to be rejected in phase 3")
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
