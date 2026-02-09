package detection

import (
	"testing"

	"github.com/ipsix/arcsent/internal/scanner"
)

func TestRuleEngine(t *testing.T) {
	engine := NewRuleEngine([]Rule{
		{
			Name:      "disk",
			Scanner:   "system.disk_usage",
			Metric:    "used_pct",
			Operator:  "gte",
			Threshold: 90,
			Severity:  scanner.SeverityHigh,
		},
	})
	result := scanner.Result{
		ScannerName: "system.disk_usage",
		Metadata: map[string]interface{}{
			"used_pct": 95.0,
		},
	}
	findings := engine.Evaluate(result)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}
