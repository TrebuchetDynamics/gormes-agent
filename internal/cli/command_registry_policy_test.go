package cli

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// validActiveTurnPolicies is the closed set of policies the CLI registry must
// declare for every recognized command. Anything outside this set is a regression
// because the contract is "every recognized command has a stable active-turn
// policy: bypass, queue, busy-reject, or unavailable".
var validActiveTurnPolicies = map[ActiveTurnPolicy]struct{}{
	ActiveTurnPolicyBypass:      {},
	ActiveTurnPolicyQueue:       {},
	ActiveTurnPolicyBusyReject:  {},
	ActiveTurnPolicyUnavailable: {},
}

func TestCommandRegistryPolicyEveryCommandHasNonEmptyPolicy(t *testing.T) {
	if len(CommandRegistry) == 0 {
		t.Fatal("CommandRegistry is empty")
	}
	seen := make(map[string]struct{}, len(CommandRegistry))
	for _, cmd := range CommandRegistry {
		if cmd.Name == "" {
			t.Errorf("command has empty name: %+v", cmd)
		}
		if _, dup := seen[cmd.Name]; dup {
			t.Errorf("duplicate command %q in CommandRegistry", cmd.Name)
		}
		seen[cmd.Name] = struct{}{}
		if _, ok := validActiveTurnPolicies[cmd.ActiveTurnPolicy]; !ok {
			t.Errorf("command %q has invalid active-turn policy %q", cmd.Name, cmd.ActiveTurnPolicy)
		}
		if cmd.ActiveTurnPolicy == ActiveTurnPolicyUnavailable && cmd.Ported {
			t.Errorf("command %q is marked Ported but policy is unavailable", cmd.Name)
		}
		if cmd.ActiveTurnPolicy != ActiveTurnPolicyUnavailable && !cmd.Ported {
			t.Errorf("command %q is not Ported but policy %q is not unavailable", cmd.Name, cmd.ActiveTurnPolicy)
		}
	}
}

func TestCommandRegistryPolicyKnownAssignments(t *testing.T) {
	cases := map[string]ActiveTurnPolicy{
		"help":     ActiveTurnPolicyBypass,
		"stop":     ActiveTurnPolicyBypass,
		"new":      ActiveTurnPolicyBusyReject,
		"restart":  ActiveTurnPolicyBypass,
		"reasoning": ActiveTurnPolicyQueue,
	}
	for name, want := range cases {
		got, ok := ResolveCommandPolicy(name)
		if !ok {
			t.Errorf("ResolveCommandPolicy(%q) not found", name)
			continue
		}
		if got.ActiveTurnPolicy != want {
			t.Errorf("ResolveCommandPolicy(%q).ActiveTurnPolicy = %q, want %q", name, got.ActiveTurnPolicy, want)
		}
	}
}

func TestCommandRegistryPolicyResolveSlash(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{raw: "/help", want: "help", ok: true},
		{raw: "help", want: "help", ok: true},
		{raw: "  /HELP  ", want: "help", ok: true},
		{raw: "/start", want: "help", ok: true},
		{raw: "/new", want: "new", ok: true},
		{raw: "/reset", want: "new", ok: true},
		{raw: "/no-such-command-xyzzy", want: "", ok: false},
		{raw: "", want: "", ok: false},
	}
	for _, tc := range cases {
		got, ok := ResolveCommandPolicy(tc.raw)
		if ok != tc.ok {
			t.Errorf("ResolveCommandPolicy(%q) ok = %v, want %v", tc.raw, ok, tc.ok)
			continue
		}
		if !tc.ok {
			continue
		}
		if got.Name != tc.want {
			t.Errorf("ResolveCommandPolicy(%q).Name = %q, want %q", tc.raw, got.Name, tc.want)
		}
	}
}

func TestCommandRegistryPolicyBusyVerdictForUnknownIsUnavailable(t *testing.T) {
	v := EvaluateActiveTurnVerdict("/no-such-command-xyzzy", true)
	if v.Known {
		t.Fatalf("verdict.Known = true for unknown command, want false (verdict=%+v)", v)
	}
	if v.Policy != ActiveTurnPolicyUnavailable {
		t.Errorf("verdict.Policy = %q, want %q", v.Policy, ActiveTurnPolicyUnavailable)
	}
	if v.Allowed {
		t.Errorf("verdict.Allowed = true for unknown command, want false")
	}
	if v.Evidence == "" {
		t.Error("verdict.Evidence is empty for unknown command, want explanation")
	}
}

func TestCommandRegistryPolicyBusyVerdictForBypassDuringTurnAllows(t *testing.T) {
	v := EvaluateActiveTurnVerdict("/help", true)
	if !v.Known {
		t.Fatal("verdict.Known = false for /help, want true")
	}
	if v.Policy != ActiveTurnPolicyBypass {
		t.Errorf("verdict.Policy = %q, want %q", v.Policy, ActiveTurnPolicyBypass)
	}
	if !v.Allowed {
		t.Errorf("verdict.Allowed = false for bypass command during active turn, want true")
	}
}

