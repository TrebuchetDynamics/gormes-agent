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

	kind, body = ParseInboundText("／queue keep going")
	if kind != EventQueue || body != "／queue keep going" {
		t.Fatalf("ParseInboundText(fullwidth /queue ...) = (%v, %q), want EventQueue with raw body", kind, body)
	}

	kind, body = ParseInboundText("/undo")
	if kind != EventSubmit || body != "/undo" {
		t.Fatalf("ParseInboundText(/undo) = (%v, %q), want unavailable command preserved as submit", kind, body)
	}

	setHome, ok := ResolveCommand("/set_home")
	if !ok || setHome.Name != "sethome" {
		t.Fatalf("ResolveCommand(/set_home) = (%+v, %v), want sethome underscore alias for platform-safe command text", setHome, ok)
	}
}

func TestDoubleSlashDoesNotResolveAsCommand(t *testing.T) {
	if cmd, ok := ResolveCommand("//title"); ok {
		t.Fatalf("ResolveCommand(//title) = %+v, true; want not resolved", cmd)
	}
	kind, body := ParseInboundText("//title Friendly Greeting")
	if kind == EventTitle || body == "//title Friendly Greeting" && kind != EventUnknown {
		t.Fatalf("ParseInboundText(//title) = (%v, %q), want no title command dispatch", kind, body)
	}
}

func TestUnknownSlashGuidanceRedactsSecretLikeCommandToken(t *testing.T) {
	guidance := UnknownSlashCommandGuidance("/api_key=plain-secret-token")
	for _, forbidden := range []string{"plain-secret-token", "api_key"} {
		if strings.Contains(guidance, forbidden) {
			t.Fatalf("guidance leaked secret-like command token %q: %s", forbidden, guidance)
		}
	}
	if !strings.Contains(guidance, "[redacted]") {
		t.Fatalf("guidance missing redaction marker: %s", guidance)
	}
}

func TestUnknownSlashGuidanceSanitizesCommandToken(t *testing.T) {
	guidance := UnknownSlashCommandGuidance("bad`name")
	if strings.Contains(guidance, "`/bad`name`") || strings.Count(guidance, "`") != 2 {
		t.Fatalf("guidance has unsafe backtick command rendering: %s", guidance)
	}
	if !strings.Contains(guidance, "/bad'name") {
		t.Fatalf("guidance = %s, want sanitized command token", guidance)
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
	if got := NormalizeSlackCommandName("/Planner.Pro_[SAFE]! now"); got != "planner-pro-safe-now" {
		t.Fatalf("NormalizeSlackCommandName = %q, want planner-pro-safe-now", got)
	}
	if got := NormalizeSlackCommandName("!!!"); got != "" {
		t.Fatalf("NormalizeSlackCommandName punctuation = %q, want empty", got)
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

	dynamicTelegram := TelegramBotCommandsWith([]PlatformCommand{{Name: "set-home", Description: "collides with built-in alias"}})
	if platformCommandsContainName(dynamicTelegram, "set_home") {
		t.Fatalf("TelegramBotCommandsWith exposed dynamic command colliding with built-in alias: %#v", dynamicTelegram)
	}

	dynamicTelegram = TelegramBotCommandsWith([]PlatformCommand{{Name: "dynamic-skill", Description: "first line\nsecond\u009b line"}})
	if desc, ok := platformCommandDescription(dynamicTelegram, "dynamic_skill"); !ok || desc != "first line second line" {
		t.Fatalf("TelegramBotCommandsWith dynamic description = %q ok=%v, want sanitized single line", desc, ok)
	}

	dynamicSlack := SlackSubcommandMapWith([]PlatformCommand{
		{Name: "/Planner.Pro_[SAFE]! now", Description: "dynamic skill"},
		{Name: "!!!", Description: "empty after normalization"},
	})
	if got := dynamicSlack["planner-pro-safe-now"]; got != "/planner-pro-safe-now" {
		t.Fatalf("SlackSubcommandMapWith sanitized dynamic command = %q, want /planner-pro-safe-now", got)
	}
	if _, ok := dynamicSlack["!!!"]; ok {
		t.Fatalf("SlackSubcommandMapWith exposed punctuation-only command: %#v", dynamicSlack)
	}
}

func platformCommandsContainName(commands []PlatformCommand, name string) bool {
	_, ok := platformCommandDescription(commands, name)
	return ok
}

func platformCommandDescription(commands []PlatformCommand, name string) (string, bool) {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd.Description, true
		}
	}
	return "", false
}
