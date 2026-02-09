package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

type ProcessMonitor struct {
	whitelistPrefixes []string
}

func (p *ProcessMonitor) Name() string { return "system.process_monitor" }

func (p *ProcessMonitor) Init(config map[string]interface{}) error {
	p.whitelistPrefixes = []string{}
	if v, ok := config["whitelist_prefixes"].([]interface{}); ok {
		for _, raw := range v {
			if s, ok := raw.(string); ok && s != "" {
				p.whitelistPrefixes = append(p.whitelistPrefixes, s)
			}
		}
	}
	if v, ok := config["whitelist_prefixes"].([]string); ok {
		p.whitelistPrefixes = append([]string{}, v...)
	}
	return nil
}

func (p *ProcessMonitor) Run(_ context.Context) (*scanner.Result, error) {
	result := &scanner.Result{
		ScannerName: p.Name(),
		Status:      scanner.StatusSuccess,
		Metadata: map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	procEntries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	processes := 0
	for _, entry := range procEntries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		processes++

		exePath, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil {
			continue
		}

		if len(p.whitelistPrefixes) == 0 {
			continue
		}

		if !hasPrefix(exePath, p.whitelistPrefixes) {
			result.Findings = append(result.Findings, scanner.Finding{
				ID:          "process_not_whitelisted",
				Severity:    scanner.SeverityMedium,
				Category:    "process",
				Description: "Process executable not in whitelist",
				Evidence: map[string]interface{}{
					"pid": pid,
					"exe": exePath,
				},
				Remediation: "Verify the process is expected or update the whitelist.",
			})
		}
	}

	result.Metadata["processes"] = processes
	result.Metadata["whitelist_prefixes"] = strings.Join(p.whitelistPrefixes, ",")
	return result, nil
}

func (p *ProcessMonitor) Halt(_ context.Context) error { return nil }

func hasPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
