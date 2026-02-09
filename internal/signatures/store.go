package signatures

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ipsix/arcsent/internal/storage"
)

const (
	signaturesBucket    = "signatures"
	signaturesStatusKey = "status"
)

type Status struct {
	LastRun          time.Time               `json:"last_run"`
	NextRun          time.Time               `json:"next_run"`
	AirgapMode       bool                    `json:"airgap_mode"`
	AirgapImportPath string                  `json:"airgap_import_path"`
	Sources          map[string]SourceStatus `json:"sources"`
}

type SourceStatus struct {
	Source    string    `json:"source"`
	URL       string    `json:"url"`
	Path      string    `json:"path"`
	Bytes     int64     `json:"bytes"`
	UpdatedAt time.Time `json:"updated_at"`
	Duration  string    `json:"duration"`
	Error     string    `json:"error"`
}

type Store struct {
	store storage.Store
}

func NewStore(store storage.Store) *Store {
	return &Store{store: store}
}

func (s *Store) SaveStatus(status Status) error {
	raw, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("encode signatures status: %w", err)
	}
	return s.store.Put(signaturesBucket, signaturesStatusKey, raw)
}

func (s *Store) LoadStatus() (Status, error) {
	raw, err := s.store.Get(signaturesBucket, signaturesStatusKey)
	if err != nil {
		if err == storage.ErrNotFound {
			return Status{Sources: map[string]SourceStatus{}}, nil
		}
		return Status{}, err
	}
	var status Status
	if err := json.Unmarshal(raw, &status); err != nil {
		return Status{}, fmt.Errorf("decode signatures status: %w", err)
	}
	if status.Sources == nil {
		status.Sources = map[string]SourceStatus{}
	}
	return status, nil
}
