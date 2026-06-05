package commandregistry

import (
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
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
		{name: "goal", raw: "/goal status", want: "goal", ok: true},
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

func TestGatewayCommandTelegramBotMentionSuffix(t *testing.T) {
	got, ok := ResolveCommand("/status@GormesBot")
	if !ok || got.Kind != EventStatus {
		t.Fatalf("ResolveCommand(/status@GormesBot) = (%+v, %v), want status command", got, ok)
	}

	kind, body := ParseInboundText("/tts@GormesBot on")
	if kind != EventTTS || body != "/tts@GormesBot on" {
		t.Fatalf("ParseInboundText(/tts@GormesBot on) = (%v, %q), want EventTTS with raw body", kind, body)
	}
}

func TestGatewayCommandAliasFidelity_CanonicalHooksKeepRawCommand(t *testing.T) {
	got := ResolveGatewayCommandDispatch("/provider openrouter --global")
	if !got.Known {
		t.Fatalf("ResolveGatewayCommandDispatch(/provider ...) Known = false")
	}
	if got.RawCommand != "provider" || got.RawArgs != "openrouter --global" {
		t.Fatalf("raw command/args = %q/%q, want provider/openrouter --global (got=%+v)", got.RawCommand, got.RawArgs, got)
	}
	if got.Canonical != "model" || got.Kind != EventModel {
		t.Fatalf("canonical dispatch = %q/%v, want model/%v (got=%+v)", got.Canonical, got.Kind, EventModel, got)
	}
	if !got.Alias {
		t.Fatalf("Alias = false, want true for /provider -> /model")
	}

	exact := ResolveGatewayCommandDispatch("/status now")
	if !exact.Known || exact.RawCommand != "status" || exact.Canonical != "status" || exact.RawArgs != "now" {
		t.Fatalf("exact /status dispatch = %+v, want canonical status with raw args", exact)
	}
}

func TestGatewayCommandAliasFidelity_UnknownSlashGuidance(t *testing.T) {
	got := ResolveGatewayCommandDispatch("/no-such-command-xyzzy")
	if got.Known {
		t.Fatalf("ResolveGatewayCommandDispatch(unknown).Known = true: %+v", got)
	}
	if got.RawCommand != "no-such-command-xyzzy" {
		t.Fatalf("RawCommand = %q, want no-such-command-xyzzy", got.RawCommand)
	}
	kind, body := ParseInboundText("/no-such-command-xyzzy")
	if kind != EventUnknown || body != "" {
		t.Fatalf("ParseInboundText(unknown) = (%v, %q), want (EventUnknown, empty)", kind, body)
	}
	guidance := UnknownSlashCommandGuidance(got.RawCommand)
	for _, want := range []string{"unknown command", "/no-such-command-xyzzy", "/commands", "resend without the leading slash"} {
		if !strings.Contains(guidance, want) {
			t.Fatalf("guidance missing %q: %s", want, guidance)
		}
	}
	if strings.Contains(guidance, "submit") || strings.Contains(guidance, "agent") {
		t.Fatalf("guidance should not imply provider submission: %s", guidance)
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
		{name: "queue", text: "/queue keep going", wantKind: EventQueue, wantBody: "/queue keep going"},
		{name: "queue alias", text: "/q keep going", wantKind: EventQueue, wantBody: "/q keep going"},
		{name: "busy", text: "/busy queue", wantKind: EventBusy, wantBody: "/busy queue"},
		{name: "tts", text: "/tts speed fast", wantKind: EventTTS, wantBody: "/tts speed fast"},
		{name: "retry immediate", text: "/retry", wantKind: EventRetry, wantBody: "/retry"},
		{name: "undo unavailable", text: "/undo", wantKind: EventSubmit, wantBody: "/undo"},
		{name: "goal", text: "/goal status", wantKind: EventGoal, wantBody: "/goal status"},
		{name: "status", text: "/status", wantKind: EventStatus, wantBody: ""},
		{name: "personality", text: "/personality pirate", wantKind: EventPersonality, wantBody: "/personality pirate"},
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

func TestParseInboundTextBodyPolicyMatchesSlashDispatch(t *testing.T) {
	for _, cmd := range CommandRegistry {
		raw := "/" + cmd.Name + " sample"
		gotKind, gotBody := ParseInboundText(raw)
		if cmd.ActiveTurnPolicy == CommandActiveTurnPolicyUnavailable {
			if gotKind != EventSubmit || gotBody != raw {
				t.Fatalf("ParseInboundText(%q) = (%v, %q), want unavailable command preserved as submit body", raw, gotKind, gotBody)
			}
			continue
		}
		if gotKind != cmd.Kind {
			t.Fatalf("ParseInboundText(%q) kind = %v, want %v", raw, gotKind, cmd.Kind)
		}
		if SlashCommandKindCarriesBody(cmd.Kind) {
			if gotBody != raw {
				t.Fatalf("ParseInboundText(%q) body = %q, want raw body for %v", raw, gotBody, cmd.Kind)
			}
		} else if gotBody != "" {
			t.Fatalf("ParseInboundText(%q) body = %q, want empty body for %v", raw, gotBody, cmd.Kind)
		}
	}
}

func TestGatewayCommandCuratorRemainsUnavailable(t *testing.T) {
	cmd, ok := ResolveCommand("/curator status")
	if !ok {
		t.Fatal("/curator did not resolve through gateway CommandRegistry")
	}
	if cmd.ActiveTurnPolicy != CommandActiveTurnPolicyUnavailable {
		t.Fatalf("/curator gateway policy = %q, want %q", cmd.ActiveTurnPolicy, CommandActiveTurnPolicyUnavailable)
	}
	kind, body := ParseInboundText("/curator status")
	if kind != EventSubmit || body != "/curator status" {
		t.Fatalf("ParseInboundText(/curator status) = (%v, %q), want unavailable command preserved as submit body", kind, body)
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

func TestNormalizeTelegramDynamicCommandNameKeepsPlatformSafeShape(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "punctuation", raw: "/Planner.Pro_[SAFE]!", want: "planner_pro_safe"},
		{name: "collapse separators", raw: "---bad...skill---", want: "bad_skill"},
		{name: "empty after sanitize", raw: "!!!", want: ""},
		{name: "telegram max length", raw: "skill-abcdefghijklmnopqrstuvwxyz-0123456789", want: "skill_abcdefghijklmnopqrstuvwxyz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTelegramCommandName(tt.raw); got != tt.want {
				t.Fatalf("NormalizeTelegramCommandName(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
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
