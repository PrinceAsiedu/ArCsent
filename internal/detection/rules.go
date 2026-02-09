package detection

import (
	"fmt"
	"strings"

	"github.com/ipsix/arcsent/internal/scanner"
)

type Rule struct {
	Name        string
	Scanner     string
	Metric      string
	Operator    string
	Threshold   float64
	Severity    scanner.Severity
	Description string
}

type RuleEngine struct {
	rules []Rule
}

func NewRuleEngine(rules []Rule) *RuleEngine {
	return &RuleEngine{rules: rules}
}

func (r *RuleEngine) Evaluate(result scanner.Result) []scanner.Finding {
	findings := []scanner.Finding{}
	for _, rule := range r.rules {
		if rule.Scanner != result.ScannerName && rule.Scanner != "*" {
			continue
		}
		raw, ok := result.Metadata[rule.Metric]
		if !ok {
			continue
		}
		value, ok := toFloat(raw)
		if !ok {
			continue
		}
		if !compare(value, rule.Operator, rule.Threshold) {
			continue
		}
		desc := rule.Description
		if desc == "" {
			desc = fmt.Sprintf("Rule %s triggered for %s", rule.Name, rule.Metric)
		}
		findings = append(findings, scanner.Finding{
			ID:          "rule_" + strings.ToLower(rule.Name),
			Severity:    rule.Severity,
			Category:    "rule",
			Description: desc,
			Evidence: map[string]interface{}{
				"metric":    rule.Metric,
				"value":     value,
				"threshold": rule.Threshold,
				"operator":  rule.Operator,
			},
			Remediation: "Review rule configuration and system state.",
		})
	}
	return findings
}

func compare(value float64, op string, threshold float64) bool {
	switch strings.ToLower(op) {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return false
	}
}

func toFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case uint:
		return float64(v), true
	default:
		return 0, false
	}
}
