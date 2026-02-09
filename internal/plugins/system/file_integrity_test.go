package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileIntegrityInitRequiresPaths(t *testing.T) {
	fi := &FileIntegrity{}
	err := fi.Init(map[string]interface{}{"paths": []interface{}{}})
	if err == nil {
		t.Fatalf("expected error for empty paths")
	}
}

func TestFileIntegrityRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fi := &FileIntegrity{}
	if err := fi.Init(map[string]interface{}{"paths": []interface{}{dir}}); err != nil {
		t.Fatalf("init: %v", err)
	}
	result, err := fi.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	hashes, ok := result.Metadata["hashes"].(map[string]string)
	if !ok {
		t.Fatalf("expected hashes metadata")
	}
	if _, exists := hashes[path]; !exists {
		t.Fatalf("expected hash entry for %s", path)
	}
}
