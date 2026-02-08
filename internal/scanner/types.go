package scanner

import (
	"context"
	"time"
)

type Plugin interface {
	Name() string
	Init(config map[string]interface{}) error
	Run(ctx context.Context) (*Result, error)
	Halt(ctx context.Context) error
}

type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusPartial Status = "partial"
)

type Result struct {
	ScannerName string
	Status      Status
	Findings    []Finding
	StartedAt   time.Time
	FinishedAt  time.Time
	Duration    time.Duration
	Metadata    map[string]interface{}
}

type Finding struct {
	ID          string
	Severity    Severity
	Category    string
	Description string
	Evidence    map[string]interface{}
	Remediation string
}

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)
