package storage

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage/codec"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage/decode"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/storage/persisted"
)

// Root is the shared caller-owned persistence root contract for directory
// stores. It centralizes root normalization, file path construction, and empty
// root validation across cache and remembered-source stores.
type Root = persisted.Root

// NewRoot returns a normalized caller-owned persistence root.
func NewRoot(root string) Root { return persisted.NewRoot(root) }

// Spec names one persisted JSON file used by a directory store. It centralizes
// file names, temp-file patterns, and root-validation labels so stores can
// share persistence metadata without each rebuilding the same contract.
type Spec = persisted.Spec

// File is the shared persisted-JSON file contract for directory stores. It
// keeps path construction, root validation, and atomic-write metadata together
// so cache and source stores do not each rebuild that policy.
type File = persisted.File

// NewFile returns a persisted JSON file rooted at root.
func NewFile(root, name, tmpPattern, label string) File {
	return persisted.NewFile(root, name, tmpPattern, label)
}

// LoadValue reads a persisted JSON value using a caller-supplied empty value
// and post-decode normalization hook. Directory stores use this to share the
// same decode lifecycle while keeping their own missing/invalid evidence policy.
func LoadValue[T any](file File, empty func() T, ensure func(T) T) (T, error) {
	return decode.LoadValue(file, empty, ensure)
}

// Writer is the injectable write seam used by stores that need deterministic
// atomic-write failure tests.
type Writer = codec.Writer

// ReadJSON reads path into value using the stores' shared JSON decoding seam.
func ReadJSON(path string, value any) error { return codec.ReadJSON(path, value) }

// WriteAtomicJSON marshals value as indented JSON and atomically replaces name
// under root. The temporary file is created in root so rename stays atomic on
// normal local filesystems.
func WriteAtomicJSON(root, name, tmpPattern string, value any, writer Writer) error {
	file := persisted.NewFile(root, name, tmpPattern, "directory json")
	if err := file.Require(); err != nil {
		return err
	}
	return codec.WriteAtomicJSON(file.Root.String(), file.Name, file.TmpPattern, value, writer)
}
