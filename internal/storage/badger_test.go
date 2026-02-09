package storage

import (
	"path/filepath"
	"testing"
)

func TestBadgerStorePutGet(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBadgerStore(filepath.Join(dir, "badger"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	if err := store.Put("bucket", "key", []byte("value")); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := store.Get("bucket", "key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "value" {
		t.Fatalf("expected value, got %s", string(got))
	}
}

func TestBadgerStoreForEach(t *testing.T) {
	dir := t.TempDir()
	store, err := NewBadgerStore(filepath.Join(dir, "badger"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	if err := store.Put("bucket", "key1", []byte("value1")); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := store.Put("bucket", "key2", []byte("value2")); err != nil {
		t.Fatalf("put: %v", err)
	}

	seen := 0
	err = store.ForEach("bucket", func(key, value []byte) error {
		seen++
		return nil
	})
	if err != nil {
		t.Fatalf("foreach: %v", err)
	}
	if seen != 2 {
		t.Fatalf("expected 2 items, got %d", seen)
	}
}
