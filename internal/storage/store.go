package storage

import "errors"

var ErrNotFound = errors.New("not found")

type Store interface {
	Put(bucket, key string, value []byte) error
	Get(bucket, key string) ([]byte, error)
	ForEach(bucket string, fn func(key, value []byte) error) error
	Delete(bucket, key string) error
	Close() error
}
