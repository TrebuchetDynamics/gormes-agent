package commandregistry

import (
	"reflect"
	"strings"
	"testing"
)

func TestRegistryResolvesAliasesAndParsesBodies(t *testing.T) {
	cmd, ok := ResolveCommand("/provider openrouter --global")
	if !ok || cmd.Name != "model" || cmd.Kind != EventModel {
		t.Fatalf("ResolveCommand(provider) = (%+v, %v), want model", cmd, ok)
	}

	dispatch := ResolveGatewayCommandDispatch("/provider openrouter --global")
	if !dispatch.Known || !dispatch.Alias || dispatch.RawCommand != "provider" || dispatch.RawArgs != "openrouter --global" || dispatch.Canonical != "model" {
		t.Fatalf("ResolveGatewayCommandDispatch(provider) = %+v", dispatch)
	}

	kind, body := ParseInboundText("/queue keep going")
	if kind != EventQueue || body != "/queue keep going" {
		t.Fatalf("ParseInboundText(/queue ...) = (%v, %q), want EventQueue with raw body", kind, body)
	}

	kind, body = ParseInboundText("/undo")
	if kind != EventSubmit || body != "/undo" {
		t.Fatalf("ParseInboundText(/undo) = (%v, %q), want unavailable command preserved as submit", kind, body)
	}
}

func TestUnknownSlashGuidanceDoesNotSubmitToModel(t *testing.T) {
	kind, body := ParseInboundText("/no-such-command-xyzzy")
	if kind != EventUnknown || body != "" {
		t.Fatalf("ParseInboundText(unknown) = (%v, %q), want EventUnknown empty body", kind, body)
	}
	guidance := UnknownSlashCommandGuidance("/no-such-command-xyzzy")
	for _, want := range []string{"unknown command", "/no-such-command-xyzzy", "/commands", "resend without the leading slash"} {
		if !strings.Contains(guidance, want) {
			t.Fatalf("guidance missing %q: %s", want, guidance)
		}
	}
	if strings.Contains(guidance, "submit") || strings.Contains(guidance, "agent") {
		t.Fatalf("guidance should not imply provider submission: %s", guidance)
	}
}

func TestPlatformExposureDeterministicAndSafe(t *testing.T) {
	commands1 := TelegramBotCommands()
	commands2 := TelegramBotCommands()
	if !reflect.DeepEqual(commands1, commands2) {
		t.Fatalf("TelegramBotCommands unstable:\n%#v\n%#v", commands1, commands2)
	}
	seen := map[string]bool{}
	for _, cmd := range commands1 {
		seen[cmd.Name] = true
	}
	for _, want := range []string{"new", "retry", "title", "stop", "steer", "status", "usage", "platforms", "profile", "sessions", "skills", "verbose"} {
		if !seen[want] {
			t.Fatalf("TelegramBotCommands missing %q; got %#v", want, commands1)
		}
	}

	if got := NormalizeTelegramCommandName("/Planner.Pro_[SAFE]!"); got != "planner_pro_safe" {
		t.Fatalf("NormalizeTelegramCommandName = %q, want planner_pro_safe", got)
	}

	slack1 := SlackSubcommandMap()
	slack2 := SlackSubcommandMap()
	if !reflect.DeepEqual(slack1, slack2) {
		t.Fatalf("SlackSubcommandMap unstable")
	}
	for _, want := range []string{"help", "new", "stop"} {
		if _, ok := slack1[want]; !ok {
			t.Fatalf("SlackSubcommandMap missing %q", want)
		}
	}
}
