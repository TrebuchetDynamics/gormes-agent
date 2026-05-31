package setup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/envmap"
)

func envValue(env map[string]string, key string) string {
	return envmap.Value(env, key)
}

func envHas(env map[string]string, key string) bool {
	return envmap.Has(env, key)
}

func DetectVSCodeLikeTerminal(env map[string]string) string {
	if envValue(env, "CURSOR_TRACE_ID") != "" {
		return "cursor"
	}
	if strings.Contains(strings.ToLower(envValue(env, "VSCODE_GIT_ASKPASS_MAIN")), "windsurf") {
		return "windsurf"
	}
	if strings.EqualFold(envValue(env, "TERM_PROGRAM"), "vscode") {
		return "vscode"
	}
	return ""
}

func VSCodeStyleConfigDir(app, platform string, env map[string]string, home string) string {
	switch platform {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", app, "User")
	case "win32":
		if appdata := envValue(env, "APPDATA"); appdata != "" {
			return filepath.ToSlash(filepath.Join(appdata, app, "User"))
		}
		return filepath.ToSlash(filepath.Join(home, "AppData", "Roaming", app, "User"))
	default:
		return filepath.Join(home, ".config", app, "User")
	}
}

func StripJSONComments(input string) string {
	var out strings.Builder
	inString := false
	escaped := false
	lineComment := false
	blockComment := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		var next byte
		if i+1 < len(input) {
			next = input[i+1]
		}

		if lineComment {
			if ch == '\n' {
				lineComment = false
				out.WriteByte(ch)
			}
			continue
		}
		if blockComment {
			if ch == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if inString {
			out.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		if ch == '"' {
			inString = true
			out.WriteByte(ch)
			continue
		}
		if ch == '/' && next == '/' {
			lineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			blockComment = true
			i++
			continue
		}
		out.WriteByte(ch)
	}
	return removeTrailingJSONCommas(out.String())
}

func removeTrailingJSONCommas(input string) string {
	var out strings.Builder
	inString := false
	escaped := false
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if inString {
			out.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			out.WriteByte(ch)
			continue
		}
		if ch == ',' {
			j := i + 1
			for j < len(input) && (input[j] == ' ' || input[j] == '\t' || input[j] == '\r' || input[j] == '\n') {
				j++
			}
			if j < len(input) && (input[j] == ']' || input[j] == '}') {
				continue
			}
		}
		out.WriteByte(ch)
	}
	return out.String()
}

func ConfigureDetectedTerminalKeybindings(opts TerminalSetupOptions) TerminalSetupResult {
	if isRemoteTerminal(opts.Env) {
		return TerminalSetupResult{
			Evidence: "tui_terminal_setup_remote_refused",
			Message:  "Configure terminal keybindings on the local machine, not inside an SSH session.",
		}
	}
	kind := DetectVSCodeLikeTerminal(opts.Env)
	return ConfigureTerminalKeybindings(kind, opts)
}

