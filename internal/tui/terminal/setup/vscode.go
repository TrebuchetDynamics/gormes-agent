package setup

import vscodeconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal/setup/vscode"

const (
	vscodeKindCursor   = vscodeconfig.KindCursor
	vscodeKindVSCode   = vscodeconfig.KindVSCode
	vscodeKindWindsurf = vscodeconfig.KindWindsurf
)

func DetectVSCodeLikeTerminal(env map[string]string) string {
	return vscodeconfig.DetectLikeTerminal(env)
}

func VSCodeStyleConfigDir(app, platform string, env map[string]string, home string) string {
	return vscodeconfig.VSCodeStyleConfigDir(app, platform, env, home)
}

func vscodeAppName(kind string) string {
	return vscodeconfig.AppName(kind)
}
