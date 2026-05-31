package envvars

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/envmap"

const (
	AppData              = "APPDATA"
	ColorTerm            = "COLORTERM"
	CursorTraceID        = "CURSOR_TRACE_ID"
	ForceColor           = "FORCE_COLOR"
	HermesTUITruecolor   = "HERMES_TUI_TRUECOLOR"
	NoColor              = "NO_COLOR"
	SSHConnection        = "SSH_CONNECTION"
	SSHTTY               = "SSH_TTY"
	TermProgram          = "TERM_PROGRAM"
	TMUX                 = "TMUX"
	VSCodeGitAskpassMain = "VSCODE_GIT_ASKPASS_MAIN"
	WaylandDisplay       = "WAYLAND_DISPLAY"
	WSLInterop           = "WSL_INTEROP"
)

// Value returns the terminal environment value for key using the shared TUI
// environment-map semantics.
func Value(env map[string]string, key string) string {
	return envmap.Value(env, key)
}

// Has reports whether key is present using the shared TUI environment-map
// semantics.
func Has(env map[string]string, key string) bool {
	return envmap.Has(env, key)
}

// IsRemote reports whether the terminal appears to run inside SSH. Terminal
// setup policy uses this to avoid writing local keybinding files from a remote
// shell.
func IsRemote(env map[string]string) bool {
	return Value(env, SSHConnection) != "" || Value(env, SSHTTY) != ""
}
