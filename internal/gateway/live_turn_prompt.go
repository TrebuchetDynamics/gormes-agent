package gateway

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// liveTurnPromptSeams is the test seam for live-turn context-file resolution.
// Production wiring resolves ProfileDir from config.GormesHome(), CWD from
// os.Getwd(), and MemoryDir from <config.GormesHome()>/memory; tests inject
// hermetic temp directories. Build / BuildDurable are seams so tests can stub
// them without reaching into the hermes package internals.
type liveTurnPromptSeams struct {
	ProfileDir   func() string
	CWD          func() string
	MemoryDir    func() string
	Build        func(opts hermes.ContextFilesOptions) (string, hermes.ContextFilesReport)
	BuildDurable func(opts hermes.DurableUserContextOptions) (string, hermes.DurableUserContextReport)
}

// defaultLiveTurnPromptSeams returns the production wiring: profile dir from
// config.GormesHome(), CWD from os.Getwd(), memory dir from
// <config.GormesHome()>/memory, and the real hermes entry points.
func defaultLiveTurnPromptSeams() liveTurnPromptSeams {
	return liveTurnPromptSeams{
		ProfileDir: func() string { return config.GormesHome() },
		CWD: func() string {
			if wd, err := os.Getwd(); err == nil {
				return wd
			}
			return ""
		},
		MemoryDir:    func() string { return filepath.Join(config.GormesHome(), "memory") },
		Build:        hermes.BuildContextFilesPrompt,
		BuildDurable: hermes.BuildDurableUserContextPrompt,
	}
}

// assembleLiveTurnPrompt composes the live-turn system prompt from three
// optional pieces in fixed order:
//
//  1. SOUL.md + project-context block (slice 1).
//  2. USER.md + MEMORY.md durable user-context block (slice 2).
//  3. The platform/session block produced by BuildSessionContextPrompt.
//
// Empty pieces are elided so the byte output collapses to the slice-1 layout
// when the durable user dir is missing or empty. When all three pieces are
// empty the result is "" (callers pass sessionBlock != "" in production).
func assembleLiveTurnPrompt(seams liveTurnPromptSeams, sessionBlock string) (string, hermes.ContextFilesReport, hermes.DurableUserContextReport) {
	var (
		contextBlock  string
		contextReport hermes.ContextFilesReport
		durableBlock  string
		durableReport hermes.DurableUserContextReport
	)

	if seams.Build != nil {
		opts := hermes.ContextFilesOptions{}
		if seams.ProfileDir != nil {
			opts.ProfileDir = strings.TrimSpace(seams.ProfileDir())
		}
		if seams.CWD != nil {
			opts.CWD = strings.TrimSpace(seams.CWD())
		}
		contextBlock, contextReport = seams.Build(opts)
	}

	if seams.BuildDurable != nil {
		opts := hermes.DurableUserContextOptions{}
		if seams.MemoryDir != nil {
			opts.MemoryDir = strings.TrimSpace(seams.MemoryDir())
		}
		durableBlock, durableReport = seams.BuildDurable(opts)
	}

	pieces := make([]string, 0, 3)
	if strings.TrimSpace(contextBlock) != "" {
		pieces = append(pieces, contextBlock)
	}
	if strings.TrimSpace(durableBlock) != "" {
		pieces = append(pieces, durableBlock)
	}
	if sessionBlock != "" {
		pieces = append(pieces, sessionBlock)
	}
	if len(pieces) == 0 {
		return "", contextReport, durableReport
	}
	return strings.Join(pieces, "\n\n"), contextReport, durableReport
}
