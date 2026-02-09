package system

import "testing"

func TestParseCPUStat(t *testing.T) {
	raw := "cpu  2255 34 2290 22625563 6290 127 456\ncpu0 1132 17 1441 11311771 3675 127 438"
	stat, err := parseCPUStat(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stat.total == 0 {
		t.Fatalf("expected total > 0")
	}
	if stat.idle == 0 {
		t.Fatalf("expected idle > 0")
	}
}

func TestParseMeminfo(t *testing.T) {
	raw := "MemTotal:       16384256 kB\nMemAvailable:   12345678 kB\nSwapTotal:       2097148 kB\nSwapFree:        1048576 kB\n"
	info, err := parseMeminfo(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.memTotal == 0 || info.memAvailable == 0 {
		t.Fatalf("expected mem totals to be populated")
	}
	if info.swapTotal == 0 || info.swapFree == 0 {
		t.Fatalf("expected swap totals to be populated")
	}
}
