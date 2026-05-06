package tuigateway

import (
	"strings"
	"testing"
)

func TestTUIConfigHealthFlagsNullSections(t *testing.T) {
	t.Parallel()

	if got := ProbeConfigHealth(map[string]any{"agent": map[string]any{"x": 1}}); got.HasWarnings() {
		t.Fatalf("ProbeConfigHealth(populated agent) warnings = %#v; want none", got.Warnings)
	}
	if got := ProbeConfigHealth(map[string]any{}); got.HasWarnings() {
		t.Fatalf("ProbeConfigHealth(empty config) warnings = %#v; want none", got.Warnings)
	}

	got := ProbeConfigHealth(map[string]any{
		"agent":   nil,
		"display": nil,
		"model":   map[string]any{},
	})
	if !got.HasCode("tui_config_health_warning") {
		t.Fatalf("ProbeConfigHealth(null sections) codes = %#v; want tui_config_health_warning", got.Warnings)
	}
	msg := got.Message()
	for _, want := range []string{"`agent`", "`display`"} {
		if !strings.Contains(msg, want) {
			t.Errorf("warning message = %q; want %s", msg, want)
		}
	}
	if strings.Contains(msg, "`model`") {
		t.Errorf("warning message = %q; did not expect populated model section", msg)
	}
}

func TestTUIConfigHealthFlagsNullPersonalitiesWithActivePersonality(t *testing.T) {
	t.Parallel()

	got := ProbeConfigHealth(map[string]any{
		"agent":   map[string]any{"personalities": nil},
		"display": map[string]any{"personality": "kawaii"},
		"model":   map[string]any{},
	})
	if !got.HasCode("tui_personality_skipped") {
		t.Fatalf("ProbeConfigHealth(null personalities) warnings = %#v; want tui_personality_skipped", got.Warnings)
	}
	msg := got.Message()
	for _, want := range []string{"display.personality", "agent.personalities", "skipped"} {
		if !strings.Contains(msg, want) {
			t.Errorf("warning message = %q; want %q", msg, want)
		}
	}
}

func TestTUIConfigHealthIgnoresDefaultPersonalityAliases(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "  ", "default", "none", "neutral", " DEFAULT "} {
		raw := raw
		t.Run(strings.TrimSpace(raw)+"_alias", func(t *testing.T) {
			t.Parallel()
			got := ProbeConfigHealth(map[string]any{
				"agent":   map[string]any{"personalities": nil},
				"display": map[string]any{"personality": raw},
			})
			if got.HasCode("tui_personality_skipped") {
				t.Fatalf("ProbeConfigHealth(%q) warnings = %#v; want no personality warning", raw, got.Warnings)
			}
		})
	}
}

func TestTUISessionOptionsTolerateNullSections(t *testing.T) {
	t.Parallel()

	got := ResolveSessionOptions(map[string]any{
		"agent":   nil,
		"display": nil,
		"model":   map[string]any{"default": "glm-5"},
	})
	if got.HasEphemeralSystemPrompt {
		t.Fatalf("HasEphemeralSystemPrompt = true; want false for null agent section")
	}
	if got.EphemeralSystemPrompt != "" {
		t.Fatalf("EphemeralSystemPrompt = %q; want empty", got.EphemeralSystemPrompt)
	}
	if got.DisplayPersonality != "" {
		t.Fatalf("DisplayPersonality = %q; want empty for null display section", got.DisplayPersonality)
	}
	if got.PersonalitySkipped {
		t.Fatalf("PersonalitySkipped = true; want false with no active display personality")
	}
}

func TestTUISessionOptionsDoNotActivateDisplayPersonalityWithoutPrompt(t *testing.T) {
	t.Parallel()

	got := ResolveSessionOptions(map[string]any{
		"agent": map[string]any{
			"system_prompt": "",
			"personalities": map[string]any{"kawaii": "sparkle system prompt"},
		},
		"display": map[string]any{"personality": "kawaii"},
	})
	if got.HasEphemeralSystemPrompt {
		t.Fatalf("HasEphemeralSystemPrompt = true; display.personality must not activate without agent.system_prompt")
	}
	if got.EphemeralSystemPrompt != "" {
		t.Fatalf("EphemeralSystemPrompt = %q; want empty", got.EphemeralSystemPrompt)
	}
	if got.DisplayPersonality != "kawaii" {
		t.Fatalf("DisplayPersonality = %q; want kawaii", got.DisplayPersonality)
	}
	if !got.PersonalitySkipped {
		t.Fatalf("PersonalitySkipped = false; want true until a concrete system prompt is supplied")
	}
	if !containsString(got.Evidence, "tui_personality_skipped") {
		t.Fatalf("Evidence = %#v; want tui_personality_skipped", got.Evidence)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
