// Package shared contains migration types and helpers used by source-specific
// migration packages. Types live here only after Hermes and OpenClaw prove
// they need the same structs.
package shared

// RedactedValue replaces every secret value in migration manifests and
// outcomes. Raw secret bytes must never enter exported data.
const RedactedValue = "[REDACTED]"

// SourceReport records every candidate path the resolver considered
// plus the one that was actually selected.
type SourceReport struct {
	Selected     string            `json:"selected"`
	SelectedPath string            `json:"selected_path"`
	Candidates   []SourceCandidate `json:"candidates"`
}

// SourceCandidate captures one candidate migration-source directory.
type SourceCandidate struct {
	Origin string `json:"origin"`
	Path   string `json:"path"`
	Found  bool   `json:"found"`
}

// ErrorEntry records a non-fatal parse or read error so dry-run callers
// can decide whether to proceed without panicking the manifest builder.
type ErrorEntry struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

// Candidate builds a SourceCandidate entry using shared.DirExists to
// determine the Found flag. It is the canonical constructor for both
// Hermes and OpenClaw source resolution.
func Candidate(origin, path string) SourceCandidate {
	return SourceCandidate{Origin: origin, Path: path, Found: DirExists(path)}
}

// FirstFoundCandidate returns the first resolved candidate whose Found
// flag is true, together with a boolean indicating whether any such
// candidate exists.
func FirstFoundCandidate(cs []SourceCandidate) (SourceCandidate, bool) {
	for _, c := range cs {
		if c.Found {
			return c, true
		}
	}
	return SourceCandidate{}, false
}
