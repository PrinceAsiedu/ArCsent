package system

import "testing"

func TestParseUptime(t *testing.T) {
	raw := "12345.67 54321.00\n"
	info, err := parseUptime(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.uptimeSeconds == 0 || info.idleSeconds == 0 {
		t.Fatalf("expected uptime values to be parsed")
	}
}
