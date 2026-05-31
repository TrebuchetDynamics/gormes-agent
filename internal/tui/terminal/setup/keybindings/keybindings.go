package keybindings

import (
	"sort"
	"strings"
)

type MatchState int

const (
	Missing MatchState = iota
	Equivalent
	Conflict
)

func State(existing []map[string]any, desired map[string]any) (MatchState, string) {
	desiredKey := stringField(desired, "key")
	for _, current := range existing {
		if !strings.EqualFold(stringField(current, "key"), desiredKey) {
			continue
		}
		if EquivalentTo(current, desired) {
			return Equivalent, ""
		}
		if whenClausesOverlap(stringField(current, "when"), stringField(desired, "when")) {
			return Conflict, desiredKey
		}
	}
	return Missing, ""
}

func EquivalentTo(a, b map[string]any) bool {
	return strings.EqualFold(stringField(a, "key"), stringField(b, "key")) &&
		stringField(a, "command") == stringField(b, "command") &&
		normalizeWhen(stringField(a, "when")) == normalizeWhen(stringField(b, "when")) &&
		argText(a) == argText(b)
}

func whenClausesOverlap(existing, desired string) bool {
	existing = normalizeWhen(existing)
	desired = normalizeWhen(desired)
	if existing == "" || desired == "" {
		return true
	}
	if strings.Contains(existing, "editorFocus") && strings.Contains(desired, "terminalFocus") {
		return false
	}
	if strings.Contains(desired, "editorFocus") && strings.Contains(existing, "terminalFocus") {
		return false
	}
	if strings.Contains(existing, "terminalTextSelected") && strings.Contains(desired, "!terminalTextSelected") {
		return false
	}
	if strings.Contains(desired, "terminalTextSelected") && strings.Contains(existing, "!terminalTextSelected") {
		return false
	}
	if strings.Contains(existing, "terminalFocus") && strings.Contains(desired, "terminalFocus") {
		return true
	}
	return existing == desired
}

func normalizeWhen(when string) string {
	return strings.Join(strings.Fields(when), " ")
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func argText(m map[string]any) string {
	args, _ := m["args"].(map[string]any)
	text, _ := args["text"].(string)
	return text
}

func DefaultTerminalKeybindings(platform string) []map[string]any {
	bindings := []map[string]any{
		terminalSendBinding("shift+enter", "terminalFocus", "\\\r\n"),
		terminalSendBinding("ctrl+enter", "terminalFocus", "\\\r\n"),
		terminalSendBinding("cmd+enter", "terminalFocus", "\\\r\n"),
		terminalSendBinding("cmd+z", "terminalFocus", "\x1b[122;9u"),
		terminalSendBinding("shift+cmd+z", "terminalFocus", "\x1b[122;10u"),
	}
	if platform == "darwin" {
		macCopyKey := "cmd" + "+c"
		bindings = append([]map[string]any{
			terminalSendBinding(macCopyKey, "terminalFocus && terminalTextSelected", "\x1b[99;13u"),
		}, bindings...)
	}
	sort.SliceStable(bindings, func(i, j int) bool {
		return stringField(bindings[i], "key") < stringField(bindings[j], "key")
	})
	return bindings
}

func terminalSendBinding(key, when, text string) map[string]any {
	return map[string]any{
		"key":     key,
		"command": "workbench.action.terminal.sendSequence",
		"when":    when,
		"args": map[string]any{
			"text": text,
		},
	}
}
