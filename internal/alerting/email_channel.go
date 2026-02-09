package alerting

import (
	"fmt"
	"net/smtp"
	"strings"
)

type EmailConfig struct {
	SMTPServer string
	SMTPUser   string
	SMTPPass   string
	From       string
	To         []string
	Subject    string
}

type EmailChannel struct {
	cfg      EmailConfig
	severity []string
}

func NewEmailChannel(cfg EmailConfig, severity []string) *EmailChannel {
	return &EmailChannel{cfg: cfg, severity: severity}
}

func (e *EmailChannel) Name() string { return "email" }

func (e *EmailChannel) Send(alert Alert) error {
	if !severityAllowed(e.severity, alert.Severity) {
		return nil
	}
	if e.cfg.SMTPServer == "" || e.cfg.From == "" || len(e.cfg.To) == 0 {
		return fmt.Errorf("email channel not configured")
	}
	subject := e.cfg.Subject
	if subject == "" {
		subject = "ArCsent Alert"
	}
	body := fmt.Sprintf("Severity: %s\nScanner: %s\nDescription: %s\n", alert.Severity, alert.ScannerName, alert.Finding.Description)
	msg := strings.Join([]string{
		"From: " + e.cfg.From,
		"To: " + strings.Join(e.cfg.To, ","),
		"Subject: " + subject,
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if e.cfg.SMTPUser != "" && e.cfg.SMTPPass != "" {
		host := strings.Split(e.cfg.SMTPServer, ":")[0]
		auth = smtp.PlainAuth("", e.cfg.SMTPUser, e.cfg.SMTPPass, host)
	}
	return smtp.SendMail(e.cfg.SMTPServer, auth, e.cfg.From, e.cfg.To, []byte(msg))
}
