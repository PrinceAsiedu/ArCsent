package system

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

type AuthLogMonitor struct {
	path           string
	failedPatterns []string
	maxLines       int
}

func (a *AuthLogMonitor) Name() string { return "system.auth_log" }

func (a *AuthLogMonitor) Init(config map[string]interface{}) error {
	a.path = "/var/log/auth.log"
	a.failedPatterns = []string{
		"Failed password",
		"authentication failure",
		"Invalid user",
	}
	a.maxLines = 500

	if v, ok := config["path"].(string); ok && v != "" {
		a.path = v
	}
	if v, ok := config["failed_patterns"].([]interface{}); ok && len(v) > 0 {
		a.failedPatterns = []string{}
		for _, raw := range v {
			if s, ok := raw.(string); ok && s != "" {
				a.failedPatterns = append(a.failedPatterns, s)
			}
		}
	}
	if v, ok := config["max_lines"].(float64); ok && v > 0 {
		a.maxLines = int(v)
	}
	return nil
}

func (a *AuthLogMonitor) Run(_ context.Context) (*scanner.Result, error) {
	file, err := os.Open(a.path)
	if err != nil {
		return nil, fmt.Errorf("open auth log: %w", err)
	}
	defer file.Close()

	result := &scanner.Result{
		ScannerName: a.Name(),
		Status:      scanner.StatusSuccess,
		Metadata: map[string]interface{}{
			"path":      a.path,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	lines := readLastLines(file, a.maxLines)
	for _, line := range lines {
		for _, pattern := range a.failedPatterns {
			if strings.Contains(line, pattern) {
				result.Findings = append(result.Findings, scanner.Finding{
					ID:          "auth_failed",
					Severity:    scanner.SeverityMedium,
					Category:    "auth",
					Description: "Authentication failure detected",
					Evidence: map[string]interface{}{
						"line": line,
					},
					Remediation: "Review auth logs and block suspicious sources.",
				})
				break
			}
		}
	}

	result.Metadata["lines_scanned"] = len(lines)
	return result, nil
}

func (a *AuthLogMonitor) Halt(_ context.Context) error { return nil }

func readLastLines(file *os.File, maxLines int) []string {
	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	return lines
}
