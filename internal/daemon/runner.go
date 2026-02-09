package daemon

import (
	"context"
	"os"
	"os/signal"
	"strings"
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
	"github.com/ipsix/arcsent/internal/signatures"
	"github.com/ipsix/arcsent/internal/state"
	"github.com/ipsix/arcsent/internal/storage"
	"github.com/ipsix/arcsent/internal/webui"
)

type Runner struct {
	cfg        config.Config
	logger     *logging.Logger
	configPath string
}

func New(cfg config.Config, logger *logging.Logger, configPath string) *Runner {
	return &Runner{
		cfg:        cfg,
		logger:     logger,
		configPath: configPath,
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

	if r.cfg.Security.SelfIntegrity {
		if err := VerifySelfIntegrity(r.cfg.Security.ExpectedSHA256); err != nil {
			return err
		}
		r.logger.Info("self-integrity check passed")
	}

	alertEngine := alerting.New(r.logger, r.cfg.Alerting)
	channels, err := alerting.BuildChannels(r.cfg.Alerting, r.logger)
	if err != nil {
		r.logger.Error("alert channel setup failed", logging.Field{Key: "error", Value: err.Error()})
		channels = []alerting.Channel{alerting.NewLogChannel(r.logger)}
	}
	for _, ch := range channels {
		alertEngine.Register(ch)
	}

	manager := scanner.NewManager()
	plugins := []scanner.Plugin{
		&system.CPUMemory{},
		&system.DiskUsage{},
		&system.FileIntegrity{},
		&system.LoadAverage{},
		&system.ProcessMonitor{},
		&system.AuthLogMonitor{},
		&system.NetworkListeners{},
		&system.Uptime{},
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

	store, err := storage.NewBadgerStoreWithKey(r.cfg.Storage.DBPath, r.cfg.Storage.EncryptionKeyBase64)
	if err != nil {
		return err
	}
	defer store.Close()

	signatureStore := signatures.NewStore(store)
	signatureUpdater := signatures.NewUpdater(signatures.Config{
		Enabled:          r.cfg.Signatures.Enabled,
		UpdateInterval:   r.cfg.Signatures.UpdateIntervalDuration(),
		Sources:          append([]string{}, r.cfg.Signatures.Sources...),
		CacheDir:         r.cfg.Signatures.CacheDir,
		AirgapImportPath: r.cfg.Signatures.AirgapImportPath,
		SourceURLs:       r.cfg.Signatures.SourceURLOverrides(),
	}, signatureStore, r.logger)

	baselineMgr := detection.NewManager(store)
	resultsStore := storage.NewResultsStore(store)
	ruleEngine := detection.NewRuleEngine(buildRules(r.cfg.Detection.Rules))
	correlator := detection.NewCorrelator(r.cfg.Detection.CorrelationWindowDuration(), r.cfg.Detection.CorrelationMinScanners, r.cfg.Detection.CorrelationCooldownDuration())
	resultCache := state.NewResultCache(50)
	if r.cfg.Storage.RetentionDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -r.cfg.Storage.RetentionDays)
		_ = resultsStore.PruneOlderThan(cutoff)
		_ = baselineMgr.PruneOlderThan(cutoff)
	}
	sched := scheduler.New(r.logger, manager)
	sched.WithStateStore(store)
	sched.SetOnResult(func(result scanner.Result) {
		for key, raw := range result.Metadata {
			if value, ok := toFloat(raw); ok {
				if drift, _, err := baselineMgr.DetectDrift(result.ScannerName, key, value, r.cfg.Detection.DriftConsecutive); err == nil && drift {
					result.Findings = append(result.Findings, scanner.Finding{
						ID:          "metric_drift",
						Severity:    scanner.SeverityHigh,
						Category:    "drift",
						Description: "Metric drift detected beyond baseline.",
						Evidence: map[string]interface{}{
							"metric": key,
							"value":  value,
						},
						Remediation: "Review system changes affecting this metric.",
					})
				}
				_, _ = baselineMgr.Update(result.ScannerName, key, value)
			}
		}

		ruleFindings := ruleEngine.Evaluate(result)
		result.Findings = append(result.Findings, ruleFindings...)

		corrFindings := correlator.Add(result)
		result.Findings = append(result.Findings, corrFindings...)

		resultCache.Add(result)
		_ = resultsStore.Save(result)
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
			MaxRetries:   sc.MaxRetries,
			RetryBackoff: sc.RetryBackoffDuration(),
			RetryMax:     sc.RetryMaxDuration(),
			AllowOverlap: sc.AllowOverlap,
			RunOnStart:   sc.RunOnStart,
		}); err != nil {
			r.logger.Error("failed to schedule job", logging.Field{Key: "job", Value: sc.Name}, logging.Field{Key: "error", Value: err.Error()})
		}
	}

	sched.Start(ctx)

	go signatureUpdater.Start(ctx)

	apiServer := api.New(r.cfg.API, r.logger, manager, sched, resultCache, baselineMgr, resultsStore, signatureStore, signatureUpdater)
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

	reload := func() {
		if r.configPath == "" {
			r.logger.Warn("config reload skipped: no config path set")
			return
		}
		newCfg, err := config.Load(r.configPath)
		if err != nil {
			r.logger.Error("config reload failed", logging.Field{Key: "error", Value: err.Error()})
			return
		}

		oldCfg := r.cfg
		r.cfg = newCfg

		if newCfg.API.BindAddr != oldCfg.API.BindAddr {
			r.logger.Warn("api.bind_addr change requires restart", logging.Field{Key: "old", Value: oldCfg.API.BindAddr}, logging.Field{Key: "new", Value: newCfg.API.BindAddr})
		}
		if newCfg.WebUI.BindAddr != oldCfg.WebUI.BindAddr {
			r.logger.Warn("web_ui.bind_addr change requires restart", logging.Field{Key: "old", Value: oldCfg.WebUI.BindAddr}, logging.Field{Key: "new", Value: newCfg.WebUI.BindAddr})
		}
		if newCfg.Storage.DBPath != oldCfg.Storage.DBPath {
			r.logger.Warn("storage.db_path change requires restart", logging.Field{Key: "old", Value: oldCfg.Storage.DBPath}, logging.Field{Key: "new", Value: newCfg.Storage.DBPath})
		}

		ruleEngine = detection.NewRuleEngine(buildRules(newCfg.Detection.Rules))
		correlator = detection.NewCorrelator(newCfg.Detection.CorrelationWindowDuration(), newCfg.Detection.CorrelationMinScanners, newCfg.Detection.CorrelationCooldownDuration())

		newAlertEngine := alerting.New(r.logger, newCfg.Alerting)
		newChannels, err := alerting.BuildChannels(newCfg.Alerting, r.logger)
		if err != nil {
			r.logger.Error("alert channel setup failed", logging.Field{Key: "error", Value: err.Error()})
			newChannels = []alerting.Channel{alerting.NewLogChannel(r.logger)}
		}
		for _, ch := range newChannels {
			newAlertEngine.Register(ch)
		}
		alertEngine = newAlertEngine

		signatureUpdater.UpdateConfig(signatures.Config{
			Enabled:          newCfg.Signatures.Enabled,
			UpdateInterval:   newCfg.Signatures.UpdateIntervalDuration(),
			Sources:          append([]string{}, newCfg.Signatures.Sources...),
			CacheDir:         newCfg.Signatures.CacheDir,
			AirgapImportPath: newCfg.Signatures.AirgapImportPath,
			SourceURLs:       newCfg.Signatures.SourceURLOverrides(),
		})

		apiServer.UpdateConfig(newCfg.API)
		webServer.UpdateConfig(newCfg.WebUI)

		for _, sc := range newCfg.Scanners {
			if sc.Config == nil {
				sc.Config = map[string]interface{}{}
			}
			p, err := manager.Get(sc.Plugin)
			if err != nil {
				continue
			}
			if err := p.Init(sc.Config); err != nil {
				r.logger.Error("plugin init failed", logging.Field{Key: "plugin", Value: sc.Plugin}, logging.Field{Key: "error", Value: err.Error()})
			}
		}

		jobs := []scheduler.JobConfig{}
		for _, sc := range newCfg.Scanners {
			if !sc.Enabled {
				continue
			}
			jobs = append(jobs, scheduler.JobConfig{
				Name:         sc.Name,
				Plugin:       sc.Plugin,
				Schedule:     sc.Schedule,
				Timeout:      sc.TimeoutDuration(),
				MaxRetries:   sc.MaxRetries,
				RetryBackoff: sc.RetryBackoffDuration(),
				RetryMax:     sc.RetryMaxDuration(),
				AllowOverlap: sc.AllowOverlap,
				RunOnStart:   sc.RunOnStart,
			})
		}
		if err := sched.ReplaceJobs(ctx, jobs); err != nil {
			r.logger.Error("scheduler reload failed", logging.Field{Key: "error", Value: err.Error()})
		}

		r.logger.Info("config reload complete")
	}

	go r.handleSignals(sigCh, cancel, reload)

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

func buildRules(rules []config.RuleConfig) []detection.Rule {
	out := make([]detection.Rule, 0, len(rules))
	for _, rule := range rules {
		out = append(out, detection.Rule{
			Name:        rule.Name,
			Scanner:     rule.Scanner,
			Metric:      rule.Metric,
			Operator:    rule.Operator,
			Threshold:   rule.Threshold,
			Severity:    parseSeverity(rule.Severity),
			Description: rule.Description,
		})
	}
	return out
}

func parseSeverity(value string) scanner.Severity {
	switch strings.ToLower(value) {
	case "low":
		return scanner.SeverityLow
	case "medium":
		return scanner.SeverityMedium
	case "high":
		return scanner.SeverityHigh
	case "critical":
		return scanner.SeverityCritical
	default:
		return scanner.SeverityInfo
	}
}

func (r *Runner) handleSignals(sigCh <-chan os.Signal, cancel context.CancelFunc, reload func()) {
	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			r.logger.Info("config reload requested")
			reload()
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
