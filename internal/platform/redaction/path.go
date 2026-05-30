package redaction

import (
	"path/filepath"
	"strings"
)

const redactedPathTailPrefix = "..."

// RedactPathTail returns a bounded display form for operator paths: only the
// final cleaned path segment is shown, prefixed with an ellipsis. Empty paths
// and root-like paths collapse to the ellipsis marker.
func RedactPathTail(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return redactedPathTailPrefix
	}
	base := filepath.Base(filepath.Clean(path))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return redactedPathTailPrefix
	}
	return redactedPathTailPrefix + "/" + base
}
