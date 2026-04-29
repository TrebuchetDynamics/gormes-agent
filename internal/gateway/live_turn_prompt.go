package gateway

import (
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// liveTurnPromptSeams is the test seam for live-turn context-file resolution.
// Production wiring resolves ProfileDir from config.GormesHome() and CWD from
// os.Getwd(); tests inject hermetic temp directories. Build is the
// hermes.BuildContextFilesPrompt entry point and is a seam so tests can stub
// it if needed without reaching into the hermes package internals.
type liveTurnPromptSeams struct {
	ProfileDir func() string
	CWD        func() string
	Build      func(opts hermes.ContextFilesOptions) (string, hermes.ContextFilesReport)
}

// defaultLiveTurnPromptSeams returns the production wiring: profile dir from
// config.GormesHome(), CWD from os.Getwd(), and the real
// hermes.BuildContextFilesPrompt entry point.
func defaultLiveTurnPromptSeams() liveTurnPromptSeams {
	return liveTurnPromptSeams{
		ProfileDir: func() string { return config.GormesHome() },
		CWD: func() string {
			if wd, err := os.Getwd(); err == nil {
				return wd
			}
			return ""
		},
		Build: hermes.BuildContextFilesPrompt,
	}
}

// assembleLiveTurnPrompt prepends the SOUL.md + project context block to the
// existing platform/session block. When the context-files block is empty the
// returned string is byte-identical to sessionBlock so existing kernel
// session-context tests keep passing.
func assembleLiveTurnPrompt(seams liveTurnPromptSeams, sessionBlock string) (string, hermes.ContextFilesReport) {
	if seams.Build == nil {
		return sessionBlock, hermes.ContextFilesReport{}
	}
	opts := hermes.ContextFilesOptions{}
	if seams.ProfileDir != nil {
		opts.ProfileDir = strings.TrimSpace(seams.ProfileDir())
	}
	if seams.CWD != nil {
		opts.CWD = strings.TrimSpace(seams.CWD())
	}
	contextBlock, report := seams.Build(opts)
	if strings.TrimSpace(contextBlock) == "" {
		return sessionBlock, report
	}
	if sessionBlock == "" {
		return contextBlock, report
	}
	return contextBlock + "\n\n" + sessionBlock, report
}
