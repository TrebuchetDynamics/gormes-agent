package setup

import (
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/envvars"
)

const (
	vscodeKindCursor   = "cursor"
	vscodeKindVSCode   = envvars.VSCodeTermProgram
	vscodeKindWindsurf = "windsurf"
)

func DetectVSCodeLikeTerminal(env map[string]string) string {
	if envvars.Value(env, envvars.CursorTraceID) != "" {
		return vscodeKindCursor
	}
	if strings.Contains(strings.ToLower(envvars.Value(env, envvars.VSCodeGitAskpassMain)), vscodeKindWindsurf) {
		return vscodeKindWindsurf
	}
	if strings.EqualFold(envvars.Value(env, envvars.TermProgram), envvars.VSCodeTermProgram) {
		return vscodeKindVSCode
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

func vscodeAppName(kind string) string {
	switch kind {
	case vscodeKindCursor:
		return "Cursor"
	case vscodeKindWindsurf:
		return "Windsurf"
	default:
		return "Code"
	}
}
