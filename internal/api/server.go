package api

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/detection"
	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
	"github.com/ipsix/arcsent/internal/scheduler"
	"github.com/ipsix/arcsent/internal/signatures"
	"github.com/ipsix/arcsent/internal/state"
	"github.com/ipsix/arcsent/internal/storage"
)

type Server struct {
	cfg          config.APIConfig
	logger       *logging.Logger
	server       *http.Server
	mgr          *scanner.Manager
	sched        *scheduler.Scheduler
	results      *state.ResultCache
	baseline     *detection.Manager
	handler      http.Handler
	resultsStore *storage.ResultsStore
	sigStore     *signatures.Store
	sigUpdater   *signatures.Updater
}

func New(cfg config.APIConfig, logger *logging.Logger, mgr *scanner.Manager, sched *scheduler.Scheduler, results *state.ResultCache, baseline *detection.Manager, resultsStore *storage.ResultsStore, sigStore *signatures.Store, sigUpdater *signatures.Updater) *Server {
	return &Server{
		cfg:          cfg,
		logger:       logger,
		mgr:          mgr,
		sched:        sched,
		results:      results,
		baseline:     baseline,
		resultsStore: resultsStore,
		sigStore:     sigStore,
		sigUpdater:   sigUpdater,
	}
}

func (s *Server) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}

	s.handler = s.buildHandler()
	s.server = &http.Server{
		Addr:              s.cfg.BindAddr,
		Handler:           s.handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.logger.Info("api server starting", logging.Field{Key: "addr", Value: s.cfg.BindAddr})
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) UpdateConfig(cfg config.APIConfig) {
	s.cfg = cfg
}

func (s *Server) buildHandler() http.Handler {
	mux := http.NewServeMux()
	register := func(path string, handler http.HandlerFunc) {
		mux.HandleFunc(path, s.withAuth(handler))
		mux.HandleFunc("/api"+path, s.withAuth(handler))
	}
	register("/health", s.handleHealth)
	register("/status", s.handleStatus)
	register("/scanners", s.handleScanners)
	register("/scanners/trigger/", s.handleTrigger)
	register("/results/latest", s.handleResultsLatest)
	register("/results/history", s.handleResultsHistory)
	register("/findings", s.handleFindings)
	register("/baselines", s.handleBaselines)
	register("/export/results", s.handleExportResults)
	register("/export/baselines", s.handleExportBaselines)
	register("/signatures/status", s.handleSignaturesStatus)
	register("/signatures/update", s.handleSignaturesUpdate)
	register("/metrics", s.handleMetrics)
	return mux
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	s.logger.Info("api server stopping")
	return s.server.Shutdown(ctx)
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if s.cfg.AuthToken != "" && token != s.cfg.AuthToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "running",
		"running_as_root": os.Geteuid() == 0,
	})
}

func (s *Server) handleScanners(w http.ResponseWriter, _ *http.Request) {
	states := map[string]interface{}{}
	for _, job := range s.sched.ListJobs() {
		state, ok := s.sched.JobState(job.Name)
		if ok {
			next, _ := s.sched.NextRun(job.Name)
			states[job.Name] = map[string]interface{}{
				"state":    state,
				"next_run": next,
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plugins": s.mgr.List(),
		"jobs":    s.sched.ListJobs(),
		"states":  states,
	})
}

func (s *Server) handleTrigger(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/scanners/trigger/")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scanner name required"})
		return
	}
	result, err := s.sched.RunOnce(r.Context(), name, 0)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleResultsLatest(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.results.Latest())
}

func (s *Server) handleResultsHistory(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.results.History())
}

func (s *Server) handleFindings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.results.FindingsHistory())
}

func (s *Server) handleBaselines(w http.ResponseWriter, _ *http.Request) {
	if s.baseline == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	baselines, err := s.baseline.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, baselines)
}

