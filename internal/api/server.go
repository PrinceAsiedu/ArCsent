package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/detection"
	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
	"github.com/ipsix/arcsent/internal/scheduler"
	"github.com/ipsix/arcsent/internal/state"
)

type Server struct {
	cfg      config.APIConfig
	logger   *logging.Logger
	server   *http.Server
	mgr      *scanner.Manager
	sched    *scheduler.Scheduler
	results  *state.ResultCache
	baseline *detection.Manager
	handler  http.Handler
}

func New(cfg config.APIConfig, logger *logging.Logger, mgr *scanner.Manager, sched *scheduler.Scheduler, results *state.ResultCache, baseline *detection.Manager) *Server {
	return &Server{
		cfg:      cfg,
		logger:   logger,
		mgr:      mgr,
		sched:    sched,
		results:  results,
		baseline: baseline,
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
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plugins": s.mgr.List(),
		"jobs":    s.sched.ListJobs(),
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

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
