package gateway

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

func TestCommandRegistryContainsRequiredCommands(t *testing.T) {
	if len(CommandRegistry) == 0 {
		t.Fatal("CommandRegistry is empty")
	}

	required := map[string]bool{
		"help": false,
		"new":  false,
		"stop": false,
	}
	for _, cmd := range CommandRegistry {
		if _, ok := required[cmd.Name]; ok {
			required[cmd.Name] = true
		}
	}
	for name, seen := range required {
		if !seen {
			t.Fatalf("CommandRegistry missing %q", name)
		}
	}
}

func TestResolveCommand(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "help", raw: "/help", want: "help", ok: true},
		{name: "new", raw: "/new", want: "new", ok: true},
		{name: "stop", raw: "/stop", want: "stop", ok: true},
		{name: "telegram alias", raw: "/start", want: "help", ok: true},
		{name: "unknown", raw: "/xyzzy", want: "", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolveCommand(tt.raw)
			if ok != tt.ok {
				t.Fatalf("ResolveCommand(%q) ok = %v, want %v", tt.raw, ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if got.Name != tt.want {
				t.Fatalf("ResolveCommand(%q).Name = %q, want %q", tt.raw, got.Name, tt.want)
			}
		})
	}
}

func TestParseInboundText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantKind EventKind
		wantBody string
	}{
		{name: "help", text: "/help", wantKind: EventStart, wantBody: ""},
		{name: "new", text: "/new", wantKind: EventReset, wantBody: ""},
		{name: "stop", text: "/stop", wantKind: EventCancel, wantBody: ""},
		{name: "steer", text: "/steer keep going", wantKind: EventSteer, wantBody: "/steer keep going"},
		{name: "status", text: "/status", wantKind: EventStatus, wantBody: ""},
		{name: "verbose", text: "/verbose", wantKind: EventVerbose, wantBody: ""},
		{name: "unknown slash", text: "/wat", wantKind: EventUnknown, wantBody: ""},
		{name: "submit", text: "hello there", wantKind: EventSubmit, wantBody: "hello there"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKind, gotBody := ParseInboundText(tt.text)
			if gotKind != tt.wantKind || gotBody != tt.wantBody {
				t.Fatalf("ParseInboundText(%q) = (%v, %q), want (%v, %q)", tt.text, gotKind, gotBody, tt.wantKind, tt.wantBody)
			}
		})
	}
}

func TestGatewayHelpLinesDerivedFromRegistry(t *testing.T) {
	lines := GatewayHelpLines()
	if len(lines) == 0 {
		t.Fatal("GatewayHelpLines returned no lines")
	}

	joined := strings.Join(lines, "\n")
	for _, want := range []string{"/help", "/new", "/stop"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("GatewayHelpLines missing %q in %q", want, joined)
		}
	}
}

func TestSlashCommandPolicyParityWithCLIRegistry(t *testing.T) {
	if len(CommandRegistry) == 0 {
		t.Fatal("gateway CommandRegistry empty")
	}
	for _, gw := range CommandRegistry {
		policy, ok := cli.ResolveCommandPolicy(gw.Name)
		if !ok {
			t.Errorf("gateway command %q not present in CLI command registry", gw.Name)
			continue
		}
		var want cli.ActiveTurnPolicy
		switch gw.ActiveTurnPolicy {
		case CommandActiveTurnPolicyImmediate:
			want = cli.ActiveTurnPolicyBypass
		case CommandActiveTurnPolicyDrain:
			if policy.ActiveTurnPolicy == cli.ActiveTurnPolicyQueue {
				continue
			}
			want = cli.ActiveTurnPolicyBypass
		case CommandActiveTurnPolicyReject:
			want = cli.ActiveTurnPolicyBusyReject
		case CommandActiveTurnPolicyUnavailable:
			if policy.Ported && policy.Surface == cli.CommandSurfaceCLI {
				continue
			}
			want = cli.ActiveTurnPolicyUnavailable
		default:
			t.Errorf("gateway command %q has unmapped policy %q", gw.Name, gw.ActiveTurnPolicy)
			continue
		}
		if policy.ActiveTurnPolicy != want {
			t.Errorf("gateway %q policy %q maps to CLI %q, want %q",
				gw.Name, gw.ActiveTurnPolicy, policy.ActiveTurnPolicy, want)
		}
	}
}