func (s *Server) handleExportResults(w http.ResponseWriter, r *http.Request) {
	if s.resultsStore == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	results, err := s.resultsStore.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	format := r.URL.Query().Get("format")
	if format == "csv" {
		writeCSV(w, []string{"scanner", "status", "findings", "started_at", "finished_at", "duration"}, resultsToRows(results))
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleExportBaselines(w http.ResponseWriter, r *http.Request) {
	if s.baseline == nil {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	baselines, err := s.baseline.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	format := r.URL.Query().Get("format")
	if format == "csv" {
		writeCSV(w, []string{"scanner", "metric", "count", "mean", "min", "max", "updated_at"}, baselinesToRows(baselines))
		return
	}
	writeJSON(w, http.StatusOK, baselines)
}

func (s *Server) handleSignaturesStatus(w http.ResponseWriter, _ *http.Request) {
	if s.sigStore == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{})
		return
	}
	status, err := s.sigStore.LoadStatus()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleSignaturesUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	if s.sigUpdater == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "signatures updater not configured"})
		return
	}
	status, err := s.sigUpdater.Trigger(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(s.buildMetrics()))
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) buildMetrics() string {
	builder := strings.Builder{}
	writeGauge := func(name string, value interface{}) {
		builder.WriteString(name)
		builder.WriteString(" ")
		builder.WriteString(fmt.Sprintf("%v", value))
		builder.WriteString("\n")
	}

	writeGauge("arcsent_up", 1)
	if os.Geteuid() == 0 {
		writeGauge("arcsent_running_as_root", 1)
	} else {
		writeGauge("arcsent_running_as_root", 0)
	}

	if s.mgr != nil {
		writeGauge("arcsent_plugins_total", len(s.mgr.List()))
	}
	if s.sched != nil {
		writeGauge("arcsent_jobs_total", len(s.sched.ListJobs()))
	}
	if s.results != nil {
		writeGauge("arcsent_results_total", len(s.results.History()))
		writeGauge("arcsent_findings_total", len(s.results.FindingsHistory()))
	}

	if s.sigStore != nil {
		if status, err := s.sigStore.LoadStatus(); err == nil {
			writeGauge("arcsent_signatures_sources_total", len(status.Sources))
			ok := 0
			for _, src := range status.Sources {
				if src.Error == "" {
					ok++
				}
			}
			writeGauge("arcsent_signatures_sources_ok_total", ok)
			if !status.LastRun.IsZero() {
				writeGauge("arcsent_signatures_last_run_timestamp_seconds", status.LastRun.Unix())
			}
			if !status.NextRun.IsZero() {
				writeGauge("arcsent_signatures_next_run_timestamp_seconds", status.NextRun.Unix())
			}
		}
	}

	return builder.String()
}

func writeCSV(w http.ResponseWriter, headers []string, rows [][]string) {
	w.Header().Set("Content-Type", "text/csv")
	w.WriteHeader(http.StatusOK)
	writer := csv.NewWriter(w)
	_ = writer.Write(headers)
	for _, row := range rows {
		_ = writer.Write(row)
	}
	writer.Flush()
}

func resultsToRows(results []scanner.Result) [][]string {
	rows := make([][]string, 0, len(results))
	for _, res := range results {
		rows = append(rows, []string{
			res.ScannerName,
			string(res.Status),
			fmt.Sprintf("%d", len(res.Findings)),
			res.StartedAt.Format(time.RFC3339),
			res.FinishedAt.Format(time.RFC3339),
			res.Duration.String(),
		})
	}
	return rows
}

func baselinesToRows(baselines []detection.Baseline) [][]string {
	rows := make([][]string, 0, len(baselines))
	for _, b := range baselines {
		rows = append(rows, []string{
			b.ScannerName,
			b.Metric,
			fmt.Sprintf("%d", b.Count),
			fmt.Sprintf("%f", b.Mean),
			fmt.Sprintf("%f", b.Min),
			fmt.Sprintf("%f", b.Max),
			b.UpdatedAt.Format(time.RFC3339),
		})
	}
	return rows
}
