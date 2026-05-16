package progress

// The agreed module-split taxonomy (2026-05-16 planner pass): the user's
// requested fine module set reconciled with the gormes-hermes-parity Source
// Buckets. These are the canonical per-subsystem buckets a later child (C5)
// keys the on-disk per-module split by.
const (
	ModuleCommands           = "commands"
	ModuleSetupConfigInstall = "setup-config-install"
	ModuleGatewayChannels    = "gateway-channels"
	ModuleTools              = "tools"
	ModuleProvidersAuth      = "providers-auth"
	ModuleTUI                = "tui"
	ModuleMemorySessions     = "memory-sessions-skills"
	ModuleOrchestrator       = "orchestrator"
	ModuleDocs               = "docs"
	// ModuleUnclassified is the explicit deterministic bucket for rows that
	// carry no execution_owner and no explicit module. It is never a silent
	// mis-bucket — a later planner pass or C5 batch resolves these by
	// setting an explicit module.
	ModuleUnclassified = "unclassified"
)

// moduleForOwner maps the coarse execution_owner enum to the agreed module
// taxonomy. An empty string means "not derivable from owner alone".
func moduleForOwner(owner ExecutionOwner) string {
	switch owner {
	case ExecutionOwnerDocs:
		return ModuleDocs
	case ExecutionOwnerGateway:
		return ModuleGatewayChannels
	case ExecutionOwnerMemory, ExecutionOwnerSkills, ExecutionOwnerGoncho:
		return ModuleMemorySessions
	case ExecutionOwnerProvider:
		return ModuleProvidersAuth
	case ExecutionOwnerTools:
		return ModuleTools
	case ExecutionOwnerOrchestrator:
		return ModuleOrchestrator
	case ExecutionOwnerTui:
		return ModuleTUI
	default:
		return ""
	}
}

// Module returns the deterministic module bucket for a row: an explicit
// it.Module always wins; otherwise it derives from ExecutionOwner via the
// agreed taxonomy; with neither, it is the explicit ModuleUnclassified
// bucket. phaseID/subphaseID are accepted for a future finer fallback and
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
