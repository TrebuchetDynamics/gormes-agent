package progress

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/markdown"

// ReplaceMarker replaces the content between PROGRESS:START kind=<kind>
// and PROGRESS:END with the supplied body. The markers themselves are
// preserved. Returns an error if the markers are missing, unbalanced,
// or the start marker's kind does not match.
func ReplaceMarker(input, kind, body string) (string, error) {
	return markdown.ReplaceMarker(input, kind, body)
}
