package tuigateway

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrUnknownPersonality tags errors produced by PersonalitySet.Validate when
// the caller-provided name is neither an "unset" alias (empty/none/default/
// neutral) nor present in the configured set. Downstream config.set handlers
// use errors.Is to map this to a JSON-RPC InvalidParams envelope.
var ErrUnknownPersonality = errors.New("unknown personality")

// Personality is the Go-native shape of a single `agent.personalities[name]`
// config entry. It mirrors the upstream `_render_personality_prompt` helper
// in `tui_gateway/server.py`, which accepts either a bare string (fully
// captured by SystemPrompt) or a dict with optional `tone`/`style` keys.
type Personality struct {
	SystemPrompt string
	Tone         string
	Style        string
}

// Render produces the final system prompt fragment for this personality.
// Non-empty parts are joined with newlines; empty parts are dropped so a
// personality that only sets Tone never emits a leading blank line.
func (p Personality) Render() string {
	parts := make([]string, 0, 3)
	if p.SystemPrompt != "" {
		parts = append(parts, p.SystemPrompt)
	}
	if p.Tone != "" {
		parts = append(parts, "Tone: "+p.Tone)
	}
	if p.Style != "" {
		parts = append(parts, "Style: "+p.Style)
	}
	return strings.Join(parts, "\n")
}

// PersonalitySet maps lowercase personality names to their Personality body.
// The map is intentionally a concrete type (not an interface) so JSON decoders
// targeting it produce a value that Validate can range over without copying.
type PersonalitySet map[string]Personality

// personalityUnsetAliases is the set of raw names that clear the active
// personality. Matching is case-insensitive after TrimSpace so operators can
// type any of them in a /config set call.
var personalityUnsetAliases = map[string]struct{}{
	"":        {},
	"none":    {},
	"default": {},
	"neutral": {},
}

// Validate normalizes a caller-supplied personality name and returns the
// canonical lowercase name plus its rendered prompt. Raw values matching an
// unset alias yield ("", "", nil) so the caller can clear the setting. Unknown
// names return ErrUnknownPersonality wrapped with a message that enumerates
// the configured alternatives in sorted order, matching the upstream Python
// error body so the operator-facing UX stays identical to hermes-agent.
func (s PersonalitySet) Validate(raw string) (string, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := personalityUnsetAliases[normalized]; ok {
		return "", "", nil
	}

	if p, ok := s[normalized]; ok {
		return normalized, p.Render(), nil
	}

	return "", "", fmt.Errorf("%w: %s", ErrUnknownPersonality, formatUnknownPersonalityMessage(raw, s))
}

// formatUnknownPersonalityMessage reproduces the upstream
// `_validate_personality` error body. The "Available:" line is only appended
// when the set is non-empty so an unconfigured deployment reads as "No
// personalities configured." instead of "Available: `none`".
func formatUnknownPersonalityMessage(raw string, s PersonalitySet) string {
	base := fmt.Sprintf("Unknown personality: `%s`.", raw)
	if len(s) == 0 {
		return base + "\n\nNo personalities configured."
	}

	names := make([]string, 0, len(s))
	for k := range s {
		names = append(names, k)
	}
	sort.Strings(names)

	quoted := make([]string, 0, len(names))
	for _, n := range names {
		quoted = append(quoted, "`"+n+"`")
	}
	return base + "\n\nAvailable: `none`, " + strings.Join(quoted, ", ")
}
