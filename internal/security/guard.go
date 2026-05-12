package security

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// Guard composes multiple security check sources (Tirith findings, path-based
// allowlists, URL safety rules, website policies) into a single allow/deny
// decision. The composer resolves conflicts deterministically:
//   - deny wins over allow (any source that denies → overall deny),
//   - policy overrides Tirith (URL/website policy can explicitly allow).
// Always returns typed evidence explaining the decision.
type Guard struct {
	tirithClient  *TirithClient
	pathAllowlist []string
	urlSafety     *tools.URLSafetyChecker
}

// NewGuard creates a Guard with no check sources loaded.
func NewGuard() *Guard {
	return &Guard{}
}

// SetTirith configures the Guard to use the given TirithClient for
// security finding checks.
func (g *Guard) SetTirith(c *TirithClient) {
	g.tirithClient = c
}

// SetPathAllowlist configures a list of allowed file/directory paths that
// bypass Tirith deny when the operation targets one of them.
func (g *Guard) SetPathAllowlist(paths []string) {
	g.pathAllowlist = paths
}

// SetURLSafety configures the Guard to use the given URLSafetyChecker for
// URL safety policy evaluation. URL policy overrides Tirith deny.
func (g *Guard) SetURLSafety(c *tools.URLSafetyChecker) {
	g.urlSafety = c
}

// GuardEvidence is the typed result of a Guard.Compose call.
type GuardEvidence struct {
	Allow        bool
	EvidenceType string
	Reason       string
}

// Compose combines all loaded check sources and returns the unified decision
// for the given target (path or URL).
// Priority:
//  1. URL safety policy (highest — can explicitly allow overriding Tirith).
//  2. Tirith findings (deny wins over path allowlist).
//  3. Path allowlist.
// If no sources are loaded, returns allow with guard_no_policies evidence.
func (g *Guard) Compose(target string) GuardEvidence {
	// Step 1: URL safety policy — highest priority, overrides Tirith.
	if g.urlSafety != nil {
		result := g.urlSafety.CheckURL(target)
		if result.Safe {
			// Policy says safe, even if Tirith disagrees.
			return GuardEvidence{
				Allow:        true,
				EvidenceType: "guard_allow",
				Reason:       "URL policy permits: " + result.Reason,
			}
		}
		// URL policy says unsafe — return deny.
		return GuardEvidence{
			Allow:        false,
			EvidenceType: "guard_deny",
			Reason:       "URL policy blocks: " + result.Reason,
		}
	}

	// Step 2: Tirith findings — deny if critical/high.
	if g.tirithClient != nil {
		tev := g.tirithClient.Decision()
		if !tev.Allow {
			return GuardEvidence{
				Allow:        false,
				EvidenceType: "guard_deny",
				Reason:       tev.Reason,
			}
		}
	}

	// Step 3: Path allowlist — check target against allowed paths.
	if len(g.pathAllowlist) > 0 {
		for _, allowed := range g.pathAllowlist {
			if target == allowed {
				evType, reason := "guard_allow", "Path allowlist matched: "+allowed
				return GuardEvidence{
					Allow:        true,
					EvidenceType: evType,
					Reason:       reason,
				}
			}
		}
		return GuardEvidence{
			Allow:        false,
			EvidenceType: "guard_deny",
			Reason:       "Path not in allowlist: " + target,
		}
	}

	// No policies loaded.
	return GuardEvidence{
		Allow:        true,
		EvidenceType: "guard_no_policies",
		Reason:       "No security policies loaded — all operations allowed",
	}
}
