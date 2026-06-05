package fileio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite writes body through a temp file in path's directory, then renames
// it into place so readers never observe a partially-written progress file.
func AtomicWrite(path string, body []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".progress-*.json")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp into place: %w", err)
	}
	return nil
}

// EncodeStable emits indented JSON with HTML escaping disabled and a trailing
// newline, matching the canonical progress file writer.
func EncodeStable(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
