package daemon

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ipsix/arcsent/internal/alerting"
	"github.com/ipsix/arcsent/internal/api"
	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/detection"
	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/plugins/system"
	"github.com/ipsix/arcsent/internal/scanner"
	"github.com/ipsix/arcsent/internal/scheduler"
	"github.com/ipsix/arcsent/internal/state"
	"github.com/ipsix/arcsent/internal/storage"
	"github.com/ipsix/arcsent/internal/webui"
)

type Runner struct {
	cfg    config.Config
	logger *logging.Logger
}

func New(cfg config.Config, logger *logging.Logger) *Runner {
	return &Runner{
		cfg:    cfg,
		logger: logger,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	if r.cfg.Daemon.DropPrivileges {
		if err := RequirePrivilegeDrop(r.cfg.Daemon.User, r.cfg.Daemon.Group); err != nil {
			return err
		}
		if err := DropPrivileges(r.cfg.Daemon.User, r.cfg.Daemon.Group); err != nil {
			return err
		}
	} else {
		r.logger.Warn("running without privilege drop")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	r.logger.Info("daemon started")

	alertEngine := alerting.New(r.logger, 5*time.Minute)
	alertEngine.Register(alerting.NewLogChannel(r.logger))

	manager := scanner.NewManager()
	plugins := []scanner.Plugin{
		&system.DiskUsage{},
		&system.FileIntegrity{},
		&system.ProcessMonitor{},
		&system.AuthLogMonitor{},
		&system.NetworkListeners{},
	}
	for _, plugin := range plugins {
		if err := manager.Register(plugin); err != nil {
			r.logger.Error("plugin register failed", logging.Field{Key: "error", Value: err.Error()})
		}
	}
	for _, sc := range r.cfg.Scanners {
		if sc.Config == nil {
			sc.Config = map[string]interface{}{}
		}
		p, err := manager.Get(sc.Plugin)
		if err != nil {
			r.logger.Error("plugin not found", logging.Field{Key: "plugin", Value: sc.Plugin})
			continue
		}
		if err := p.Init(sc.Config); err != nil {
			r.logger.Error("plugin init failed", logging.Field{Key: "plugin", Value: sc.Plugin}, logging.Field{Key: "error", Value: err.Error()})
		}
	}

	store, err := storage.NewBadgerStore(r.cfg.Storage.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	baselineMgr := detection.NewManager(store)
	resultCache := state.NewResultCache(50)
	sched := scheduler.New(r.logger, manager)
	sched.SetOnResult(func(result scanner.Result) {
		resultCache.Add(result)
		for key, raw := range result.Metadata {
			if value, ok := toFloat(raw); ok {
				_, _ = baselineMgr.Update(result.ScannerName, key, value)
			}
		}
		if len(result.Findings) == 0 {
			return
		}
		for _, finding := range result.Findings {
			alertEngine.Send(alerting.Alert{
				ScannerName: result.ScannerName,
				Severity:    finding.Severity,
				Finding:     finding,
				Reason:      "finding_detected",
			})
		}
	})

	for _, sc := range r.cfg.Scanners {
		if !sc.Enabled {
			continue
		}
		if err := sched.AddJob(scheduler.JobConfig{
			Name:         sc.Name,
			Plugin:       sc.Plugin,
			Schedule:     sc.Schedule,
			Timeout:      sc.TimeoutDuration(),
			AllowOverlap: sc.AllowOverlap,
			RunOnStart:   sc.RunOnStart,
		}); err != nil {
			r.logger.Error("failed to schedule job", logging.Field{Key: "job", Value: sc.Name}, logging.Field{Key: "error", Value: err.Error()})
		}
	}

	sched.Start(ctx)

	apiServer := api.New(r.cfg.API, r.logger, manager, sched, resultCache, baselineMgr)
	go func() {
		if err := apiServer.Start(ctx); err != nil {
			r.logger.Error("api server exited", logging.Field{Key: "error", Value: err.Error()})
		}
	}()

	webServer := webui.New(r.cfg.WebUI, r.cfg.API.BindAddr, r.logger)
	go func() {
		if err := webServer.Start(ctx); err != nil {
			r.logger.Error("web ui exited", logging.Field{Key: "error", Value: err.Error()})
		}
	}()

	go r.handleSignals(sigCh, cancel)

	<-ctx.Done()

	_ = apiServer.Shutdown(context.Background())
	_ = webServer.Shutdown(context.Background())

	return r.shutdown(r.cfg.Daemon.ShutdownTimeoutDuration())
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

func (r *Runner) handleSignals(sigCh <-chan os.Signal, cancel context.CancelFunc) {
	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			r.logger.Info("config reload requested")
		case syscall.SIGINT, syscall.SIGTERM:
			r.logger.Warn("shutdown signal received", logging.Field{Key: "signal", Value: sig.String()})
			cancel()
			return
		default:
			r.logger.Warn("unexpected signal received", logging.Field{Key: "signal", Value: sig.String()})
		}
	}
}

func (r *Runner) shutdown(timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	r.logger.Info("shutdown starting", logging.Field{Key: "timeout", Value: timeout.String()})
	r.logger.Info("shutdown complete")
	return nil
}
