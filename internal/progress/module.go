package progress

// The grill-corrected module-split taxonomy (2026-05-16): module names are
// feature homes for the physical split, while execution_owner remains work
// ownership. Keep this list alphabetized so filenames, validation messages,
// and generated module views stay stable.
const (
	ModuleBrowser      = "browser"
	ModuleBuilder      = "builder"
	ModuleChannels     = "channels"
	ModuleCLI          = "cli"
	ModuleConfig       = "config"
	ModuleCrossCutting = "cross-cutting"
	ModuleDoctor       = "doctor"
	ModuleDocs         = "docs"
	ModuleFleet        = "fleet"
	ModuleGateway      = "gateway"
	ModuleGoncho       = "goncho"
	ModuleInstall      = "install"
	ModuleKanban       = "kanban"
	ModuleLanding      = "landing"
	ModuleLearningLoop = "learning-loop"
	ModuleMemory       = "memory"
	ModuleNavivox      = "navivox"
	ModulePlanner      = "planner"
	ModuleProfiles     = "profiles"
	ModuleProgress     = "progress"
	ModuleProviders    = "providers"
	ModuleRelease      = "release"
	ModuleRuntime      = "runtime"
	ModuleSessions     = "sessions"
	ModuleSkills       = "skills"
	ModuleSTT          = "stt"
	ModuleTools        = "tools"
	ModuleTTS          = "tts"
	ModuleTUI          = "tui"

	// ModuleUnclassified is the explicit deterministic bucket for rows that
	// carry no execution_owner and no explicit module. It is a compatibility
	// fallback only, not a valid final physical module for C5.
	ModuleUnclassified = "unclassified"
)

var allowedModules = []string{
	ModuleBrowser,
	ModuleBuilder,
	ModuleChannels,
	ModuleCLI,
	ModuleConfig,
	ModuleCrossCutting,
	ModuleDoctor,
	ModuleDocs,
	ModuleFleet,
	ModuleGateway,
	ModuleGoncho,
	ModuleInstall,
	ModuleKanban,
	ModuleLanding,
	ModuleLearningLoop,
	ModuleMemory,
	ModuleNavivox,
	ModulePlanner,
	ModuleProfiles,
	ModuleProgress,
	ModuleProviders,
	ModuleRelease,
	ModuleRuntime,
	ModuleSessions,
	ModuleSkills,
	ModuleSTT,
	ModuleTools,
	ModuleTTS,
	ModuleTUI,
}

var allowedModuleSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(allowedModules))
	for _, module := range allowedModules {
		set[module] = struct{}{}
	}
	return set
}()

// AllowedModules returns the closed, stable feature-module taxonomy accepted
// by progress validation. The returned slice is a copy.
func AllowedModules() []string {
	return append([]string(nil), allowedModules...)
}

// ValidModule reports whether module is one of the approved physical feature
// buckets. Empty strings are intentionally invalid here; Validate treats them
// separately as "not assigned yet" until C5g makes explicit modules universal.
func ValidModule(module string) bool {
	_, ok := allowedModuleSet[module]
	return ok
}

// moduleForOwner maps the coarse execution_owner enum to a compatibility
// taxonomy. An empty string means "not derivable from owner alone".
func moduleForOwner(owner ExecutionOwner) string {
	switch owner {
	case ExecutionOwnerDocs:
		return ModuleDocs
	case ExecutionOwnerGateway:
		return ModuleGateway
	case ExecutionOwnerMemory:
		return ModuleMemory
	case ExecutionOwnerSkills:
		return ModuleSkills
	case ExecutionOwnerGoncho:
		return ModuleGoncho
	case ExecutionOwnerProvider:
		return ModuleProviders
	case ExecutionOwnerTools:
		return ModuleTools
	case ExecutionOwnerOrchestrator:
		return ModuleFleet
	case ExecutionOwnerTui:
		return ModuleTUI
	default:
		return ""
	}
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
