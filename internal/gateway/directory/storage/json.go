package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Root is the shared caller-owned persistence root contract for directory
// stores. It centralizes root normalization, file path construction, and empty
// root validation across cache and remembered-source stores.
type Root string

// NewRoot returns a normalized caller-owned persistence root.
func NewRoot(root string) Root {
	return Root(strings.TrimSpace(root))
}

func (r Root) String() string { return string(r) }

// Path returns the path to name under the normalized root.
func (r Root) Path(name string) string {
	return filepath.Join(string(r), name)
}

// Require returns a store-specific empty-root error when the root is not set.
func (r Root) Require(label string) error {
	if strings.TrimSpace(string(r)) == "" {
		return fmt.Errorf("%s root is empty", label)
	}
	return nil
}

// Writer is the injectable write seam used by stores that need deterministic
// atomic-write failure tests.
type Writer func(string, []byte, os.FileMode) error

// ReadJSON reads path into value using the stores' shared JSON decoding seam.
func ReadJSON(path string, value any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, value)
}

// WriteAtomicJSON marshals value as indented JSON and atomically replaces name
// under root. The temporary file is created in root so rename stays atomic on
// normal local filesystems.
func WriteAtomicJSON(root, name, tmpPattern string, value any, writer Writer) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(root, tmpPattern)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if writer == nil {
		writer = os.WriteFile
	}
	if err := writer(tmpPath, body, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, filepath.Join(root, name)); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
