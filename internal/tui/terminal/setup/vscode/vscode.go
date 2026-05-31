package vscode

import (
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

const (
	KindCursor   = "cursor"
	KindVSCode   = envvars.VSCodeTermProgram
	KindWindsurf = "windsurf"
)

func DetectLikeTerminal(env map[string]string) string {
	if envvars.Value(env, envvars.CursorTraceID) != "" {
		return KindCursor
	}
	if strings.Contains(strings.ToLower(envvars.Value(env, envvars.VSCodeGitAskpassMain)), KindWindsurf) {
		return KindWindsurf
	}
	if strings.EqualFold(envvars.Value(env, envvars.TermProgram), envvars.VSCodeTermProgram) {
		return KindVSCode
	}
	return ""
}

func VSCodeStyleConfigDir(app, platform string, env map[string]string, home string) string {
	switch platform {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", app, "User")
	case "win32":
		if appdata := envvars.Value(env, envvars.AppData); appdata != "" {
			return filepath.ToSlash(filepath.Join(appdata, app, "User"))
		}
		return filepath.ToSlash(filepath.Join(home, "AppData", "Roaming", app, "User"))
	default:
		return filepath.Join(home, ".config", app, "User")
	}
}

func AppName(kind string) string {
	switch kind {
	case KindCursor:
		return "Cursor"
	case KindWindsurf:
		return "Windsurf"
	default:
		return "Code"
	}
}