func ConfigureTerminalKeybindings(kind string, opts TerminalSetupOptions) TerminalSetupResult {
	if kind == "" {
		return TerminalSetupResult{Evidence: "tui_terminal_setup_unsupported", Message: "No supported VS Code-family terminal detected."}
	}
	ops := opts.FileOps.withDefaults()
	platform := opts.Platform
	if platform == "" {
		platform = "linux"
	}
	home := opts.HomeDir
	if home == "" {
		home = "."
	}
	configDir := VSCodeStyleConfigDir(vscodeAppName(kind), platform, opts.Env, home)
	path := filepath.Join(configDir, "keybindings.json")

	body, err := ops.ReadFile(path)
	existed := err == nil
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return TerminalSetupResult{
				Evidence: "tui_terminal_keybindings_read_failed",
				Message:  "Failed to read terminal keybindings.",
				Path:     path,
			}
		}
		body = []byte("[]")
	}
	bindings, err := parseKeybindings(body)
	if err != nil {
		return TerminalSetupResult{
			Evidence: "tui_terminal_keybindings_parse_failed",
			Message:  "Failed to parse terminal keybindings.",
			Path:     path,
		}
	}

	desired := defaultTerminalKeybindings(platform)
	var toAdd []map[string]any
	for _, want := range desired {
		state, conflictKey := keybindingState(bindings, want)
		switch state {
		case keybindingEquivalent:
			continue
		case keybindingConflict:
			return TerminalSetupResult{
				Evidence: "tui_terminal_keybinding_conflict",
				Message:  fmt.Sprintf("Keybinding conflict for %s.", conflictKey),
				Path:     path,
			}
		default:
			toAdd = append(toAdd, want)
		}
	}
	if len(toAdd) == 0 {
		return TerminalSetupResult{Success: true, Path: path}
	}

	if err := ops.MkdirAll(configDir, 0o755); err != nil {
		return TerminalSetupResult{Evidence: "tui_terminal_keybindings_write_failed", Message: "Failed to create terminal keybindings directory.", Path: path}
	}
	if existed {
		if err := ops.CopyFile(path, backupPath(path)); err != nil {
			return TerminalSetupResult{Evidence: "tui_terminal_keybindings_backup_failed", Message: "Failed to back up terminal keybindings.", Path: path}
		}
	}
	bindings = append(bindings, toAdd...)
	rendered, err := json.MarshalIndent(bindings, "", "  ")
	if err != nil {
		return TerminalSetupResult{Evidence: "tui_terminal_keybindings_write_failed", Message: "Failed to render terminal keybindings.", Path: path}
	}
	rendered = append(rendered, '\n')
	if err := ops.WriteFile(path, rendered, 0o644); err != nil {
		return TerminalSetupResult{Evidence: "tui_terminal_keybindings_write_failed", Message: "Failed to write terminal keybindings.", Path: path}
	}
	return TerminalSetupResult{Success: true, RequiresRestart: true, Path: path}
}

type keybindingMatchState int

const (
	keybindingMissing keybindingMatchState = iota
	keybindingEquivalent
	keybindingConflict
)

func keybindingState(existing []map[string]any, desired map[string]any) (keybindingMatchState, string) {
	desiredKey := stringField(desired, "key")
	for _, current := range existing {
		if !strings.EqualFold(stringField(current, "key"), desiredKey) {
			continue
		}
		if keybindingEquivalentTo(current, desired) {
			return keybindingEquivalent, ""
		}
		if whenClausesOverlap(stringField(current, "when"), stringField(desired, "when")) {
			return keybindingConflict, desiredKey
		}
	}
	return keybindingMissing, ""
}

func keybindingEquivalentTo(a, b map[string]any) bool {
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

func parseKeybindings(body []byte) ([]map[string]any, error) {
	body = bytes.TrimSpace([]byte(StripJSONComments(string(body))))
	if len(body) == 0 {
		body = []byte("[]")
	}
	var bindings []map[string]any
	if err := json.Unmarshal(body, &bindings); err != nil {
		return nil, err
	}
	return bindings, nil
}

func defaultTerminalKeybindings(platform string) []map[string]any {
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

func ShouldPromptForTerminalSetup(opts TerminalSetupOptions) bool {
	if isRemoteTerminal(opts.Env) {
		return false
	}
	kind := DetectVSCodeLikeTerminal(opts.Env)
	if kind == "" {
		return false
	}
	ops := opts.FileOps.withDefaults()
	platform := opts.Platform
	if platform == "" {
		platform = "darwin"
	}
	configDir := VSCodeStyleConfigDir(vscodeAppName(kind), platform, opts.Env, opts.HomeDir)
	body, err := ops.ReadFile(filepath.Join(configDir, "keybindings.json"))
	if err != nil {
		return errors.Is(err, os.ErrNotExist)
	}
	bindings, err := parseKeybindings(body)
	if err != nil {
		return true
	}
	for _, want := range defaultTerminalKeybindings(platform) {
		state, _ := keybindingState(bindings, want)
		if state != keybindingEquivalent {
			return true
		}
	}
	return false
}

func vscodeAppName(kind string) string {
	switch kind {
	case "cursor":
		return "Cursor"
	case "windsurf":
		return "Windsurf"
	default:
		return "Code"
	}
}

func isRemoteTerminal(env map[string]string) bool {
	return envValue(env, "SSH_CONNECTION") != "" || envValue(env, "SSH_TTY") != ""
}
