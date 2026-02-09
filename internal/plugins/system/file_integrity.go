package system

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

type FileIntegrity struct {
	paths []string
}

func (f *FileIntegrity) Name() string { return "system.file_integrity" }

func (f *FileIntegrity) Init(config map[string]interface{}) error {
	f.paths = []string{"/etc", "/bin"}
	if v, ok := config["paths"].([]interface{}); ok {
		f.paths = make([]string, 0, len(v))
		for _, raw := range v {
			if path, ok := raw.(string); ok && path != "" {
				f.paths = append(f.paths, path)
			}
		}
	}
	if v, ok := config["paths"].([]string); ok && len(v) > 0 {
		f.paths = append([]string{}, v...)
	}
	if len(f.paths) == 0 {
		return fmt.Errorf("paths must not be empty")
	}
	return nil
}

func (f *FileIntegrity) Run(_ context.Context) (*scanner.Result, error) {
	result := &scanner.Result{
		ScannerName: f.Name(),
		Status:      scanner.StatusSuccess,
		Metadata: map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	hashes := make(map[string]string)
	for _, root := range f.paths {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				result.Findings = append(result.Findings, scanner.Finding{
					ID:          "file_access_error",
					Severity:    scanner.SeverityLow,
					Category:    "file_integrity",
					Description: fmt.Sprintf("Failed to access %s", path),
					Evidence: map[string]interface{}{
						"path":  path,
						"error": err.Error(),
					},
					Remediation: "Verify file permissions and integrity.",
				})
				return nil
			}
			if d.IsDir() {
				return nil
			}
			hash, err := hashFile(path)
			if err != nil {
				result.Findings = append(result.Findings, scanner.Finding{
					ID:          "file_hash_error",
					Severity:    scanner.SeverityLow,
					Category:    "file_integrity",
					Description: fmt.Sprintf("Failed to hash %s", path),
					Evidence: map[string]interface{}{
						"path":  path,
						"error": err.Error(),
					},
					Remediation: "Verify file readability.",
				})
				return nil
			}
			hashes[path] = hash
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", root, err)
		}
	}

	result.Metadata["hashes"] = hashes
	return result, nil
}

func (f *FileIntegrity) Halt(_ context.Context) error { return nil }

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
