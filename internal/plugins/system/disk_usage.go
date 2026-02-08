package system

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

type DiskUsage struct {
	path         string
	warnPercent  float64
	critPercent  float64
}

func (d *DiskUsage) Name() string { return "system.disk_usage" }

func (d *DiskUsage) Init(config map[string]interface{}) error {
	d.path = "/"
	d.warnPercent = 85
	d.critPercent = 95

	if v, ok := config["path"].(string); ok && v != "" {
		d.path = v
	}
	if v, ok := config["warn_percent"].(float64); ok && v > 0 {
		d.warnPercent = v
	}
	if v, ok := config["crit_percent"].(float64); ok && v > 0 {
		d.critPercent = v
	}
	if d.warnPercent >= d.critPercent {
		return fmt.Errorf("warn_percent must be less than crit_percent")
	}
	return nil
}

func (d *DiskUsage) Run(_ context.Context) (*scanner.Result, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(d.path, &stat); err != nil {
		return nil, fmt.Errorf("statfs %s: %w", d.path, err)
	}

	total := float64(stat.Blocks) * float64(stat.Bsize)
	free := float64(stat.Bavail) * float64(stat.Bsize)
	used := total - free
	usedPct := 0.0
	if total > 0 {
		usedPct = (used / total) * 100
	}

	result := &scanner.Result{
		ScannerName: d.Name(),
		Status:      scanner.StatusSuccess,
		Metadata: map[string]interface{}{
			"path":       d.path,
			"total":      total,
			"used":       used,
			"used_pct":   usedPct,
			"warn_pct":   d.warnPercent,
			"crit_pct":   d.critPercent,
			"timestamp":  time.Now().Format(time.RFC3339),
		},
	}

	if usedPct >= d.critPercent {
		result.Findings = append(result.Findings, scanner.Finding{
			ID:          "disk_usage_critical",
			Severity:    scanner.SeverityCritical,
			Category:    "resource",
			Description: fmt.Sprintf("Disk usage %.2f%% exceeds critical threshold", usedPct),
			Evidence: map[string]interface{}{
				"path":     d.path,
				"used_pct": usedPct,
			},
			Remediation: "Free disk space or expand storage.",
		})
	} else if usedPct >= d.warnPercent {
		result.Findings = append(result.Findings, scanner.Finding{
			ID:          "disk_usage_warning",
			Severity:    scanner.SeverityMedium,
			Category:    "resource",
			Description: fmt.Sprintf("Disk usage %.2f%% exceeds warning threshold", usedPct),
			Evidence: map[string]interface{}{
				"path":     d.path,
				"used_pct": usedPct,
			},
			Remediation: "Investigate disk usage growth.",
		})
	}

	return result, nil
}

func (d *DiskUsage) Halt(_ context.Context) error { return nil }
