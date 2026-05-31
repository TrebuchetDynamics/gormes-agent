package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrEmpty reports that a JSON file exists but has no bytes to decode.
var ErrEmpty = errors.New("empty json file")

// ReadError wraps OS read failures so callers can distinguish them from
// malformed JSON while preserving their own invalid-marker state machines.
type ReadError struct {
	Label string
	Err   error
}

func (e ReadError) Error() string { return fmt.Sprintf("read %s: %v", e.Label, e.Err) }
func (e ReadError) Unwrap() error { return e.Err }

func IsReadError(err error) bool {
	var readErr ReadError
	return errors.As(err, &readErr)
}

// Read decodes a JSON file when it exists. A missing file returns false with no
// error so marker stores can keep their missing/invalid state machines local.
func Read(ctx context.Context, path string, out any, label string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if label == "" {
		label = "json file"
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, ReadError{Label: label, Err: err}
	}
	if len(raw) == 0 {
		return true, ErrEmpty
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return true, err
	}
	return true, nil
}

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
