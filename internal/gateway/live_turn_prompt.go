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
// Production wiring resolves ProfileDir/CWD/MemoryDir from explicit context
// environment first, then configured homes and workspace ancestors; tests
// inject hermetic temp directories. Build / BuildDurable / BuildMetadata are
// seams so tests can stub them without reaching into the hermes package
// internals.
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

// defaultLiveTurnPromptSeams returns the production wiring: context dirs from
// explicit Gormes env/config locations or Gormes workspace ancestors, CWD from
// TERMINAL_CWD or os.Getwd(), and the real hermes entry points. Upstream
// HERMES_HOME is intentionally not a live persona/memory source.
//
// The slice-4 metadata clock and active model/provider getters default to nil
// so callers that have not configured them produce an empty metadata block.
// The default SelfHelpGate uses the shipped helper.
// BuildMetadata always points at hermes.BuildTurnMetadataBlock so the gate
// (Now != zero or any field set) is the only switch that controls whether
// the block actually appears.
func defaultLiveTurnPromptSeams() liveTurnPromptSeams {
	return liveTurnPromptSeams{
		ProfileDir:    func() string { return defaultLiveTurnProfileDir(defaultLiveTurnCWD()) },
		CWD:           defaultLiveTurnCWD,
		MemoryDir:     func() string { return defaultLiveTurnMemoryDir(defaultLiveTurnCWD()) },
		Build:         hermes.BuildContextFilesPrompt,
		BuildDurable:  hermes.BuildDurableUserContextPrompt,
		BuildMetadata: hermes.BuildTurnMetadataBlock,
		SelfHelpGate:  hermes.GormesSelfHelpGuidanceForPrompt,
	}
}

func defaultLiveTurnCWD() string {
	if cwd := strings.TrimSpace(os.Getenv("TERMINAL_CWD")); cwd != "" {
		return cwd
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return ""
}

func defaultLiveTurnProfileDir(cwd string) string {
	if override := strings.TrimSpace(os.Getenv("GORMES_CONTEXT_HOME")); override != "" {
		return override
	}
	gormesHome := config.GormesHome()
	if hasLiveTurnFile(gormesHome, "SOUL.md") {
		return gormesHome
	}
	if migrated := filepath.Join(gormesHome, "memory"); hasLiveTurnFile(migrated, "SOUL.md") {
		return migrated
	}
	if root := findLiveTurnAncestorWith(cwd, "SOUL.md"); root != "" {
		return root
	}
	return gormesHome
}

func defaultLiveTurnMemoryDir(cwd string) string {
	if override := strings.TrimSpace(os.Getenv("GORMES_CONTEXT_MEMORY_DIR")); override != "" {
		return override
	}
	gormesHome := config.GormesHome()
	if dir := filepath.Join(gormesHome, "memory"); hasAnyLiveTurnFile(dir, "USER.md", "MEMORY.md") {
		return dir
	}
	if dir := filepath.Join(gormesHome, "memories"); hasAnyLiveTurnFile(dir, "USER.md", "MEMORY.md") {
		return dir
	}
	if root := findLiveTurnAncestorWithAny(cwd, "USER.md", "MEMORY.md"); root != "" {
		return root
	}
	if root := findLiveTurnAncestorSubdirWithAny(cwd, "memory", "USER.md", "MEMORY.md"); root != "" {
		return root
	}
	return filepath.Join(gormesHome, "memory")
}

func findLiveTurnAncestorWith(start, name string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	for {
		if hasLiveTurnFile(dir, name) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func findLiveTurnAncestorWithAny(start string, names ...string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	for {
		if hasAnyLiveTurnFile(dir, names...) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func findLiveTurnAncestorSubdirWithAny(start, subdir string, names ...string) string {
	start = strings.TrimSpace(start)
	if start == "" {
		return ""
	}
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = start
	}
	for {
		candidate := filepath.Join(dir, subdir)
		if hasAnyLiveTurnFile(candidate, names...) {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func hasAnyLiveTurnFile(dir string, names ...string) bool {
	for _, name := range names {
		if hasLiveTurnFile(dir, name) {
			return true
		}
	}
	return false
}

func hasLiveTurnFile(dir, name string) bool {
	if strings.TrimSpace(dir) == "" || strings.TrimSpace(name) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, name))
	return err == nil && !info.IsDir()
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
