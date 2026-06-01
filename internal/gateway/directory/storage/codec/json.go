package codec

import (
	"context"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/jsonfile"
)

// Writer is the injectable write seam used by stores that need deterministic
// atomic-write failure tests.
type Writer = jsonfile.Writer

// ReadJSON reads path into value using the stores' shared JSON decoding seam.
func ReadJSON(path string, value any) error {
	return jsonfile.ReadRequired(context.Background(), path, value, "directory json")
}

// WriteAtomicJSON marshals value as indented JSON and atomically replaces name
// under root. The temporary file is created in root so rename stays atomic on
// normal local filesystems.
func WriteAtomicJSON(root, name, tmpPattern string, value any, writer Writer) error {
	return jsonfile.WriteAtomicWithOptions(context.Background(), filepath.Join(root, name), value, name, jsonfile.WriteOptions{
		DirMode:    0o700,
		FileMode:   0o600,
		TmpPattern: tmpPattern,
		Writer:     writer,
	})
}
