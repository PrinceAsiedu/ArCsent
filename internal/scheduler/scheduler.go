package scheduler

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
)

type JobConfig struct {
	Name         string
	Plugin       string
	Schedule     string
	Timeout      time.Duration
	AllowOverlap bool
	RunOnStart   bool
}

type Scheduler struct {
	logger *logging.Logger
	mgr    *scanner.Manager
	mu     sync.Mutex
	jobs   map[string]*job
}

func New(logger *logging.Logger, mgr *scanner.Manager) *Scheduler {
	return &Scheduler{
		logger: logger,
		mgr:    mgr,
		jobs:   make(map[string]*job),
	}
}

func (s *Scheduler) AddJob(cfg JobConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if cfg.Plugin == "" {
		return fmt.Errorf("job plugin is required")
	}
	interval, err := parseSchedule(cfg.Schedule)
	if err != nil {
		return err
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Minute
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[cfg.Name]; exists {
		return fmt.Errorf("job %q already exists", cfg.Name)
	}

	j := &job{
		cfg:      cfg,
		interval: interval,
		stop:     make(chan struct{}),
	}
	s.jobs[cfg.Name] = j
	return nil
}

func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.started {
			continue
		}
		j.started = true
		go s.runJob(ctx, j)
	}
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.started {
			close(j.stop)
			j.started = false
		}
	}
}

func (s *Scheduler) runJob(ctx context.Context, j *job) {
	if j.cfg.RunOnStart {
		s.executeJob(ctx, j)
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-j.stop:
			return
		case <-ticker.C:
			s.executeJob(ctx, j)
		}
	}
}

func (s *Scheduler) executeJob(ctx context.Context, j *job) {
	if !j.cfg.AllowOverlap {
		if !j.running.CompareAndSwap(false, true) {
			s.logger.Warn("job skipped due to overlap", logging.Field{Key: "job", Value: j.cfg.Name})
			return
		}
		defer j.running.Store(false)
	}

	p, err := s.mgr.Get(j.cfg.Plugin)
	if err != nil {
		s.logger.Error("plugin lookup failed", logging.Field{Key: "job", Value: j.cfg.Name}, logging.Field{Key: "error", Value: err.Error()})
		return
	}

	runCtx, cancel := context.WithTimeout(ctx, j.cfg.Timeout)
	defer cancel()

	started := time.Now()
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("job panic recovered",
				logging.Field{Key: "job", Value: j.cfg.Name},
				logging.Field{Key: "panic", Value: r},
				logging.Field{Key: "stack", Value: string(debug.Stack())},
			)
		}
	}()

	result, err := p.Run(runCtx)
	finished := time.Now()

	if err != nil {
		s.logger.Error("job failed",
			logging.Field{Key: "job", Value: j.cfg.Name},
			logging.Field{Key: "error", Value: err.Error()},
			logging.Field{Key: "duration", Value: finished.Sub(started).String()},
		)
		return
	}

	if result == nil {
		s.logger.Warn("job returned nil result", logging.Field{Key: "job", Value: j.cfg.Name})
		return
	}

	result.StartedAt = started
	result.FinishedAt = finished
	result.Duration = finished.Sub(started)

	s.logger.Info("job completed",
		logging.Field{Key: "job", Value: j.cfg.Name},
		logging.Field{Key: "status", Value: result.Status},
		logging.Field{Key: "duration", Value: result.Duration.String()},
		logging.Field{Key: "findings", Value: len(result.Findings)},
	)
}

type job struct {
	cfg      JobConfig
	interval time.Duration
	stop     chan struct{}
	running  atomic.Bool
	started  bool
}

func parseSchedule(expr string) (time.Duration, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, fmt.Errorf("schedule is required")
	}
	if strings.HasPrefix(expr, "@every ") {
		expr = strings.TrimPrefix(expr, "@every ")
	}
	interval, err := time.ParseDuration(expr)
	if err != nil {
		return 0, fmt.Errorf("unsupported schedule %q (use duration or @every <duration>)", expr)
	}
	if interval <= 0 {
		return 0, fmt.Errorf("schedule interval must be positive")
	}
	return interval, nil
}
