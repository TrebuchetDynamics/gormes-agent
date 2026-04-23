package gateway

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeJSONAtomic(path string, data []byte, mode os.FileMode) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("gateway: path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("gateway: create parent dir for %s: %w", path, err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return fmt.Errorf("gateway: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("gateway: rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}
