package persisted

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage/codec"
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

// Spec names one persisted JSON file used by a directory store. It centralizes
// file names, temp-file patterns, and root-validation labels so stores can
// share persistence metadata without each rebuilding the same contract.
type Spec struct {
	Name       string
	TmpPattern string
	Label      string
}

// File returns a persisted JSON file for root using the spec metadata.
func (s Spec) File(root string) File {
	return File{Root: NewRoot(root), Name: s.Name, TmpPattern: s.TmpPattern, Label: s.Label}
}

// Apply returns f when it is already configured, otherwise it builds the spec's
// default persisted JSON file while preserving any root carried by f.
func (s Spec) Apply(f File) File {
	if f.Name != "" {
		return f
	}
	return s.File(f.Root.String())
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
	return Spec{Name: name, TmpPattern: tmpPattern, Label: label}.File(root)
}

// WithDefaults returns f when it is already configured, otherwise it builds the
// store's default persisted JSON file while preserving any root carried by f.
func (f File) WithDefaults(name, tmpPattern, label string) File {
	return Spec{Name: name, TmpPattern: tmpPattern, Label: label}.Apply(f)
}

// Path returns the persisted file path.
func (f File) Path() string {
	return f.Root.Path(f.Name)
}

// Require validates the file root with the store-specific label.
func (f File) Require() error {
	if err := f.Root.Require(f.Label); err != nil {
		return err
	}
	name := strings.TrimSpace(f.Name)
	if name == "" || name == "." || name != f.Name || filepath.IsAbs(name) || filepath.Clean(name) != name || filepath.Base(name) != name {
		return fmt.Errorf("%s file name is invalid", f.Label)
	}
	tmpPattern := strings.TrimSpace(f.TmpPattern)
	if tmpPattern != "" && (tmpPattern != f.TmpPattern || filepath.IsAbs(tmpPattern) || filepath.Clean(tmpPattern) != tmpPattern || filepath.Base(tmpPattern) != tmpPattern) {
		return fmt.Errorf("%s temp pattern is invalid", f.Label)
	}
	return nil
}

// Read decodes the persisted JSON file into value.
func (f File) Read(value any) error {
	if err := f.Require(); err != nil {
		return err
	}
	return codec.ReadJSON(f.Path(), value)
}

// WriteAtomic marshals value as indented JSON and atomically replaces the
// persisted file.
func (f File) WriteAtomic(value any, writer codec.Writer) error {
	if err := f.Require(); err != nil {
		return err
	}
	return codec.WriteAtomicJSON(f.Root.String(), f.Name, f.TmpPattern, value, writer)
}
