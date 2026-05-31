package setup

import (
	"path/filepath"
	"strings"
)

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
