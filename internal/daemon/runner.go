package daemon

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/logging"
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
	if err := RequirePrivilegeDrop(r.cfg.Daemon.User, r.cfg.Daemon.Group); err != nil {
		return err
	}

	if err := DropPrivileges(r.cfg.Daemon.User, r.cfg.Daemon.Group); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	r.logger.Info("daemon started")

	go r.handleSignals(sigCh, cancel)

	<-ctx.Done()

	return r.shutdown(r.cfg.Daemon.ShutdownTimeoutDuration())
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
