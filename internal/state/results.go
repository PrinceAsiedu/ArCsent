package state

import (
	"sync"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

type ResultSummary struct {
	ScannerName string            `json:"scanner_name"`
	Status      scanner.Status    `json:"status"`
	Findings    int               `json:"findings"`
	StartedAt   time.Time         `json:"started_at"`
	FinishedAt  time.Time         `json:"finished_at"`
	Duration    time.Duration     `json:"duration"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type ResultCache struct {
	mu      sync.RWMutex
	latest  map[string]scanner.Result
	history []scanner.Result
	limit   int
}

func NewResultCache(limit int) *ResultCache {
	if limit <= 0 {
		limit = 50
	}
	return &ResultCache{
		latest: make(map[string]scanner.Result),
		limit:  limit,
	}
}

func (c *ResultCache) Add(result scanner.Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latest[result.ScannerName] = result
	c.history = append(c.history, result)
	if len(c.history) > c.limit {
		c.history = c.history[len(c.history)-c.limit:]
	}
}

func (c *ResultCache) Latest() []ResultSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ResultSummary, 0, len(c.latest))
	for _, res := range c.latest {
		out = append(out, summarize(res))
	}
	return out
}

func (c *ResultCache) History() []ResultSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]ResultSummary, 0, len(c.history))
	for _, res := range c.history {
		out = append(out, summarize(res))
	}
	return out
}

type FindingSummary struct {
	ScannerName string            `json:"scanner_name"`
	Severity    scanner.Severity  `json:"severity"`
	Category    string            `json:"category"`
	Description string            `json:"description"`
	OccurredAt  time.Time         `json:"occurred_at"`
	Evidence    map[string]string `json:"evidence,omitempty"`
}

func (c *ResultCache) FindingsHistory() []FindingSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := []FindingSummary{}
	for _, res := range c.history {
		for _, finding := range res.Findings {
			evidence := map[string]string{}
			for k, v := range finding.Evidence {
				if s, ok := v.(string); ok {
					evidence[k] = s
				}
			}
			out = append(out, FindingSummary{
				ScannerName: res.ScannerName,
				Severity:    finding.Severity,
				Category:    finding.Category,
				Description: finding.Description,
				OccurredAt:  res.FinishedAt,
				Evidence:    evidence,
			})
		}
	}
	return out
}

func summarize(res scanner.Result) ResultSummary {
	meta := map[string]string{}
	if res.Metadata != nil {
		for k, v := range res.Metadata {
			if s, ok := v.(string); ok {
				meta[k] = s
			}
		}
	}
	return ResultSummary{
		ScannerName: res.ScannerName,
		Status:      res.Status,
		Findings:    len(res.Findings),
		StartedAt:   res.StartedAt,
		FinishedAt:  res.FinishedAt,
		Duration:    res.Duration,
		Metadata:    meta,
	}
}
