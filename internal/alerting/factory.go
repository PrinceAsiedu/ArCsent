package alerting

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/scanner"
)

func BuildChannels(cfg config.AlertingConfig, logger *logging.Logger) ([]Channel, error) {
	channels := []Channel{}
	for _, ch := range cfg.Channels {
		if !ch.Enabled {
			continue
		}
		switch ch.Type {
		case "log":
			channels = append(channels, NewLogChannel(logger))
		case "webhook":
			if ch.URL == "" {
				return nil, fmt.Errorf("webhook url required")
			}
			channels = append(channels, NewWebhookChannel(ch.URL, ch.Severity))
		case "syslog":
			channels = append(channels, NewSyslogChannel(ch.SyslogNetwork, ch.SyslogAddress, ch.SyslogTag, ch.Severity))
		case "email":
			channels = append(channels, NewEmailChannel(EmailConfig{
				SMTPServer: ch.SMTPServer,
				SMTPUser:   ch.SMTPUser,
				SMTPPass:   ch.SMTPPass,
				From:       ch.From,
				To:         ch.To,
				Subject:    ch.Subject,
			}, ch.Severity))
		default:
			return nil, fmt.Errorf("unknown alert channel type: %s", ch.Type)
		}
	}
	if len(channels) == 0 {
		channels = append(channels, NewLogChannel(logger))
	}
	return channels, nil
}

func severityAllowed(allow []string, sev scanner.Severity) bool {
	if len(allow) == 0 {
		return true
	}
	for _, v := range allow {
		if parseSeverity(v) == sev {
			return true
		}
	}
	return false
}

func parseSeverity(value string) scanner.Severity {
	switch strings.ToLower(value) {
	case "low":
		return scanner.SeverityLow
	case "medium":
		return scanner.SeverityMedium
	case "high":
		return scanner.SeverityHigh
	case "critical":
		return scanner.SeverityCritical
	default:
		return scanner.SeverityInfo
	}
}

var httpClient = &http.Client{Timeout: 10 * time.Second}
