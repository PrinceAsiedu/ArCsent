package webui

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/logging"
)

//go:embed dist/*
var content embed.FS

type Server struct {
	cfg     config.WebUIConfig
	logger  *logging.Logger
	server  *http.Server
	apiURL  *url.URL
	handler http.Handler
}

func New(cfg config.WebUIConfig, apiAddr string, logger *logging.Logger) *Server {
	var parsed *url.URL
	if apiAddr != "" {
		parsed, _ = url.Parse("http://" + apiAddr)
	}
	return &Server{cfg: cfg, logger: logger, apiURL: parsed}
}

func (s *Server) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}

	h, err := s.buildHandler()
	if err != nil {
		return err
	}
	s.handler = h

	s.server = &http.Server{
		Addr:              s.cfg.BindAddr,
		Handler:           s.handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.logger.Info("web ui starting", logging.Field{Key: "addr", Value: s.cfg.BindAddr})
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

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	s.logger.Info("web ui stopping")
	return s.server.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.handler
}

func (s *Server) UpdateConfig(cfg config.WebUIConfig) {
	s.cfg = cfg
}

func (s *Server) buildHandler() (http.Handler, error) {
	// Access the "dist" directory within the embedded filesystem
	distFS, err := fs.Sub(content, "dist")
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	// API Proxy
	mux.Handle("/api/", s.withAuthHandler(s.proxyAPI()))

	// SPA Handler
	fileServer := http.FileServer(http.FS(distFS))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the requested file exists, serve it
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path != "" {
			if _, err := distFS.Open(path); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// Otherwise, serve index.html for client-side routing
		// We manually open index.html from distFS to serve it
		indexFile, err := http.FS(distFS).Open("index.html")
		if err != nil {
			http.Error(w, "Web UI not built", http.StatusInternalServerError)
			return
		}
		defer indexFile.Close()
		stat, _ := indexFile.Stat()
		http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile)
	}))

	return mux, nil
}

func (s *Server) withAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if s.cfg.AuthToken != "" && token != s.cfg.AuthToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) proxyAPI() http.Handler {
	if s.apiURL == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})
	}
	proxy := httputil.NewSingleHostReverseProxy(s.apiURL)
	return proxy
}
