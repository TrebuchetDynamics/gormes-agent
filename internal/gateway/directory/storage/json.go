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

// File is the shared persisted-JSON file contract for directory stores. It
// keeps path construction, root validation, and atomic-write metadata together
// so cache and source stores do not each rebuild that policy.
type File struct {
	Root       Root
	Name       string
	TmpPattern string
	Label      string
}

// NewFile returns a persisted JSON file rooted at root.
func NewFile(root, name, tmpPattern, label string) File {
	return File{Root: NewRoot(root), Name: name, TmpPattern: tmpPattern, Label: label}
}

// WithDefaults returns f when it is already configured, otherwise it builds the
// store's default persisted JSON file while preserving any root carried by f.
func (f File) WithDefaults(name, tmpPattern, label string) File {
	if f.Name != "" {
		return f
	}
	return NewFile(f.Root.String(), name, tmpPattern, label)
}

// Path returns the persisted file path.
func (f File) Path() string {
	return f.Root.Path(f.Name)
}

// Require validates the file root with the store-specific label.
func (f File) Require() error {
	return f.Root.Require(f.Label)
}

// Read decodes the persisted JSON file into value.
func (f File) Read(value any) error {
	return ReadJSON(f.Path(), value)
}

// WriteAtomic marshals value as indented JSON and atomically replaces the
// persisted file.
func (f File) WriteAtomic(value any, writer Writer) error {
	if err := f.Require(); err != nil {
		return err
	}
	return WriteAtomicJSON(f.Root.String(), f.Name, f.TmpPattern, value, writer)
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
