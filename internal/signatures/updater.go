package signatures

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ipsix/arcsent/internal/logging"
)

type Config struct {
	Enabled          bool
	UpdateInterval   time.Duration
	Sources          []string
	CacheDir         string
	AirgapImportPath string
	SourceURLs       map[string]string
}

type Source interface {
	ID() string
	Update(ctx context.Context, destDir string, client *http.Client) (SourceStatus, error)
}

type HTTPSource struct {
	id  string
	url string
}

func (s HTTPSource) ID() string {
	return s.id
}

func (s HTTPSource) Update(ctx context.Context, destDir string, client *http.Client) (SourceStatus, error) {
	status := SourceStatus{
		Source: s.id,
		URL:    s.url,
	}
	if s.url == "" {
		return status, fmt.Errorf("no URL configured for source %s", s.id)
	}
	start := time.Now()
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return status, fmt.Errorf("create dest dir: %w", err)
	}
	filename := filenameForURL(s.url)
	if filename == "" {
		filename = "latest.data"
	}
	dest := filepath.Join(destDir, filename)
	tmp, err := os.CreateTemp(destDir, "download-*")
	if err != nil {
		return status, fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return status, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "arcsent-signatures/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return status, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return status, fmt.Errorf("download status %d", resp.StatusCode)
	}
	written, err := io.Copy(tmp, resp.Body)
	if err != nil {
		return status, fmt.Errorf("write download: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return status, fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return status, fmt.Errorf("rename download: %w", err)
	}

	status.Path = dest
	status.Bytes = written
	status.UpdatedAt = time.Now()
	status.Duration = time.Since(start).String()
	return status, nil
}

type Updater struct {
	cfg     Config
	store   *Store
	logger  *logging.Logger
	sources map[string]Source
	client  *http.Client
	cfgMu   sync.RWMutex
	mu      sync.Mutex
	lastRun time.Time
	nextRun time.Time
}

func NewUpdater(cfg Config, store *Store, logger *logging.Logger) *Updater {
	return &Updater{
		cfg:     cfg,
		store:   store,
		logger:  logger,
		sources: builtInSources(cfg.SourceURLs),
		client:  &http.Client{Timeout: 2 * time.Minute},
	}
}

