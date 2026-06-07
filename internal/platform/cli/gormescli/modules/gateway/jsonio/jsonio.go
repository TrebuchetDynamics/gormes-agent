package jsonio

import (
	"encoding/json"
	"io"
)

// WriteIndented writes value using the stable pretty JSON shape expected by
// gateway CLI --json output: two-space indentation and a trailing newline.
func WriteIndented(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
