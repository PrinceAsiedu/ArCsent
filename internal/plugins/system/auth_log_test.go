package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAuthLogMonitor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.log")
	if err := os.WriteFile(path, []byte("Failed password for invalid user\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	mon := &AuthLogMonitor{}
	if err := mon.Init(map[string]interface{}{"path": path, "max_lines": float64(10)}); err != nil {
		t.Fatalf("init: %v", err)
	}
	result, err := mon.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Findings) == 0 {
		t.Fatalf("expected findings")
	}
}
