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

// ReadRequired decodes a JSON file that must exist. Missing files return
// os.ErrNotExist so callers with degraded-state machines can distinguish
// absent state from malformed JSON while sharing the decode contract.
func ReadRequired(ctx context.Context, path string, out any, label string) error {
	exists, err := Read(ctx, path, out, label)
	if err != nil {
		return err
	}
	if !exists {
		return os.ErrNotExist
	}
	return nil
}

// Writer is an injectable file-write seam used by stores that need
// deterministic atomic-write failure tests.
type Writer func(string, []byte, os.FileMode) error

// WriteOptions carries filesystem policy for JSON marker stores that need
// stricter permissions, stable temp-file prefixes, or injected write seams
// while sharing the same encode/write/rename contract.
type WriteOptions struct {
	DirMode    os.FileMode
	FileMode   os.FileMode
	TmpPattern string
	Writer     Writer
	Sync       bool
}

// MarshalIndentNewline marshals payload as indented JSON and appends the
// trailing newline used by gateway JSON marker files.
func MarshalIndentNewline(payload any) ([]byte, error) {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// WriteAtomic marshals payload as indented JSON and atomically replaces path.
func WriteAtomic(ctx context.Context, path string, payload any, label string) error {
	return WriteAtomicWithOptions(ctx, path, payload, label, WriteOptions{})
}

// WriteAtomicWithOptions marshals payload as indented JSON and atomically
// replaces path using the supplied filesystem policy. Zero-valued options keep
// the historical gateway marker behavior: 0755 directories, CreateTemp's file
// mode, and the .json-atomic-*.tmp prefix.
func WriteAtomicWithOptions(ctx context.Context, path string, payload any, label string, opts WriteOptions) error {
	raw, err := MarshalIndentNewline(payload)
	if err != nil {
		if label == "" {
			label = "json file"
		}
		return fmt.Errorf("encode %s: %w", label, err)
	}
	return WriteRawAtomicWithOptions(ctx, path, raw, label, opts)
}

// WriteRawAtomicWithOptions atomically replaces path with pre-encoded JSON
// bytes using the supplied filesystem policy. It is for stores that already
// own their encode/defaulting pipeline but should share the gateway JSON-file
// write/rename contract.
func WriteRawAtomicWithOptions(ctx context.Context, path string, raw []byte, label string, opts WriteOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if label == "" {
		label = "json file"
	}
	dirMode := opts.DirMode
	if dirMode == 0 {
		dirMode = 0o755
	}
	tmpPattern := opts.TmpPattern
	if tmpPattern == "" {
		tmpPattern = ".json-atomic-*.tmp"
	}
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return fmt.Errorf("create %s dir: %w", label, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), tmpPattern)
	if err != nil {
		return fmt.Errorf("create %s temp file: %w", label, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if opts.Writer != nil {
		if err := tmp.Close(); err != nil {
			return fmt.Errorf("close %s temp file: %w", label, err)
		}
		mode := opts.FileMode
		if mode == 0 {
			mode = 0o600
		}
		if err := opts.Writer(tmpPath, raw, mode); err != nil {
			return fmt.Errorf("write %s temp file: %w", label, err)
		}
	} else {
		if _, err := tmp.Write(raw); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("write %s temp file: %w", label, err)
		}
		if opts.Sync {
			if err := tmp.Sync(); err != nil {
				_ = tmp.Close()
				return fmt.Errorf("sync %s temp file: %w", label, err)
			}
		}
		if err := tmp.Close(); err != nil {
			return fmt.Errorf("close %s temp file: %w", label, err)
		}
	}
	if opts.FileMode != 0 {
		if err := os.Chmod(tmpPath, opts.FileMode); err != nil {
			return fmt.Errorf("chmod %s temp file: %w", label, err)
		}
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", label, err)
	}
	if opts.FileMode != 0 {
		if err := os.Chmod(path, opts.FileMode); err != nil {
			return fmt.Errorf("chmod %s: %w", label, err)
		}
	}
	return nil
}
