package setup

import (
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

func DetectVSCodeLikeTerminal(env map[string]string) string {
	if envvars.Value(env, "CURSOR_TRACE_ID") != "" {
		return "cursor"
	}
	if strings.Contains(strings.ToLower(envvars.Value(env, "VSCODE_GIT_ASKPASS_MAIN")), "windsurf") {
		return "windsurf"
	}
	if strings.EqualFold(envvars.Value(env, "TERM_PROGRAM"), "vscode") {
		return "vscode"
	}
	return ""
}

func VSCodeStyleConfigDir(app, platform string, env map[string]string, home string) string {
	switch platform {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", app, "User")
	case "win32":
		if appdata := envvars.Value(env, "APPDATA"); appdata != "" {
			return filepath.ToSlash(filepath.Join(appdata, app, "User"))
		}
		return filepath.ToSlash(filepath.Join(home, "AppData", "Roaming", app, "User"))
	default:
		return filepath.Join(home, ".config", app, "User")
	}
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
