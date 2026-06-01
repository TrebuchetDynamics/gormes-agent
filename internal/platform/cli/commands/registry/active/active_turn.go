package active

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands/registry/catalog"

// ActiveTurnVerdict is the result of evaluating a slash command against the
// current active-turn state. Allowed reports whether the command should be
// dispatched immediately; Evidence is the operator-facing reason when the
// command is not allowed (busy, queue, or unavailable).
type ActiveTurnVerdict struct {
	Name     string
	Policy   catalog.ActiveTurnPolicy
	Known    bool
	Allowed  bool
	Evidence string
}

// EvaluateActiveTurnVerdict returns the dispatch decision for a slash command
// in the current active-turn state. Unknown commands evaluate as
// catalog.ActiveTurnPolicyUnavailable with explicit evidence so the caller never lets
// the original slash text reach the kernel as ordinary prompt content.
func EvaluateActiveTurnVerdict(name string, busy bool) ActiveTurnVerdict {
	cmd, ok := catalog.ResolveCommandPolicy(name)
	if !ok {
		return ActiveTurnVerdict{
			Name:     catalog.NormalizeCommandToken(name),
			Policy:   catalog.ActiveTurnPolicyUnavailable,
			Known:    false,
			Allowed:  false,
			Evidence: "unknown command — no slash command by that name is available",
		}
	}
	switch cmd.ActiveTurnPolicy {
	case catalog.ActiveTurnPolicyUnavailable:
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   catalog.ActiveTurnPolicyUnavailable,
			Known:    true,
			Allowed:  false,
			Evidence: "/" + cmd.Name + " is recognized but unavailable in this build",
		}
	case catalog.ActiveTurnPolicyBypass:
		return ActiveTurnVerdict{
			Name:    cmd.Name,
			Policy:  catalog.ActiveTurnPolicyBypass,
			Known:   true,
			Allowed: true,
		}
	case catalog.ActiveTurnPolicyQueue:
		if !busy {
			return ActiveTurnVerdict{
				Name:    cmd.Name,
				Policy:  catalog.ActiveTurnPolicyQueue,
				Known:   true,
				Allowed: true,
			}
		}
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   catalog.ActiveTurnPolicyQueue,
			Known:    true,
			Allowed:  false,
			Evidence: "/" + cmd.Name + " was queued — it will run after the current turn finishes",
		}
	case catalog.ActiveTurnPolicyBusyReject:
		if !busy {
			return ActiveTurnVerdict{
				Name:    cmd.Name,
				Policy:  catalog.ActiveTurnPolicyBusyReject,
				Known:   true,
				Allowed: true,
			}
		}
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   catalog.ActiveTurnPolicyBusyReject,
			Known:    true,
			Allowed:  false,
			Evidence: "Gormes is busy — finish the current turn or send /stop before /" + cmd.Name,
		}
	}
	// Unreachable for valid registry entries; treat as unavailable defensively.
	return ActiveTurnVerdict{
		Name:     cmd.Name,
		Policy:   catalog.ActiveTurnPolicyUnavailable,
		Known:    true,
		Allowed:  false,
		Evidence: "/" + cmd.Name + " has no defined active-turn policy",
	}
}
