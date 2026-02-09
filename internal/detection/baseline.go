package detection

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/ipsix/arcsent/internal/storage"
)

const (
	baselineBucket = "baselines"
	maxSamples     = 200
	minSamples     = 10
)

type Baseline struct {
	ScannerName string    `json:"scanner_name"`
	Metric      string    `json:"metric"`
	Count       int       `json:"count"`
	Mean        float64   `json:"mean"`
	M2          float64   `json:"m2"`
	Min         float64   `json:"min"`
	Max         float64   `json:"max"`
	Samples     []float64 `json:"samples"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Manager struct {
	store storage.Store
}

func NewManager(store storage.Store) *Manager {
	return &Manager{store: store}
}

func (m *Manager) Update(scannerName, metric string, value float64) (*Baseline, error) {
	if scannerName == "" || metric == "" {
		return nil, fmt.Errorf("scannerName and metric are required")
	}

	baseline, err := m.get(scannerName, metric)
	if err != nil && err != storage.ErrNotFound {
		return nil, err
	}
	if baseline == nil {
		baseline = &Baseline{
			ScannerName: scannerName,
			Metric:      metric,
			Min:         value,
			Max:         value,
		}
	}

	baseline.Count++
	delta := value - baseline.Mean
	baseline.Mean += delta / float64(baseline.Count)
	delta2 := value - baseline.Mean
	baseline.M2 += delta * delta2

	if value < baseline.Min {
		baseline.Min = value
	}
	if value > baseline.Max {
		baseline.Max = value
	}

	baseline.Samples = append(baseline.Samples, value)
	if len(baseline.Samples) > maxSamples {
		baseline.Samples = baseline.Samples[len(baseline.Samples)-maxSamples:]
	}
	baseline.UpdatedAt = time.Now().UTC()

	if err := m.put(baseline); err != nil {
		return nil, err
	}
	return baseline, nil
}

func (m *Manager) IsAnomaly(scannerName, metric string, value float64) (bool, string, error) {
	baseline, err := m.get(scannerName, metric)
	if err != nil {
		return false, "", err
	}
	if baseline.Count < minSamples {
		return false, "insufficient_samples", nil
	}

	z := zScore(baseline, value)
	if math.Abs(z) >= 3.0 {
		return true, fmt.Sprintf("zscore=%.2f", z), nil
	}

	q1, q3 := quartiles(baseline.Samples)
	iqr := q3 - q1
	low := q1 - 1.5*iqr
	high := q3 + 1.5*iqr
	if value < low || value > high {
		return true, fmt.Sprintf("iqr_outlier (%.2f..%.2f)", low, high), nil
	}

	return false, "within_baseline", nil
}

func (m *Manager) Get(scannerName, metric string) (*Baseline, error) {
	return m.get(scannerName, metric)
}

func (m *Manager) List() ([]Baseline, error) {
	var out []Baseline
	err := m.store.ForEach(baselineBucket, func(_, value []byte) error {
		var baseline Baseline
		if err := json.Unmarshal(value, &baseline); err != nil {
			return fmt.Errorf("decode baseline: %w", err)
		}
		out = append(out, baseline)
		return nil
	})
	if err != nil {
		if err == storage.ErrNotFound {
			return []Baseline{}, nil
		}
		return nil, err
	}
	return out, nil
}

func (m *Manager) get(scannerName, metric string) (*Baseline, error) {
	key := baselineKey(scannerName, metric)
	raw, err := m.store.Get(baselineBucket, key)
	if err != nil {
		return nil, err
	}
	var baseline Baseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return nil, fmt.Errorf("decode baseline: %w", err)
	}
	return &baseline, nil
}

func (m *Manager) put(b *Baseline) error {
	raw, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("encode baseline: %w", err)
	}
	return m.store.Put(baselineBucket, baselineKey(b.ScannerName, b.Metric), raw)
}

func baselineKey(scannerName, metric string) string {
	return scannerName + "::" + metric
}

func zScore(b *Baseline, value float64) float64 {
	if b.Count < 2 {
		return 0
	}
	variance := b.M2 / float64(b.Count-1)
	if variance <= 0 {
		return 0
	}
	stddev := math.Sqrt(variance)
	if stddev == 0 {
		return 0
	}
	return (value - b.Mean) / stddev
}

func quartiles(samples []float64) (float64, float64) {
	if len(samples) == 0 {
		return 0, 0
	}
	sorted := append([]float64{}, samples...)
	sort.Float64s(sorted)
	q1 := percentile(sorted, 25)
	q3 := percentile(sorted, 75)
	return q1, q3
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	rank := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(rank))
	upper := int(math.Ceil(rank))
	if lower == upper {
		return sorted[lower]
	}
	weight := rank - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
