// Package tuigateway hosts the Go-native port of Hermes' tui_gateway server.
// This file ports the pure-helper portion only — no transport, no session
// state, no config-file reads. Each helper mirrors a single function in
// hermes-agent/tui_gateway/server.py so the upstream invariants stay
// reviewable side-by-side.
package tuigateway

import "strings"

// NormalizeToolProgressMode maps a raw config value to one of the four
// recognised tool-progress modes: "off", "new", "all", or "verbose".
//
// It mirrors hermes-agent/tui_gateway/server.py:_load_tool_progress_mode
// (line 661) so the same Python config payload, once decoded into Go's
// any, produces the same mode string Hermes would have chosen:
//
//   - bool false → "off"
//   - bool true  → "all"
//   - string is trimmed and lowercased; if it lands in the recognised set
//     it is returned, otherwise it falls back to "all"
//   - anything else (nil, numbers, slices, …) also falls back to "all",
//     matching upstream's `str(raw or "all")` catch-all
//
// The helper has no side effects: no file reads, no session lookups, no
// goroutines. Callers that need session- or config-scoped behaviour
// compose this helper into their own lookup path.
func NormalizeToolProgressMode(raw any) string {
	switch v := raw.(type) {
	case bool:
		if v {
			return "all"
		}
		return "off"
	case string:
		mode := strings.ToLower(strings.TrimSpace(v))
		switch mode {
		case "off", "new", "all", "verbose":
			return mode
		}
	}
	return "all"
}
