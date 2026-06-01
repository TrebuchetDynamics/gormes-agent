package registry

// ActiveTurnVerdict is the result of evaluating a slash command against the
// current active-turn state. Allowed reports whether the command should be
// dispatched immediately; Evidence is the operator-facing reason when the
// command is not allowed (busy, queue, or unavailable).
type ActiveTurnVerdict struct {
	Name     string
	Policy   ActiveTurnPolicy
	Known    bool
	Allowed  bool
	Evidence string
}

// EvaluateActiveTurnVerdict returns the dispatch decision for a slash command
// in the current active-turn state. Unknown commands evaluate as
// ActiveTurnPolicyUnavailable with explicit evidence so the caller never lets
// the original slash text reach the kernel as ordinary prompt content.
func EvaluateActiveTurnVerdict(name string, busy bool) ActiveTurnVerdict {
	cmd, ok := ResolveCommandPolicy(name)
	if !ok {
		return ActiveTurnVerdict{
			Name:     NormalizeCommandToken(name),
			Policy:   ActiveTurnPolicyUnavailable,
			Known:    false,
			Allowed:  false,
			Evidence: "unknown command — no slash command by that name is available",
		}
	}
	switch cmd.ActiveTurnPolicy {
	case ActiveTurnPolicyUnavailable:
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   ActiveTurnPolicyUnavailable,
			Known:    true,
			Allowed:  false,
			Evidence: "/" + cmd.Name + " is recognized but unavailable in this build",
		}
	case ActiveTurnPolicyBypass:
		return ActiveTurnVerdict{
			Name:    cmd.Name,
			Policy:  ActiveTurnPolicyBypass,
			Known:   true,
			Allowed: true,
		}
	case ActiveTurnPolicyQueue:
		if !busy {
			return ActiveTurnVerdict{
				Name:    cmd.Name,
				Policy:  ActiveTurnPolicyQueue,
				Known:   true,
				Allowed: true,
			}
		}
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   ActiveTurnPolicyQueue,
			Known:    true,
			Allowed:  false,
			Evidence: "/" + cmd.Name + " was queued — it will run after the current turn finishes",
		}
	case ActiveTurnPolicyBusyReject:
		if !busy {
			return ActiveTurnVerdict{
				Name:    cmd.Name,
				Policy:  ActiveTurnPolicyBusyReject,
				Known:   true,
				Allowed: true,
			}
		}
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   ActiveTurnPolicyBusyReject,
			Known:    true,
			Allowed:  false,
			Evidence: "Gormes is busy — finish the current turn or send /stop before /" + cmd.Name,
		}
	}
	// Unreachable for valid registry entries; treat as unavailable defensively.
	return ActiveTurnVerdict{
		Name:     cmd.Name,
		Policy:   ActiveTurnPolicyUnavailable,
		Known:    true,
		Allowed:  false,
		Evidence: "/" + cmd.Name + " has no defined active-turn policy",
	}
}
