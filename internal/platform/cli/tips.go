package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display"

// Tips is the fixed corpus of operator-facing tip strings rendered by Gormes
// CLI surfaces (banner footers, doctor hints, idle screens). Entries are owned
// by Gormes — they reference Gormes commands and concepts and intentionally do
// not mirror any upstream tip text. Each entry is unique, non-empty, and free
// of newline characters so callers can render a tip on a single line without
// further sanitisation.
var Tips = display.Tips

// TipFor returns the tip selected by the provided seed. The function is pure:
// the same seed always yields the same tip, so callers can drive deterministic
// rendering in tests by passing a fixed seed and a non-deterministic display
// by passing time.Now().UnixNano(). Negative seeds are normalised before
// indexing so the function never panics on the operator's clock skew or on a
// freshly zero-valued counter.
func TipFor(seed int64) string { return display.TipFor(seed) }