func TestGatewayRegistryRecognizesSharedAndGatewayCLICommands(t *testing.T) {
	for _, policy := range cli.CommandRegistry {
		if policy.ActiveTurnPolicy == cli.ActiveTurnPolicyQueue {
			continue
		}
		cmd, ok := ResolveCommand(policy.Name)
		if !ok {
			t.Errorf("gateway registry does not recognize CLI %s command %q", policy.Surface, policy.Name)
			continue
		}
		if !policy.Ported && cmd.ActiveTurnPolicy != CommandActiveTurnPolicyUnavailable {
			t.Errorf("unported CLI command %q gateway policy = %q, want unavailable", policy.Name, cmd.ActiveTurnPolicy)
		}
		for _, alias := range policy.Aliases {
			if _, ok := ResolveCommand(alias); !ok {
				t.Errorf("gateway registry does not recognize alias %q for %q", alias, policy.Name)
			}
		}
	}
}

func TestSlashCommandBusyVerdictRejectsKnownMutators(t *testing.T) {
	v := cli.EvaluateActiveTurnVerdict("/new", true)
	if !v.Known {
		t.Fatal("verdict.Known = false for /new, want true")
	}
	if v.Allowed {
		t.Errorf("verdict.Allowed = true for /new during active turn, want busy reject")
	}
	if v.Policy != cli.ActiveTurnPolicyBusyReject {
		t.Errorf("verdict.Policy = %q, want busy_reject", v.Policy)
	}
}

func TestSlashCommandUnknownDoesNotEnterModelPrompt(t *testing.T) {
	if cli.SlashLeaksToModelPrompt("/no-such-command") {
		t.Error("unknown slash command must not leak into model prompt")
	}
	if cli.SlashLeaksToModelPrompt("/help") {
		t.Error("recognized slash command must not leak into model prompt")
	}
	if !cli.SlashLeaksToModelPrompt("hello world") {
		t.Error("plain text must reach the model prompt as submit text")
	}
	kind, body := ParseInboundText("/no-such-command")
	if kind != EventUnknown {
		t.Errorf("ParseInboundText(unknown slash) kind = %v, want EventUnknown", kind)
	}
	if body != "" {
		t.Errorf("ParseInboundText(unknown slash) body = %q, want empty", body)
	}
}

func TestTelegramBotCommandsExposeHermesGatewayMenu(t *testing.T) {
	commands := TelegramBotCommands()
	seen := make(map[string]string, len(commands))
	for _, cmd := range commands {
		seen[cmd.Name] = cmd.Description
	}
	for _, want := range []string{
		"new",
		"retry",
		"title",
		"stop",
		"steer",
		"status",
		"usage",
		"platforms",
		"profile",
		"sessions",
		"skills",
		"verbose",
	} {
		if _, ok := seen[want]; !ok {
			t.Fatalf("TelegramBotCommands missing Hermes gateway command %q; got %#v", want, commands)
		}
	}
}

func TestPlatformExposureDeterministic(t *testing.T) {
	tg1 := TelegramBotCommands()
	tg2 := TelegramBotCommands()
	if !reflect.DeepEqual(tg1, tg2) {
		t.Fatalf("TelegramBotCommands unstable:\n%#v\n%#v", tg1, tg2)
	}
	if len(tg1) == 0 {
		t.Fatal("TelegramBotCommands returned no commands")
	}

	slack1 := SlackSubcommandMap()
	slack2 := SlackSubcommandMap()
	if !reflect.DeepEqual(slack1, slack2) {
		t.Fatalf("SlackSubcommandMap unstable:\n%#v\n%#v", slack1, slack2)
	}
	for _, want := range []string{"help", "new", "stop"} {
		if _, ok := slack1[want]; !ok {
			t.Fatalf("SlackSubcommandMap missing %q", want)
		}
	}
}