func (u *Updater) Start(ctx context.Context) {
	for {
		cfg := u.getConfig()
		if !cfg.Enabled {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		interval := cfg.UpdateInterval
		if interval <= 0 {
			u.logger.Warn("signatures update interval invalid or zero")
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
				continue
			}
		}

		u.logger.Info("signatures updater started", logging.Field{Key: "interval", Value: interval.String()})
		u.runOnce(ctx)

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (u *Updater) Trigger(ctx context.Context) (Status, error) {
	if !u.getConfig().Enabled {
		return Status{}, errors.New("signatures updates are disabled")
	}
	return u.runOnce(ctx)
}

func (u *Updater) Status() (Status, error) {
	if u.store == nil {
		return Status{}, nil
	}
	return u.store.LoadStatus()
}

func (u *Updater) runOnce(ctx context.Context) (Status, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	cfg := u.getConfig()
	status := Status{
		Sources:          map[string]SourceStatus{},
		AirgapMode:       cfg.AirgapImportPath != "",
		AirgapImportPath: cfg.AirgapImportPath,
	}
	u.lastRun = time.Now()
	status.LastRun = u.lastRun
	if interval := cfg.UpdateInterval; interval > 0 {
		u.nextRun = u.lastRun.Add(interval)
		status.NextRun = u.nextRun
	}

	if cfg.AirgapImportPath != "" {
		if err := importAirgap(cfg.AirgapImportPath, cfg.CacheDir); err != nil {
			status.Sources["airgap"] = SourceStatus{
				Source: "airgap",
				Error:  err.Error(),
			}
		} else {
			status.Sources["airgap"] = SourceStatus{
				Source:    "airgap",
				Path:      cfg.CacheDir,
				UpdatedAt: time.Now(),
			}
		}
		if u.store != nil {
			_ = u.store.SaveStatus(status)
		}
		u.logger.Info("signatures airgap import completed")
		return status, nil
	}

	for _, srcID := range cfg.Sources {
		src, ok := u.sources[srcID]
		if !ok {
			status.Sources[srcID] = SourceStatus{
				Source: srcID,
				Error:  "source not registered",
			}
			continue
		}
		srcDir := filepath.Join(cfg.CacheDir, srcID)
		srcStatus, err := src.Update(ctx, srcDir, u.client)
		if err != nil {
			srcStatus.Error = err.Error()
			u.logger.Warn("signatures update failed", logging.Field{Key: "source", Value: srcID}, logging.Field{Key: "error", Value: err.Error()})
		} else {
			u.logger.Info("signatures update complete", logging.Field{Key: "source", Value: srcID})
		}
		status.Sources[srcID] = srcStatus
	}

	if u.store != nil {
		_ = u.store.SaveStatus(status)
	}

	return status, nil
}

func (u *Updater) UpdateConfig(cfg Config) {
	u.cfgMu.Lock()
	u.cfg = cfg
	u.sources = builtInSources(cfg.SourceURLs)
	u.cfgMu.Unlock()
}

func (u *Updater) getConfig() Config {
	u.cfgMu.RLock()
	defer u.cfgMu.RUnlock()
	cfg := u.cfg
	cfg.Sources = append([]string{}, cfg.Sources...)
	if cfg.SourceURLs != nil {
		clone := map[string]string{}
		for key, value := range cfg.SourceURLs {
			clone[key] = value
		}
		cfg.SourceURLs = clone
	}
	return cfg
}

func builtInSources(overrides map[string]string) map[string]Source {
	urls := defaultSourceURLs()
	for id, url := range overrides {
		urls[id] = url
	}

	sources := map[string]Source{}
	for id, url := range urls {
		sources[id] = HTTPSource{id: id, url: url}
	}
	return sources
}

func defaultSourceURLs() map[string]string {
	return map[string]string{
		SourceMITREATTACK: "https://raw.githubusercontent.com/mitre/cti/master/enterprise-attack/enterprise-attack.json",
		SourceNVD:         "https://nvd.nist.gov/feeds/json/cve/2.0/nvdcve-2.0-recent.json.gz",
		SourceCISAKEV:     "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json",
		SourceExploitDB:   "https://gitlab.com/exploit-database/exploitdb/-/raw/main/files_exploits.csv",
		// Optional sources require override URLs or mirrors.
		SourceOSV:        "",
		SourceMITRECAPEC: "",
		SourceMITRECWE:   "",
		SourceEPSS:       "",
		SourceGHSA:       "",
	}
}

func filenameForURL(raw string) string {
	parsed := strings.Split(raw, "?")[0]
	base := path.Base(parsed)
	base = strings.TrimSpace(base)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	return base
}

func importAirgap(srcPath, destDir string) error {
	info, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("stat airgap path: %w", err)
	}
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	if info.IsDir() {
		return copyDir(srcPath, destDir)
	}
	switch {
	case strings.HasSuffix(srcPath, ".zip"):
		return extractZip(srcPath, destDir)
	case strings.HasSuffix(srcPath, ".tar.gz"), strings.HasSuffix(srcPath, ".tgz"):
		return extractTarGz(srcPath, destDir)
	case strings.HasSuffix(srcPath, ".tar"):
		return extractTar(srcPath, destDir)
	default:
		return copyFile(srcPath, filepath.Join(destDir, filepath.Base(srcPath)))
	}
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(pathname string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, pathname)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		return copyFile(pathname, target)
	})
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func extractZip(src, dest string) error {
	archive, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer archive.Close()
	for _, f := range archive.File {
		target, err := safeExtractPath(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o750); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			in.Close()
			out.Close()
			return err
		}
		in.Close()
		out.Close()
	}
	return nil
}

func extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractTarReader(gz, dest)
}

func extractTar(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()
	return extractTarReader(file, dest)
}

func extractTarReader(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeExtractPath(dest, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o750); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
				return err
			}
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		default:
			continue
		}
	}
}

func safeExtractPath(dest, name string) (string, error) {
	cleaned := filepath.Clean(name)
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid path traversal: %s", name)
	}
	cleanDest := filepath.Clean(dest)
	target := filepath.Join(cleanDest, cleaned)
	prefix := cleanDest + string(filepath.Separator)
	if target != cleanDest && !strings.HasPrefix(target, prefix) {
		return "", fmt.Errorf("invalid extract target: %s", target)
	}
	return target, nil
}
