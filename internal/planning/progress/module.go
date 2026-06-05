package progress

import "github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/featuremodule"

const (
	ModuleBrowser      = featuremodule.ModuleBrowser
	ModuleBuilder      = featuremodule.ModuleBuilder
	ModuleChannels     = featuremodule.ModuleChannels
	ModuleCLI          = featuremodule.ModuleCLI
	ModuleConfig       = featuremodule.ModuleConfig
	ModuleCrossCutting = featuremodule.ModuleCrossCutting
	ModuleDoctor       = featuremodule.ModuleDoctor
	ModuleDocs         = featuremodule.ModuleDocs
	ModuleFleet        = featuremodule.ModuleFleet
	ModuleGateway      = featuremodule.ModuleGateway
	ModuleGoncho       = featuremodule.ModuleGoncho
	ModuleInstall      = featuremodule.ModuleInstall
	ModuleKanban       = featuremodule.ModuleKanban
	ModuleLanding      = featuremodule.ModuleLanding
	ModuleLearningLoop = featuremodule.ModuleLearningLoop
	ModuleMemory       = featuremodule.ModuleMemory
	ModuleNavivox      = featuremodule.ModuleNavivox
	ModulePlanner      = featuremodule.ModulePlanner
	ModuleProfiles     = featuremodule.ModuleProfiles
	ModuleProgress     = featuremodule.ModuleProgress
	ModuleProviders    = featuremodule.ModuleProviders
	ModuleRelease      = featuremodule.ModuleRelease
	ModuleRuntime      = featuremodule.ModuleRuntime
	ModuleSessions     = featuremodule.ModuleSessions
	ModuleSkills       = featuremodule.ModuleSkills
	ModuleSTT          = featuremodule.ModuleSTT
	ModuleTools        = featuremodule.ModuleTools
	ModuleTTS          = featuremodule.ModuleTTS
	ModuleTUI          = featuremodule.ModuleTUI

	// ModuleUnclassified is the explicit deterministic bucket for rows that
	// carry no execution_owner and no explicit module. It is a compatibility
	// fallback only, not a valid final physical module for C5.
	ModuleUnclassified = featuremodule.ModuleUnclassified
)

// AllowedModules returns the closed, stable feature-module taxonomy accepted
// by progress validation. The returned slice is a copy.
func AllowedModules() []string {
	return featuremodule.Allowed()
}

// ValidModule reports whether module is one of the approved physical feature
// buckets. Empty strings are intentionally invalid here; Validate treats them
// separately as "not assigned yet" until C5g makes explicit modules universal.
func ValidModule(module string) bool {
	return featuremodule.Valid(module)
}

// moduleForOwner maps the coarse execution_owner enum to a compatibility
// taxonomy. An empty string means "not derivable from owner alone".
func moduleForOwner(owner ExecutionOwner) string {
	return featuremodule.ForExecutionOwner(string(owner))
}

// Module returns the deterministic module bucket for a row: an explicit
// it.Module always wins; otherwise it derives from ExecutionOwner via the
// compatibility taxonomy; with neither, it is the explicit ModuleUnclassified
// bucket. phaseID/subphaseID are accepted for a future fallback and
// kept in the signature so callers (and C5) stay stable; the current
// derivation is intentionally owner-or-unclassified so it is never a silent
// mis-bucket. Module never errors.
func Module(it Item, phaseID, subphaseID string) string {
	if it.Module != "" {
		return it.Module
	}
	if m := moduleForOwner(it.ExecutionOwner); m != "" {
		return m
	}
	return ModuleUnclassified
}

// BackfillModules sets Item.Module on every row where it is empty AND
// deterministically derivable from execution_owner, leaving every other
// field untouched. Rows with no owner are left unset (Module() still returns
// ModuleUnclassified at read time) so they remain visibly pending rather
// than guessed. It returns the number of rows changed and is idempotent: a
// second pass over an already-backfilled backlog returns 0. It is a
// standalone explicit maintenance action — validate/write/compact never
// invoke it — so doc regeneration and schema validation stay pure, and the
// change is fully git-reversible.
func BackfillModules(p *Progress) int {
	if p == nil {
		return 0
	}
	n := 0
	for pk := range p.Phases {
		ph := p.Phases[pk]
		for sk := range ph.Subphases {
			sp := ph.Subphases[sk]
			for i := range sp.Items {
				if sp.Items[i].Module != "" {
					continue
				}
				m := moduleForOwner(sp.Items[i].ExecutionOwner)
				if m == "" {
					continue
				}
				sp.Items[i].Module = m
				n++
			}
		}
	}
	return n
}
