package daemon

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

func VerifySelfIntegrity(expected string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	file, err := os.Open(exe)
	if err != nil {
		return fmt.Errorf("open executable: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("hash executable: %w", err)
	}
	actual := fmt.Sprintf("%x", hasher.Sum(nil))
	if actual != expected {
		return fmt.Errorf("self-integrity mismatch: expected %s got %s", expected, actual)
	}
	return nil
}
