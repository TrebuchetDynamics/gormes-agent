package cli

import (
	"strings"
	"testing"
)

func TestOnboardingHintFlagsMatchHermes(t *testing.T) {
	if BusyInputPromptFlag != "busy_input_prompt" {
		t.Fatalf("BusyInputPromptFlag = %q, want busy_input_prompt", BusyInputPromptFlag)
	}
	if ToolProgressPromptFlag != "tool_progress_prompt" {
		t.Fatalf("ToolProgressPromptFlag = %q, want tool_progress_prompt", ToolProgressPromptFlag)
	}
	if OpenClawResidueCleanupFlag != "openclaw_residue_cleanup" {
		t.Fatalf("OpenClawResidueCleanupFlag = %q, want openclaw_residue_cleanup", OpenClawResidueCleanupFlag)
	}
}

func TestBusyInputHintCLIByMode(t *testing.T) {
	tests := []struct {
		mode string
		want []string
	}{
		{mode: "interrupt", want: []string{"interrupted", "/busy queue", "/busy steer", "only shows once"}},
		{mode: "queue", want: []string{"queued", "/busy interrupt", "/busy steer", "only shows once"}},
		{mode: "steer", want: []string{"steered", "/busy interrupt", "/busy queue", "only shows once"}},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := BusyInputHint("cli", tt.mode)
			assertHintContains(t, got, tt.want...)
			assertHintOmits(t, got, "Hermes", "onboarding.seen", "`")
		})
	}
}

func TestBusyInputHintGatewayByMode(t *testing.T) {
	tests := []struct {
		mode string
		want []string
	}{
		{mode: "interrupt", want: []string{"interrupted", "/busy queue", "/busy steer", "/busy status", "only shows once"}},
		{mode: "queue", want: []string{"queued", "/busy interrupt", "/busy status", "only shows once"}},
		{mode: "steer", want: []string{"steered", "/busy interrupt", "/busy queue", "/busy status", "only shows once"}},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := BusyInputHint("gateway", tt.mode)
			assertHintContains(t, got, tt.want...)
			assertHintOmits(t, got, "Hermes", "onboarding.seen", "busy_input_prompt")
		})
	}
}

func TestToolProgressHintCLIAndGateway(t *testing.T) {
	for _, surface := range []string{"cli", "gateway"} {
		t.Run(surface, func(t *testing.T) {
			got := ToolProgressHint(surface)
			assertHintContains(t, got, "/verbose", "all -> new -> off", "only shows once")
			assertHintOmits(t, got, "Hermes", "display.tool_progress", "config.toml")
		})
	}
}

func TestOnboardingHintsUnknownInputsDegrade(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  string
		want []string
	}{
		{name: "unknown busy mode", got: BusyInputHint("cli", "surprise"), want: []string{"interrupted", "/busy queue"}},
		{name: "unknown surface busy", got: BusyInputHint("unknown", "queue"), want: []string{"queued", "/busy interrupt", "only shows once"}},
		{name: "unknown surface tool", got: ToolProgressHint("unknown"), want: []string{"/verbose", "all -> new -> off", "only shows once"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if strings.TrimSpace(tc.got) == "" {
				t.Fatal("hint is empty")
			}
			assertHintContains(t, tc.got, tc.want...)
		})
	}
}

func assertHintContains(t *testing.T, got string, wants ...string) {
	t.Helper()
	lower := strings.ToLower(got)
	for _, want := range wants {
		if !strings.Contains(lower, strings.ToLower(want)) {
			t.Fatalf("hint missing %q:\n%s", want, got)
		}
	}
}

func assertHintOmits(t *testing.T, got string, forbiddens ...string) {
	t.Helper()
	for _, forbidden := range forbiddens {
		if strings.Contains(got, forbidden) {
			t.Fatalf("hint contains %q:\n%s", forbidden, got)
		}
	}
}
