package system

import "testing"

func TestParseLoadAvg(t *testing.T) {
	raw := "0.12 0.34 0.56 1/234 5678"
	avg, err := parseLoadAvg(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avg.load1 == 0 || avg.load5 == 0 || avg.load15 == 0 {
		t.Fatalf("expected load averages to be parsed")
	}
	if avg.totalThreads != 234 {
		t.Fatalf("expected totalThreads=234, got %d", avg.totalThreads)
	}
}
