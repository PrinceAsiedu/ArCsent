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

type CPUMemory struct {
	sampleMS    int
	includeSwap bool
}

func (c *CPUMemory) Name() string { return "system.cpu_memory" }

func (c *CPUMemory) Init(config map[string]interface{}) error {
	c.sampleMS = 200
	c.includeSwap = true

	if v, ok := config["sample_ms"].(float64); ok && v > 0 {
		c.sampleMS = int(v)
	}
	if v, ok := config["include_swap"].(bool); ok {
		c.includeSwap = v
	}
	if c.sampleMS <= 0 {
		return fmt.Errorf("sample_ms must be > 0")
	}
	return nil
}

func (c *CPUMemory) Run(_ context.Context) (*scanner.Result, error) {
	start := time.Now()
	first, err := readCPUStat()
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Duration(c.sampleMS) * time.Millisecond)
	second, err := readCPUStat()
	if err != nil {
		return nil, err
	}

	deltaTotal := second.total - first.total
	deltaIdle := second.idle - first.idle
	cpuUsage := 0.0
	if deltaTotal > 0 {
		cpuUsage = (float64(deltaTotal-deltaIdle) / float64(deltaTotal)) * 100
	}

	mem, err := readMeminfo()
	if err != nil {
		return nil, err
	}
	memUsed := mem.memTotal - mem.memAvailable
	memUsedPct := 0.0
	if mem.memTotal > 0 {
		memUsedPct = (float64(memUsed) / float64(mem.memTotal)) * 100
	}

	metadata := map[string]interface{}{
		"cpu_usage_pct":   cpuUsage,
		"mem_used_pct":    memUsedPct,
		"mem_total_bytes": mem.memTotal,
		"mem_used_bytes":  memUsed,
		"sample_ms":       c.sampleMS,
		"timestamp":       time.Now().Format(time.RFC3339),
	}
	if c.includeSwap && mem.swapTotal > 0 {
		swapUsed := mem.swapTotal - mem.swapFree
		swapUsedPct := (float64(swapUsed) / float64(mem.swapTotal)) * 100
		metadata["swap_used_pct"] = swapUsedPct
		metadata["swap_used_bytes"] = swapUsed
		metadata["swap_total_bytes"] = mem.swapTotal
	}

	return &scanner.Result{
		ScannerName: c.Name(),
		Status:      scanner.StatusSuccess,
		Metadata:    metadata,
		StartedAt:   start,
		FinishedAt:  time.Now(),
		Duration:    time.Since(start),
	}, nil
}

func (c *CPUMemory) Halt(_ context.Context) error { return nil }

type cpuStat struct {
	total uint64
	idle  uint64
}

func readCPUStat() (cpuStat, error) {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuStat{}, fmt.Errorf("read /proc/stat: %w", err)
	}
	return parseCPUStat(string(raw))
}

func parseCPUStat(raw string) (cpuStat, error) {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "cpu ") {
			parts := strings.Fields(line)
			if len(parts) < 5 {
				return cpuStat{}, fmt.Errorf("invalid cpu stat line")
			}
			values := make([]uint64, 0, len(parts)-1)
			for _, p := range parts[1:] {
				val, err := strconv.ParseUint(p, 10, 64)
				if err != nil {
					return cpuStat{}, fmt.Errorf("parse cpu stat: %w", err)
				}
				values = append(values, val)
			}
			var total uint64
			for _, v := range values {
				total += v
			}
			idle := values[3]
			if len(values) > 4 {
				idle += values[4]
			}
			return cpuStat{total: total, idle: idle}, nil
		}
	}
	return cpuStat{}, fmt.Errorf("cpu stat line not found")
}

type meminfo struct {
	memTotal     uint64
	memAvailable uint64
	swapTotal    uint64
	swapFree     uint64
}

func readMeminfo() (meminfo, error) {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return meminfo{}, fmt.Errorf("read /proc/meminfo: %w", err)
	}
	return parseMeminfo(string(raw))
}

func parseMeminfo(raw string) (meminfo, error) {
	info := meminfo{}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		valueBytes := value * 1024
		switch key {
		case "MemTotal":
			info.memTotal = valueBytes
		case "MemAvailable":
			info.memAvailable = valueBytes
		case "SwapTotal":
			info.swapTotal = valueBytes
		case "SwapFree":
			info.swapFree = valueBytes
		}
	}
	if info.memTotal == 0 || info.memAvailable == 0 {
		return meminfo{}, fmt.Errorf("meminfo missing MemTotal/MemAvailable")
	}
	return info, nil
}
