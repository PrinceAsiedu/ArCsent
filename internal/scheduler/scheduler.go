package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
	"github.com/ipsix/arcsent/internal/storage"
	"github.com/robfig/cron/v3"
)

type JobConfig struct {
	Name         string
	Plugin       string
	Schedule     string
	Timeout      time.Duration
	MaxRetries   int
	RetryBackoff time.Duration
	RetryMax     time.Duration
	AllowOverlap bool
	RunOnStart   bool
}

type Scheduler struct {
	logger     *logging.Logger
	mgr        *scanner.Manager
	mu         sync.Mutex
	jobs       map[string]*job
	onResult   func(scanner.Result)
	stateStore storage.Store
}

func New(logger *logging.Logger, mgr *scanner.Manager) *Scheduler {
	return &Scheduler{
		logger: logger,
		mgr:    mgr,
		jobs:   make(map[string]*job),
	}
}

func (s *Scheduler) WithStateStore(store storage.Store) {
	s.stateStore = store
}

func (s *Scheduler) SetOnResult(fn func(scanner.Result)) {
	s.onResult = fn
}

func (s *Scheduler) ListJobs() []JobConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]JobConfig, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, j.cfg)
	}
	return out
}

func (s *Scheduler) JobState(name string) (JobState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[name]
	if !ok {
		return JobState{}, false
	}
	return j.state, true
}

func (s *Scheduler) NextRun(name string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[name]
	if !ok {
		return time.Time{}, false
	}
	return j.nextRun, true
}

func (s *Scheduler) AddJob(cfg JobConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if cfg.Plugin == "" {
		return fmt.Errorf("job plugin is required")
	}
	spec, err := parseSchedule(cfg.Schedule)
	if err != nil {
		return err
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Minute
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = 2 * time.Second
	}
	if cfg.RetryMax <= 0 {
		cfg.RetryMax = 30 * time.Second
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[cfg.Name]; exists {
		return fmt.Errorf("job %q already exists", cfg.Name)
	}

	j := &job{
		cfg:   cfg,
		spec:  spec,
		stop:  make(chan struct{}),
		state: JobState{},
	}
	s.loadState(j)
	j.nextRun = s.computeNextRun(j, time.Now())
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

func (s *Scheduler) ReplaceJobs(ctx context.Context, configs []JobConfig) error {
	s.mu.Lock()
	for _, j := range s.jobs {
		if j.started {
			close(j.stop)
			j.started = false
		}
	}
	s.jobs = make(map[string]*job)
	s.mu.Unlock()

	for _, cfg := range configs {
		if err := s.AddJob(cfg); err != nil {
			return err
		}
	}

	s.Start(ctx)
	return nil
}

func (s *Scheduler) runJob(ctx context.Context, j *job) {
	for {
		if j.cfg.RunOnStart && j.state.LastRun.IsZero() {
			s.executeJob(ctx, j)
		}

		next := j.nextRun
		if next.IsZero() {
			next = s.computeNextRun(j, time.Now())
			j.nextRun = next
		}
		wait := time.Until(next)
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-j.stop:
			timer.Stop()
			return
		case <-timer.C:
			s.executeJob(ctx, j)
			j.nextRun = s.computeNextRun(j, time.Now())
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

	result, err := s.executeWithRetry(ctx, j)
	if err != nil {
		s.logger.Error("job failed",
			logging.Field{Key: "job", Value: j.cfg.Name},
			logging.Field{Key: "error", Value: err.Error()},
		)
		s.updateState(j, scanner.StatusFailed, err)
		return
	}
	if result == nil {
		s.logger.Warn("job returned nil result", logging.Field{Key: "job", Value: j.cfg.Name})
		s.updateState(j, scanner.StatusFailed, fmt.Errorf("nil result"))
		return
	}
	s.updateState(j, result.Status, nil)
	if s.onResult != nil {
		s.onResult(*result)
	}
}

func (s *Scheduler) RunOnce(ctx context.Context, pluginName string, timeout time.Duration) (*scanner.Result, error) {
	p, err := s.mgr.Get(pluginName)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("runonce panic recovered",
				logging.Field{Key: "plugin", Value: pluginName},
				logging.Field{Key: "panic", Value: r},
			)
		}
	}()

	result, err := p.Run(runCtx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("plugin returned nil result")
	}
	finished := time.Now()
	result.StartedAt = started
	result.FinishedAt = finished
	result.Duration = finished.Sub(started)
	if s.onResult != nil {
		s.onResult(*result)
	}
	return result, nil
}

type job struct {
	cfg     JobConfig
	spec    scheduleSpec
	stop    chan struct{}
	running atomic.Bool
	started bool
	state   JobState
	nextRun time.Time
}

type scheduleKind int

const (
	scheduleInterval scheduleKind = iota
	scheduleCron
)

type scheduleSpec struct {
	kind     scheduleKind
	interval time.Duration
	cron     cron.Schedule
	raw      string
}

func (s scheduleSpec) Next(from time.Time) time.Time {
	if s.kind == scheduleInterval {
		return from.Add(s.interval)
	}
	return s.cron.Next(from)
}

func parseSchedule(expr string) (scheduleSpec, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return scheduleSpec{}, fmt.Errorf("schedule is required")
	}
	if strings.HasPrefix(expr, "@every ") {
		expr = strings.TrimPrefix(expr, "@every ")
	}
	interval, err := time.ParseDuration(expr)
	if err != nil {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		sched, cronErr := parser.Parse(expr)
		if cronErr != nil {
			return scheduleSpec{}, fmt.Errorf("unsupported schedule %q (use duration or cron)", expr)
		}
		return scheduleSpec{kind: scheduleCron, cron: sched, raw: expr}, nil
	}
	if interval <= 0 {
		return scheduleSpec{}, fmt.Errorf("schedule interval must be positive")
	}
	return scheduleSpec{kind: scheduleInterval, interval: interval, raw: expr}, nil
}

