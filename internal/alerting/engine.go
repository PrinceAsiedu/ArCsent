package alerting

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
)

type Alert struct {
	ID          string
	Timestamp   time.Time
	Severity    scanner.Severity
	ScannerName string
	Finding     scanner.Finding
	Reason      string
}

type Channel interface {
	Name() string
	Send(alert Alert) error
}

type Engine struct {
	logger   *logging.Logger
	channels []Channel
	throttle time.Duration
	mu       sync.Mutex
	lastSeen map[string]time.Time
}

func New(logger *logging.Logger, throttle time.Duration) *Engine {
	if throttle <= 0 {
		throttle = 5 * time.Minute
	}
	return &Engine{
		logger:   logger,
		throttle: throttle,
		lastSeen: make(map[string]time.Time),
	}
}

func (e *Engine) Register(channel Channel) {
	e.channels = append(e.channels, channel)
}

func (e *Engine) Send(alert Alert) {
	if alert.ID == "" {
		alert.ID = fingerprint(alert)
	}
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now().UTC()
	}

	if e.isThrottled(alert.ID) {
		e.logger.Warn("alert throttled", logging.Field{Key: "alert_id", Value: alert.ID})
		return
	}

	for _, ch := range e.channels {
		if err := ch.Send(alert); err != nil {
			e.logger.Error("alert delivery failed",
				logging.Field{Key: "channel", Value: ch.Name()},
				logging.Field{Key: "error", Value: err.Error()},
			)
		}
	}
}

func (e *Engine) isThrottled(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	last, ok := e.lastSeen[id]
	if ok && time.Since(last) < e.throttle {
		return true
	}
	e.lastSeen[id] = time.Now()
	return false
}

func fingerprint(alert Alert) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s", alert.ScannerName, alert.Severity, alert.Finding.ID, alert.Finding.Description)
	return hex.EncodeToString(h.Sum(nil))
}
