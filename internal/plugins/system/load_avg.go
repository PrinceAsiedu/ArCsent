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

type LoadAverage struct{}

func (l *LoadAverage) Name() string { return "system.load_avg" }

func (l *LoadAverage) Init(_ map[string]interface{}) error { return nil }

func (l *LoadAverage) Run(_ context.Context) (*scanner.Result, error) {
	start := time.Now()
	raw, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, fmt.Errorf("read /proc/loadavg: %w", err)
	}
	avg, err := parseLoadAvg(string(raw))
	if err != nil {
		return nil, err
	}

	return &scanner.Result{
		ScannerName: l.Name(),
		Status:      scanner.StatusSuccess,
		Metadata: map[string]interface{}{
			"load1":         avg.load1,
			"load5":         avg.load5,
			"load15":        avg.load15,
			"runnable":      avg.runnable,
			"total_threads": avg.totalThreads,
			"last_pid":      avg.lastPID,
			"timestamp":     time.Now().Format(time.RFC3339),
		},
		StartedAt:  start,
		FinishedAt: time.Now(),
		Duration:   time.Since(start),
	}, nil
}

func (l *LoadAverage) Halt(_ context.Context) error { return nil }

type loadAvg struct {
	load1        float64
	load5        float64
	load15       float64
	runnable     int
	totalThreads int
	lastPID      int
}

func parseLoadAvg(raw string) (loadAvg, error) {
	fields := strings.Fields(raw)
	if len(fields) < 4 {
		return loadAvg{}, fmt.Errorf("invalid loadavg format")
	}
	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return loadAvg{}, fmt.Errorf("parse load1: %w", err)
	}
	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return loadAvg{}, fmt.Errorf("parse load5: %w", err)
	}
	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return loadAvg{}, fmt.Errorf("parse load15: %w", err)
	}

	runningParts := strings.Split(fields[3], "/")
	if len(runningParts) != 2 {
		return loadAvg{}, fmt.Errorf("invalid loadavg running format")
	}
	runnable, err := strconv.Atoi(runningParts[0])
	if err != nil {
		return loadAvg{}, fmt.Errorf("parse runnable: %w", err)
	}
	totalThreads, err := strconv.Atoi(runningParts[1])
	if err != nil {
		return loadAvg{}, fmt.Errorf("parse total threads: %w", err)
	}

	lastPID := 0
	if len(fields) > 4 {
		lastPID, _ = strconv.Atoi(fields[4])
	}

	return loadAvg{
		load1:        load1,
		load5:        load5,
		load15:       load15,
		runnable:     runnable,
		totalThreads: totalThreads,
		lastPID:      lastPID,
	}, nil
}
