package skills

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/install"
)

// URLFetcher fetches the bytes of a remote SKILL.md. It is the only seam
// through which network IO enters URL install policy.
type URLFetcher = install.URLFetcher

// QuarantineScanner runs the downloaded SKILL.md bytes through quarantine
// scan. It returns ok=false to leave the active store untouched.
type QuarantineScanner = install.QuarantineScanner

// SkillStore is the minimal surface URL install policy needs to write a
// SKILL.md into the active store. It is satisfied by an in-memory test fake
// or a real on-disk implementation.
type SkillStore = install.SkillStore

// InteractiveConsole is a placeholder for interactive prompts. nil means
// non-interactive surface — URL installs that require a name fail closed
// with retry guidance.
type InteractiveConsole = install.InteractiveConsole

// URLInstallPolicy bundles the seams URL install needs.
type URLInstallPolicy = install.URLInstallPolicy

// URLInstallRequest is one direct-URL install request.
type URLInstallRequest = install.URLInstallRequest

// URLInstallEvidence records the outcome of a URL install attempt.
type URLInstallEvidence = install.URLInstallEvidence

func PerformURLInstall(ctx context.Context, p URLInstallPolicy, req URLInstallRequest) URLInstallEvidence {
	return install.PerformURLInstall(ctx, p, req)
}
