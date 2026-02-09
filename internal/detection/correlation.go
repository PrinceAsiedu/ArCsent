package detection

import (
	"sync"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

type Correlator struct {
	window   time.Duration
	minScan  int
	cooldown time.Duration

	mu            sync.Mutex
	events        []correlationEvent
	lastTriggered time.Time
}

type correlationEvent struct {
	at      time.Time
	scanner string
}

func NewCorrelator(window time.Duration, minScanners int, cooldown time.Duration) *Correlator {
	if window <= 0 {
		window = 5 * time.Minute
	}
	if minScanners < 1 {
		minScanners = 2
	}
	if cooldown <= 0 {
		cooldown = window
	}
	return &Correlator{
		window:   window,
		minScan:  minScanners,
		cooldown: cooldown,
	}
}

func (c *Correlator) Add(result scanner.Result) []scanner.Finding {
	if len(result.Findings) == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.events = append(c.events, correlationEvent{at: now, scanner: result.ScannerName})
	c.prune(now)

	unique := map[string]struct{}{}
	for _, ev := range c.events {
		unique[ev.scanner] = struct{}{}
	}

	if len(unique) < c.minScan {
		return nil
	}
	if now.Sub(c.lastTriggered) < c.cooldown {
		return nil
	}

	c.lastTriggered = now
	return []scanner.Finding{
		{
			ID:          "correlation_multi_scanner",
			Severity:    scanner.SeverityHigh,
			Category:    "correlation",
			Description: "Multiple scanners reported findings within correlation window.",
			Evidence: map[string]interface{}{
				"unique_scanners": len(unique),
				"window":          c.window.String(),
			},
			Remediation: "Investigate combined signals for coordinated activity.",
		},
	}
}

func (c *Correlator) prune(now time.Time) {
	cutoff := now.Add(-c.window)
	filtered := c.events[:0]
	for _, ev := range c.events {
		if ev.at.After(cutoff) {
			filtered = append(filtered, ev)
		}
	}
	c.events = filtered
}
