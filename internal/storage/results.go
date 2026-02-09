package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

const resultsBucket = "results"

type ResultsStore struct {
	store Store
}

func NewResultsStore(store Store) *ResultsStore {
	return &ResultsStore{store: store}
}

func (r *ResultsStore) Save(result scanner.Result) error {
	key := fmt.Sprintf("%d-%s-%s", time.Now().UnixNano(), result.ScannerName, randSuffix())
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode result: %w", err)
	}
	return r.store.Put(resultsBucket, key, raw)
}

func (r *ResultsStore) List() ([]scanner.Result, error) {
	results := []scanner.Result{}
	err := r.store.ForEach(resultsBucket, func(_, value []byte) error {
		var res scanner.Result
		if err := json.Unmarshal(value, &res); err != nil {
			return fmt.Errorf("decode result: %w", err)
		}
		results = append(results, res)
		return nil
	})
	if err != nil {
		if err == ErrNotFound {
			return []scanner.Result{}, nil
		}
		return nil, err
	}
	return results, nil
}

func (r *ResultsStore) PruneOlderThan(cutoff time.Time) error {
	return r.store.ForEach(resultsBucket, func(key, value []byte) error {
		var res scanner.Result
		if err := json.Unmarshal(value, &res); err != nil {
			return nil
		}
		if !res.FinishedAt.IsZero() && res.FinishedAt.Before(cutoff) {
			return r.store.Delete(resultsBucket, string(key))
		}
		return nil
	})
}

func randSuffix() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "0000"
	}
	return hex.EncodeToString(buf)
}
