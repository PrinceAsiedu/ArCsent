package storage

import (
	"encoding/base64"
	"fmt"
	"path/filepath"

	"github.com/dgraph-io/badger/v4"
)

type BadgerStore struct {
	db *badger.DB
}

func NewBadgerStore(path string) (*BadgerStore, error) {
	return NewBadgerStoreWithKey(path, "")
}

func NewBadgerStoreWithKey(path string, keyBase64 string) (*BadgerStore, error) {
	if path == "" {
		return nil, fmt.Errorf("storage path is required")
	}
	opts := badger.DefaultOptions(path)
	if keyBase64 != "" {
		key, err := base64.StdEncoding.DecodeString(keyBase64)
		if err != nil {
			return nil, fmt.Errorf("decode encryption key: %w", err)
		}
		if len(key) != 32 {
			return nil, fmt.Errorf("encryption key must be 32 bytes")
		}
		opts = opts.WithEncryptionKey(key)
	}
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger: %w", err)
	}
	return &BadgerStore{db: db}, nil
}

func (b *BadgerStore) Put(bucket, key string, value []byte) error {
	if bucket == "" || key == "" {
		return fmt.Errorf("bucket and key are required")
	}
	return b.db.Update(func(txn *badger.Txn) error {
		itemKey := makeKey(bucket, key)
		return txn.Set(itemKey, value)
	})
}

func (b *BadgerStore) Get(bucket, key string) ([]byte, error) {
	if bucket == "" || key == "" {
		return nil, fmt.Errorf("bucket and key are required")
	}
	var out []byte
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(makeKey(bucket, key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFound
			}
			return err
		}
		return item.Value(func(val []byte) error {
			out = append([]byte{}, val...)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (b *BadgerStore) ForEach(bucket string, fn func(key, value []byte) error) error {
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	prefix := []byte(bucket + "/")
	return b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			key := string(k[len(prefix):])
			if err := item.Value(func(val []byte) error {
				return fn([]byte(key), val)
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *BadgerStore) Delete(bucket, key string) error {
	if bucket == "" || key == "" {
		return fmt.Errorf("bucket and key are required")
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(makeKey(bucket, key))
	})
}

func (b *BadgerStore) Close() error {
	if b.db == nil {
		return nil
	}
	return b.db.Close()
}

func makeKey(bucket, key string) []byte {
	return []byte(filepath.ToSlash(bucket + "/" + key))
}
