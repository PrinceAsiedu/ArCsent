package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type WebhookChannel struct {
	url      string
	severity []string
}

func NewWebhookChannel(url string, severity []string) *WebhookChannel {
	return &WebhookChannel{url: url, severity: severity}
}

func (w *WebhookChannel) Name() string { return "webhook" }

func (w *WebhookChannel) Send(alert Alert) error {
	if !severityAllowed(w.severity, alert.Severity) {
		return nil
	}
	payload, err := json.Marshal(alert)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %d", resp.StatusCode)
	}
	return nil
}
