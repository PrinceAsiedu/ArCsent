package system

import "testing"

func TestDiskUsageInit(t *testing.T) {
	du := &DiskUsage{}
	if err := du.Init(map[string]interface{}{"warn_percent": float64(90), "crit_percent": float64(95)}); err != nil {
		t.Fatalf("expected init to succeed: %v", err)
	}
	if du.warnPercent >= du.critPercent {
		t.Fatalf("expected warn < crit")
	}
}

func TestDiskUsageInitRejectsBadThresholds(t *testing.T) {
	du := &DiskUsage{}
	err := du.Init(map[string]interface{}{"warn_percent": float64(95), "crit_percent": float64(90)})
	if err == nil {
		t.Fatalf("expected error for invalid thresholds")
	}
}