func (s *Scheduler) executeWithRetry(ctx context.Context, j *job) (*scanner.Result, error) {
	var lastErr error
	for attempt := 0; attempt <= j.cfg.MaxRetries; attempt++ {
		runCtx, cancel := context.WithTimeout(ctx, j.cfg.Timeout)
		started := time.Now()
		result, err := s.runOnce(runCtx, j)
		cancel()
		if err == nil && result != nil {
			finished := time.Now()
			result.StartedAt = started
			result.FinishedAt = finished
			result.Duration = finished.Sub(started)
			s.logger.Info("job completed",
				logging.Field{Key: "job", Value: j.cfg.Name},
				logging.Field{Key: "status", Value: result.Status},
				logging.Field{Key: "duration", Value: result.Duration.String()},
				logging.Field{Key: "findings", Value: len(result.Findings)},
			)
			return result, nil
		}
		lastErr = err
		if attempt < j.cfg.MaxRetries {
			backoff := j.cfg.RetryBackoff * time.Duration(1<<attempt)
			if backoff > j.cfg.RetryMax {
				backoff = j.cfg.RetryMax
			}
			time.Sleep(backoff)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("job failed")
	}
	return nil, lastErr
}

func (s *Scheduler) runOnce(ctx context.Context, j *job) (*scanner.Result, error) {
	p, err := s.mgr.Get(j.cfg.Plugin)
	if err != nil {
		s.logger.Error("plugin lookup failed", logging.Field{Key: "job", Value: j.cfg.Name}, logging.Field{Key: "error", Value: err.Error()})
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("job panic recovered",
				logging.Field{Key: "job", Value: j.cfg.Name},
				logging.Field{Key: "panic", Value: r},
				logging.Field{Key: "stack", Value: string(debug.Stack())},
			)
		}
	}()
	return p.Run(ctx)
}

type JobState struct {
	LastRun             time.Time      `json:"last_run"`
	LastSuccess         time.Time      `json:"last_success"`
	LastError           time.Time      `json:"last_error"`
	LastStatus          scanner.Status `json:"last_status"`
	LastErrorMessage    string         `json:"last_error_message"`
	ConsecutiveFailures int            `json:"consecutive_failures"`
}

func (s *Scheduler) loadState(j *job) {
	if s.stateStore == nil {
		return
	}
	raw, err := s.stateStore.Get("scheduler_state", j.cfg.Name)
	if err != nil {
		return
	}
	var state JobState
	if err := json.Unmarshal(raw, &state); err != nil {
		return
	}
	j.state = state
}

func (s *Scheduler) saveState(j *job) {
	if s.stateStore == nil {
		return
	}
	raw, err := json.Marshal(j.state)
	if err != nil {
		return
	}
	_ = s.stateStore.Put("scheduler_state", j.cfg.Name, raw)
}

func (s *Scheduler) updateState(j *job, status scanner.Status, err error) {
	now := time.Now().UTC()
	j.state.LastRun = now
	j.state.LastStatus = status
	if err != nil {
		j.state.LastError = now
		j.state.LastErrorMessage = err.Error()
		j.state.ConsecutiveFailures++
	} else {
		j.state.LastSuccess = now
		j.state.LastErrorMessage = ""
		j.state.ConsecutiveFailures = 0
	}
	s.saveState(j)
}

func (s *Scheduler) computeNextRun(j *job, now time.Time) time.Time {
	if !j.state.LastRun.IsZero() && j.spec.kind == scheduleInterval {
		next := j.state.LastRun.Add(j.spec.interval)
		if next.After(now) {
			return next
		}
	}
	return j.spec.Next(now)
}