func TestCommandRegistryPolicyBusyVerdictForBusyRejectDuringTurn(t *testing.T) {
	v := EvaluateActiveTurnVerdict("/new", true)
	if !v.Known {
		t.Fatal("verdict.Known = false for /new, want true")
	}
	if v.Policy != ActiveTurnPolicyBusyReject {
		t.Errorf("verdict.Policy = %q, want %q", v.Policy, ActiveTurnPolicyBusyReject)
	}
	if v.Allowed {
		t.Errorf("verdict.Allowed = true for busy_reject command during active turn, want false")
	}
	if !strings.Contains(strings.ToLower(v.Evidence), "busy") {
		t.Errorf("verdict.Evidence = %q, want to mention busy", v.Evidence)
	}
}

func TestCommandRegistryPolicyBusyVerdictForQueueDuringTurn(t *testing.T) {
	v := EvaluateActiveTurnVerdict("/reasoning", true)
	if !v.Known {
		t.Fatal("verdict.Known = false for /reasoning, want true")
	}
	if v.Policy != ActiveTurnPolicyQueue {
		t.Errorf("verdict.Policy = %q, want %q", v.Policy, ActiveTurnPolicyQueue)
	}
	if v.Allowed {
		t.Errorf("verdict.Allowed = true for queue policy during active turn, want false")
	}
	if !strings.Contains(strings.ToLower(v.Evidence), "queue") {
		t.Errorf("verdict.Evidence = %q, want to mention queue", v.Evidence)
	}
}

func TestCommandRegistryPolicyBusyVerdictForUnavailableSurfacesEvidence(t *testing.T) {
	// Find any command in the registry that is intentionally unavailable.
	var name string
	for _, cmd := range CommandRegistry {
		if cmd.ActiveTurnPolicy == ActiveTurnPolicyUnavailable {
			name = cmd.Name
			break
		}
	}
	if name == "" {
		t.Skip("no unavailable command in registry")
	}
	idle := EvaluateActiveTurnVerdict("/"+name, false)
	busy := EvaluateActiveTurnVerdict("/"+name, true)
	for _, v := range []ActiveTurnVerdict{idle, busy} {
		if !v.Known {
			t.Errorf("verdict.Known = false for known unavailable command %q", name)
		}
		if v.Policy != ActiveTurnPolicyUnavailable {
			t.Errorf("verdict.Policy = %q, want %q", v.Policy, ActiveTurnPolicyUnavailable)
		}
		if v.Allowed {
			t.Errorf("verdict.Allowed = true for unavailable command %q, want false", name)
		}
		if !strings.Contains(strings.ToLower(v.Evidence), "unavailable") {
			t.Errorf("verdict.Evidence for %q = %q, want to mention unavailable", name, v.Evidence)
		}
	}
}

func TestCommandRegistryPolicyBusyVerdictWhenIdleAllowsKnownCommands(t *testing.T) {
	for _, cmd := range CommandRegistry {
		if cmd.ActiveTurnPolicy == ActiveTurnPolicyUnavailable {
			continue
		}
		v := EvaluateActiveTurnVerdict("/"+cmd.Name, false)
		if !v.Known {
			t.Errorf("verdict.Known = false for %q (idle)", cmd.Name)
			continue
		}
		if !v.Allowed {
			t.Errorf("verdict.Allowed = false for %q during idle (policy=%q)", cmd.Name, v.Policy)
		}
	}
}

func TestSlashCommandTextLeaksToModelPromptOnlyForPlainText(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"hello there", true},
		{"  hello   ", true},
		{"", false},
		{"   ", false},
		{"/help", false},
		{"/no-such-command-xyzzy", false},
		{"/HELP arg1", false},
	}
	for _, tc := range cases {
		got := SlashLeaksToModelPrompt(tc.text)
		if got != tc.want {
			t.Errorf("SlashLeaksToModelPrompt(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

func TestCommandRegistryPolicyGatewayParity(t *testing.T) {
	// Every command in the gateway registry must also exist in the CLI
	// registry with a compatible active-turn policy. The CLI registry is the
	// single source of truth for policy; the gateway-side registry exposes a
	// subset shaped for inbound platform parsing.
	for _, gw := range gateway.CommandRegistry {
		cli, ok := ResolveCommandPolicy(gw.Name)
		if !ok {
			t.Errorf("gateway command %q not found in CLI CommandRegistry", gw.Name)
			continue
		}
		want := mapGatewayPolicyToCLI(gw.ActiveTurnPolicy)
		if want == "" {
			t.Errorf("gateway command %q has unmapped policy %q", gw.Name, gw.ActiveTurnPolicy)
			continue
		}
		if cli.ActiveTurnPolicy != want {
			t.Errorf("gateway %q policy %q, CLI %q (want CLI=%q)", gw.Name, gw.ActiveTurnPolicy, cli.ActiveTurnPolicy, want)
		}
		// Aliases must also round-trip.
		for _, alias := range gw.Aliases {
			if got, ok := ResolveCommandPolicy(alias); !ok || got.Name != gw.Name {
				t.Errorf("gateway alias %q for %q does not resolve to CLI %q", alias, gw.Name, gw.Name)
			}
		}
	}
}

func mapGatewayPolicyToCLI(p gateway.CommandActiveTurnPolicy) ActiveTurnPolicy {
	switch p {
	case gateway.CommandActiveTurnPolicyImmediate, gateway.CommandActiveTurnPolicyDrain:
		return ActiveTurnPolicyBypass
	case gateway.CommandActiveTurnPolicyReject:
		return ActiveTurnPolicyBusyReject
	}
	return ""
}
