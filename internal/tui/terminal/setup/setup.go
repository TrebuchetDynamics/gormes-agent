package setup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/fileops"
)

func ConfigureDetectedTerminalKeybindings(opts TerminalSetupOptions) TerminalSetupResult {
	if envvars.IsRemote(opts.Env) {
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
	ops := fileops.WithDefaults(opts.FileOps)
	platform := opts.Platform
	if platform == "" {
		platform = "linux"
	}
	home := opts.HomeDir
	if home == "" {
		home = "."
	}
	path := vscodeKeybindingsPath(kind, platform, opts.Env, home)
	configDir := filepath.Dir(path)

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
		if err := ops.CopyFile(path, fileops.BackupPath(path)); err != nil {
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

func ShouldPromptForTerminalSetup(opts TerminalSetupOptions) bool {
	if envvars.IsRemote(opts.Env) {
		return false
	}
	kind := DetectVSCodeLikeTerminal(opts.Env)
	if kind == "" {
		return false
	}
	ops := fileops.WithDefaults(opts.FileOps)
	platform := opts.Platform
	if platform == "" {
		platform = "darwin"
	}
	path := vscodeKeybindingsPath(kind, platform, opts.Env, opts.HomeDir)
	body, err := ops.ReadFile(path)
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
