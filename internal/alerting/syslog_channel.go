package alerting

import (
	"fmt"
	"log/syslog"

	"github.com/ipsix/arcsent/internal/scanner"
)

type SyslogChannel struct {
	writer   *syslog.Writer
	severity []string
}

func NewSyslogChannel(network, address, tag string, severity []string) *SyslogChannel {
	if network == "" {
		network = "unixgram"
	}
	if address == "" {
		address = "/dev/log"
	}
	if tag == "" {
		tag = "arcsent"
	}
	writer, _ := syslog.Dial(network, address, syslog.LOG_USER|syslog.LOG_INFO, tag)
	return &SyslogChannel{writer: writer, severity: severity}
}

func (s *SyslogChannel) Name() string { return "syslog" }

func (s *SyslogChannel) Send(alert Alert) error {
	if !severityAllowed(s.severity, alert.Severity) {
		return nil
	}
	if s.writer == nil {
		return fmt.Errorf("syslog writer not available")
	}
	msg := fmt.Sprintf("[%s] %s - %s", alert.Severity, alert.ScannerName, alert.Finding.Description)
	switch alert.Severity {
	case scanner.SeverityCritical, scanner.SeverityHigh:
		return s.writer.Err(msg)
	case scanner.SeverityMedium:
		return s.writer.Warning(msg)
	default:
		return s.writer.Info(msg)
	}
}
