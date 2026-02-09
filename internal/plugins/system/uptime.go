package system

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

type Uptime struct{}

func (u *Uptime) Name() string { return "system.uptime" }

func (u *Uptime) Init(_ map[string]interface{}) error { return nil }

func (u *Uptime) Run(_ context.Context) (*scanner.Result, error) {
	start := time.Now()
	raw, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return nil, fmt.Errorf("read /proc/uptime: %w", err)
	}
	info, err := parseUptime(string(raw))
	if err != nil {
		return nil, err
	}

	return &scanner.Result{
		ScannerName: u.Name(),
		Status:      scanner.StatusSuccess,
		Metadata: map[string]interface{}{
			"uptime_seconds": info.uptimeSeconds,
			"idle_seconds":   info.idleSeconds,
			"timestamp":      time.Now().Format(time.RFC3339),
		},
		StartedAt:  start,
		FinishedAt: time.Now(),
		Duration:   time.Since(start),
	}, nil
}

func (u *Uptime) Halt(_ context.Context) error { return nil }

type uptimeInfo struct {
	uptimeSeconds float64
	idleSeconds   float64
}

func parseUptime(raw string) (uptimeInfo, error) {
	fields := strings.Fields(raw)
	if len(fields) < 2 {
		return uptimeInfo{}, fmt.Errorf("invalid uptime format")
	}
	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return uptimeInfo{}, fmt.Errorf("parse uptime: %w", err)
	}
	idle, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return uptimeInfo{}, fmt.Errorf("parse idle: %w", err)
	}
	return uptimeInfo{uptimeSeconds: uptime, idleSeconds: idle}, nil
}
