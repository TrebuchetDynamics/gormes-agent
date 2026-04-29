package gateway

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// liveTurnPromptSeams is the test seam for live-turn prompt assembly.
//
// Slice 1 (SOUL/AGENTS context-files) and slice 2 (USER.md/MEMORY.md durable
// user context) provide ProfileDir/CWD/MemoryDir + Build/BuildDurable.
// Slice 4 adds Now/ActiveModel/ActiveProvider seams plus the
// BuildMetadata renderer and the SelfHelpGate gate function so the gateway
// can inject a Hermes-format timestamp/session/model/provider block and the
// pre-shipped GormesSelfHelpGuidanceForPrompt body without a model call.
//
// Production wiring resolves ProfileDir from config.GormesHome(), CWD from
// os.Getwd(), and MemoryDir from <config.GormesHome()>/memory; tests inject
// hermetic temp directories. Build / BuildDurable / BuildMetadata are seams
// so tests can stub them without reaching into the hermes package internals.
//
// Now/ActiveModel/ActiveProvider default to nil so the metadata block is fully
// elided unless production wiring (or a test fixture) supplies them — this
// keeps the slice-1+2 byte output stable. Active session id is not a global
// seam: submitPinned passes the resolved per-turn session id into
// assembleLiveTurnPrompt so resumed/non-resumable turns render the same id the
// kernel receives.
type liveTurnPromptSeams struct {
	ProfileDir   func() string
	CWD          func() string
	MemoryDir    func() string
	Build        func(opts hermes.ContextFilesOptions) (string, hermes.ContextFilesReport)
	BuildDurable func(opts hermes.DurableUserContextOptions) (string, hermes.DurableUserContextReport)

	// Slice 4: turn-metadata seams.
	Now            func() time.Time
	ActiveModel    func() string
	ActiveProvider func() string
	BuildMetadata  func(opts hermes.TurnMetadataOptions) string

	// Slice 4: self-help guidance gate.
	SelfHelpGate func(submitText string) (string, bool)
}

// defaultLiveTurnPromptSeams returns the production wiring: profile dir from
// config.GormesHome(), CWD from os.Getwd(), memory dir from
// <config.GormesHome()>/memory, and the real hermes entry points.
//
// The slice-4 metadata clock and active model/provider getters default to nil
// so callers that have not configured them produce an empty metadata block.
// The default SelfHelpGate uses the shipped helper.
// BuildMetadata always points at hermes.BuildTurnMetadataBlock so the gate
// (Now != zero or any field set) is the only switch that controls whether
// the block actually appears.
func defaultLiveTurnPromptSeams() liveTurnPromptSeams {
	return liveTurnPromptSeams{
		ProfileDir: func() string { return config.GormesHome() },
		CWD: func() string {
			if wd, err := os.Getwd(); err == nil {
				return wd
			}
			return ""
		},
		MemoryDir:     func() string { return filepath.Join(config.GormesHome(), "memory") },
		Build:         hermes.BuildContextFilesPrompt,
		BuildDurable:  hermes.BuildDurableUserContextPrompt,
		BuildMetadata: hermes.BuildTurnMetadataBlock,
		SelfHelpGate:  hermes.GormesSelfHelpGuidanceForPrompt,
	}
}

// assembleLiveTurnPrompt composes the live-turn system prompt from five
// optional pieces in fixed order:
//
//  1. SOUL.md + project-context block (slice 1).
//  2. USER.md + MEMORY.md durable user-context block (slice 2).
//  3. Hermes-format timestamp + session/model/provider block (slice 4).
//  4. Self-help guidance body when the gate opens for submitText (slice 4).
//  5. The platform/session block produced by BuildSessionContextPrompt.
//
// Empty pieces are elided so the byte output collapses to the slice-2 layout
// when neither the metadata seams nor the self-help gate produce content.
// When all pieces are empty the result is "" (callers pass sessionBlock != ""
// in production).
func assembleLiveTurnPrompt(seams liveTurnPromptSeams, submitText, activeSessionID, sessionBlock string) (string, hermes.ContextFilesReport, hermes.DurableUserContextReport) {
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

	metadataBlock := buildMetadataFromSeams(seams, activeSessionID)
	selfHelpBlock := buildSelfHelpFromSeams(seams, submitText)

	pieces := make([]string, 0, 5)
	if strings.TrimSpace(contextBlock) != "" {
		pieces = append(pieces, contextBlock)
	}
	if strings.TrimSpace(durableBlock) != "" {
		pieces = append(pieces, durableBlock)
	}
	if strings.TrimSpace(metadataBlock) != "" {
		pieces = append(pieces, metadataBlock)
	}
	if strings.TrimSpace(selfHelpBlock) != "" {
		pieces = append(pieces, selfHelpBlock)
	}
	if sessionBlock != "" {
		pieces = append(pieces, sessionBlock)
	}
	if len(pieces) == 0 {
		return "", contextReport, durableReport
	}
	return strings.Join(pieces, "\n\n"), contextReport, durableReport
}

func buildMetadataFromSeams(seams liveTurnPromptSeams, activeSessionID string) string {
	if seams.BuildMetadata == nil {
		return ""
	}
	if seams.Now == nil && seams.ActiveModel == nil && seams.ActiveProvider == nil {
		return ""
	}
	opts := hermes.TurnMetadataOptions{SessionID: strings.TrimSpace(activeSessionID)}
	if seams.Now != nil {
		opts.Now = seams.Now()
	}
	if seams.ActiveModel != nil {
		opts.Model = strings.TrimSpace(seams.ActiveModel())
	}
	if seams.ActiveProvider != nil {
		opts.Provider = strings.TrimSpace(seams.ActiveProvider())
	}
	return seams.BuildMetadata(opts)
}

func buildSelfHelpFromSeams(seams liveTurnPromptSeams, submitText string) string {
	if seams.SelfHelpGate == nil {
		return ""
	}
	body, ok := seams.SelfHelpGate(submitText)
	if !ok {
		return ""
	}
	return body
}
