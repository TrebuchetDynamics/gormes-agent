package display

import tipsmodel "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display/tips"

// Tips is the fixed corpus of operator-facing tip strings rendered by Gormes
// CLI surfaces (banner footers, doctor hints, idle screens).
var Tips = tipsmodel.Corpus

// TipFor returns the tip selected by the provided seed.
func TipFor(seed int64) string { return tipsmodel.For(seed) }
