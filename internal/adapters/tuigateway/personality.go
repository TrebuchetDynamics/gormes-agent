package tuigateway

import (
	"fmt"
	"sort"
	"strings"
)

// personalityAliases enumerates the upstream "blank" set:
// hermes-agent/tui_gateway/server.py:_validate_personality treats the empty
// string and the literal aliases "none", "default", and "neutral" as the
// "no overlay" instruction.
var personalityAliases = map[string]struct{}{
	"none":    {},
	"default": {},
	"neutral": {},
}

// RenderPersonalityPrompt mirrors hermes-agent/tui_gateway/server.py:
// _render_personality_prompt. A personality entry is either a structured
// map carrying system_prompt/tone/style or a bare string.
//
//   - Map values render as up to three newline-separated lines:
//     system_prompt, then "Tone: <tone>" if non-empty, then "Style: <style>"
//     if non-empty. Blank fields are dropped (matching the upstream
//     `"\n".join(p for p in parts if p)` join).
//   - Anything else is coerced through fmt.Sprint to mirror upstream's
//     `return str(value)` fallback so dynamically-typed config values
//     (numbers, booleans) round-trip predictably.
//
// The helper is pure: no I/O, no globals.
func RenderPersonalityPrompt(value any) string {
	if value == nil {
		return ""
	}
	if dict, ok := value.(map[string]any); ok {
		var parts []string
		if sp, _ := dict["system_prompt"].(string); sp != "" {
			parts = append(parts, sp)
		}
		if tone, _ := dict["tone"].(string); tone != "" {
			parts = append(parts, "Tone: "+tone)
		}
		if style, _ := dict["style"].(string); style != "" {
			parts = append(parts, "Style: "+style)
		}
		return strings.Join(parts, "\n")
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

// ValidatePersonality mirrors hermes-agent/tui_gateway/server.py:
// _validate_personality. It returns:
//
//   - ("", "", nil) when the input is blank or matches one of the aliases
//     ("none", "default", "neutral"); callers interpret this as "clear the
//     overlay" without surfacing an error.
//   - (name, prompt, nil) when the trimmed lowercase name is present in
//     the personalities map, where prompt is RenderPersonalityPrompt of
//     the matched value.
//   - ("", "", err) when the name does not match. The error message is
//     deterministic: it preserves the operator's raw casing in
//     `Unknown personality: \`<raw>\`.` and lists the configured choices
//     sorted alphabetically as "\n\nAvailable: \`none\`, \`x\`, \`y\`".
//     When no personalities are configured the suffix becomes
//     "\n\nNo personalities configured." instead.
//
// The helper does not read configuration on its own — callers pass the
// already-loaded personalities map so the validator stays pure.
func ValidatePersonality(value string, personalities map[string]any) (string, string, error) {
	raw := strings.TrimSpace(value)
	name := strings.ToLower(raw)
	if name == "" {
		return "", "", nil
	}
	if _, ok := personalityAliases[name]; ok {
		return "", "", nil
	}

	if entry, ok := personalities[name]; ok {
		return name, RenderPersonalityPrompt(entry), nil
	}

	msg := fmt.Sprintf("Unknown personality: `%s`.", raw)
	if len(personalities) == 0 {
		msg += "\n\nNo personalities configured."
		return "", "", fmt.Errorf("%s", msg)
	}

	names := make([]string, 0, len(personalities))
	for n := range personalities {
		names = append(names, n)
	}
	sort.Strings(names)
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = "`" + n + "`"
	}
	msg += "\n\nAvailable: `none`, " + strings.Join(quoted, ", ")
	return "", "", fmt.Errorf("%s", msg)
}
