package jsonfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteAtomic marshals payload as indented JSON and atomically replaces path.
func WriteAtomic(ctx context.Context, path string, payload any, label string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if label == "" {
		label = "json file"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s dir: %w", label, err)
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", label, err)
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(path), ".json-atomic-*.tmp")
	if err != nil {
		return fmt.Errorf("create %s temp file: %w", label, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s temp file: %w", label, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s temp file: %w", label, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", label, err)
	}
	return nil
}
