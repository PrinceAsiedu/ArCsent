package alerting

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/ipsix/arcsent/internal/config"
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
	logger       *logging.Logger
	channels     []Channel
	throttle     time.Duration
	retryMax     int
	retryBackoff time.Duration
	enabled      bool
	mu           sync.Mutex
	lastSeen     map[string]time.Time
	queue        chan Alert
	stop         chan struct{}
}

func New(logger *logging.Logger, cfg config.AlertingConfig) *Engine {
	throttle := 5 * time.Minute
	if cfg.DedupWindow != "" {
		if parsed, err := time.ParseDuration(cfg.DedupWindow); err == nil {
			throttle = parsed
		}
	}
	backoff := 2 * time.Second
	if cfg.RetryBackoff != "" {
		if parsed, err := time.ParseDuration(cfg.RetryBackoff); err == nil {
			backoff = parsed
		}
	}
	if cfg.RetryMax < 0 {
		cfg.RetryMax = 0
	}
	engine := &Engine{
		logger:       logger,
		throttle:     throttle,
		retryMax:     cfg.RetryMax,
		retryBackoff: backoff,
		enabled:      cfg.Enabled,
		lastSeen:     make(map[string]time.Time),
		queue:        make(chan Alert, 256),
		stop:         make(chan struct{}),
	}
	if cfg.Enabled {
		go engine.worker()
	}
	return engine
}

func (e *Engine) Register(channel Channel) {
	e.channels = append(e.channels, channel)
}

func (e *Engine) Send(alert Alert) {
	if !e.enabled {
		return
	}
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

	select {
	case e.queue <- alert:
	default:
		e.logger.Warn("alert queue full, dropping", logging.Field{Key: "alert_id", Value: alert.ID})
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

func (e *Engine) worker() {
	for {
		select {
		case alert := <-e.queue:
			e.deliver(alert)
		case <-e.stop:
			return
		}
	}
}

func (e *Engine) deliver(alert Alert) {
	for _, ch := range e.channels {
		var err error
		for attempt := 0; attempt <= e.retryMax; attempt++ {
			err = ch.Send(alert)
			if err == nil {
				break
			}
			time.Sleep(e.retryBackoff * time.Duration(1<<attempt))
		}
		if err != nil {
			e.logger.Error("alert delivery failed",
				logging.Field{Key: "channel", Value: ch.Name()},
				logging.Field{Key: "error", Value: err.Error()},
			)
		}
	}
}

func fingerprint(alert Alert) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s", alert.ScannerName, alert.Severity, alert.Finding.ID, alert.Finding.Description)
	return hex.EncodeToString(h.Sum(nil))
}
