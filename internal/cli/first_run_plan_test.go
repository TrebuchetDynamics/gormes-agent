package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildFirstRunPlan_FreshInstallRouterOptions(t *testing.T) {
	plan := BuildFirstRunPlan(FirstRunPlanInput{
		Interactive: true,
		Target:      "tui",
		Channels:    DefaultFirstRunChannels(nil),
	})

	if plan.Ready {
		t.Fatal("Ready = true, want false for fresh install")
	}
	if !plan.PromptAllowed {
		t.Fatal("PromptAllowed = false, want true for interactive input")
	}
	if plan.Target != SetupTargetTerminal {
		t.Fatalf("Target = %q, want terminal alias normalization", plan.Target)
	}
	if plan.TargetLabel != "Terminal" {
		t.Fatalf("TargetLabel = %q, want Terminal", plan.TargetLabel)
	}
	if plan.DefaultTarget != SetupTargetTerminal {
		t.Fatalf("DefaultTarget = %q, want terminal", plan.DefaultTarget)
	}
	assertActionIDs(t, plan.Actions, []FirstRunActionID{FirstRunActionQuick, FirstRunActionFull})
	if plan.DefaultAction != FirstRunActionQuick {
		t.Fatalf("DefaultAction = %q, want quick", plan.DefaultAction)
	}
	if plan.NextCommand != "gormes setup --quick --target terminal" {
		t.Fatalf("NextCommand = %q, want quick terminal setup command", plan.NextCommand)
	}
	if len(plan.Targets) != 6 {
		t.Fatalf("len(Targets) = %d, want 6: %#v", len(plan.Targets), plan.Targets)
	}
	wantTargets := []SetupTargetID{
		SetupTargetTerminal,
		SetupTargetTelegram,
		SetupTargetWhatsApp,
		SetupTargetDiscord,
		SetupTargetSlack,
		SetupTargetNavibox,
	}
	for i, want := range wantTargets {
		if plan.Targets[i].ID != want {
			t.Fatalf("target %d = %q, want %q: %#v", i, plan.Targets[i].ID, want, plan.Targets)
		}
	}
	if _, ok := plan.Step(FirstRunStepProvider); !ok {
		t.Fatalf("missing provider step: %#v", plan.MissingSteps)
	}
	if !strings.Contains(plan.Summary, plan.MissingSteps[0].Detail) {
		t.Fatalf("Summary = %q, want first missing detail %q", plan.Summary, plan.MissingSteps[0].Detail)
	}
}

func TestBuildFirstRunPlan_MigrationsAppearOnlyWhenAvailable(t *testing.T) {
	withoutSources := BuildFirstRunPlan(FirstRunPlanInput{})
	assertActionIDs(t, withoutSources.Actions, []FirstRunActionID{FirstRunActionQuick, FirstRunActionFull})

	withSources := BuildFirstRunPlan(FirstRunPlanInput{
		HermesSourcePath:   "/tmp/hermes",
		OpenClawSourcePath: "/tmp/openclaw",
	})
	assertActionIDs(t, withSources.Actions, []FirstRunActionID{
		FirstRunActionQuick,
		FirstRunActionFull,
		FirstRunActionMigrateHermes,
		FirstRunActionMigrateOpenClaw,
	})
	for _, action := range withSources.Actions {
		if !action.Available {
			t.Fatalf("action %q Available = false, want true: %#v", action.ID, action)
		}
		if action.Command == "" {
			t.Fatalf("action %q missing command: %#v", action.ID, action)
		}
	}
}

func TestBuildFirstRunPlan_ReadyTerminalNeedsCoreProviderPieces(t *testing.T) {
	tests := []struct {
		name  string
		input FirstRunPlanInput
		step  FirstRunStepID
	}{
		{
			name: "provider requires endpoint",
			input: FirstRunPlanInput{
				Provider:      "openai",
				Model:         "gpt-4.1",
				APIKeyPresent: true,
			},
			step: FirstRunStepProvider,
		},
		{
			name: "auth missing with provider uses provider command",
			input: FirstRunPlanInput{
				Provider: "anthropic",
				Endpoint: "https://api.anthropic.com",
				Model:    "claude-sonnet-4-5",
			},
			step: FirstRunStepAuth,
		},
		{
			name: "model missing",
			input: FirstRunPlanInput{
				Provider:      "openai",
				Endpoint:      "https://api.openai.com/v1",
				APIKeyPresent: true,
			},
			step: FirstRunStepModel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan := BuildFirstRunPlan(tc.input)
			if plan.Ready {
				t.Fatalf("Ready = true, want false: %#v", plan)
			}
			step, ok := plan.Step(tc.step)
			if !ok {
				t.Fatalf("missing step %q: %#v", tc.step, plan.MissingSteps)
			}
			switch tc.step {
			case FirstRunStepAuth:
				if step.Command != "gormes auth add anthropic" {
					t.Fatalf("auth Command = %q, want provider-specific auth command", step.Command)
				}
			case FirstRunStepModel:
				if step.Command != "gormes setup model" {
					t.Fatalf("model Command = %q, want setup model", step.Command)
				}
			}
		})
	}

	ready := BuildFirstRunPlan(FirstRunPlanInput{
		Provider:      "openai",
		Endpoint:      "https://api.openai.com/v1",
		Model:         "gpt-4.1",
		APIKeyPresent: true,
		Target:        "chat",
	})
	if !ready.Ready {
		t.Fatalf("Ready = false, want true: %#v", ready)
	}
	if ready.Summary != "terminal chat is ready" {
		t.Fatalf("Summary = %q, want terminal ready summary", ready.Summary)
	}
	if ready.NextCommand != "gormes" {
		t.Fatalf("NextCommand = %q, want terminal handoff", ready.NextCommand)
	}
	if len(ready.MissingSteps) != 0 {
		t.Fatalf("MissingSteps = %#v, want none", ready.MissingSteps)
	}
}

func TestBuildFirstRunPlan_ChannelTargetIncludesChannelStep(t *testing.T) {
	plan := BuildFirstRunPlan(FirstRunPlanInput{
		Provider:      "openai",
		Endpoint:      "https://api.openai.com/v1",
		Model:         "gpt-4.1",
		APIKeyPresent: true,
		Target:        SetupTargetWhatsApp,
		Channels: DefaultFirstRunChannels(map[SetupTargetID]ChannelState{
			SetupTargetTelegram: {
				Configured: true,
				Detail:     "telegram token configured",
			},
		}),
	})

	if plan.Ready {
		t.Fatal("Ready = true, want false while selected channel is missing")
	}
	step, ok := plan.Step(FirstRunStepChannel)
	if !ok {
		t.Fatalf("missing channel step: %#v", plan.MissingSteps)
	}
	if step.Command != "gormes whatsapp" {
		t.Fatalf("channel Command = %q, want WhatsApp setup command", step.Command)
	}
	if !strings.Contains(step.Detail, "WhatsApp") {
		t.Fatalf("channel Detail = %q, want selected channel label", step.Detail)
	}
	if plan.NextCommand != "gormes whatsapp" {
		t.Fatalf("NextCommand = %q, want WhatsApp setup", plan.NextCommand)
	}

	ready := BuildFirstRunPlan(FirstRunPlanInput{
		Provider:      "openai",
		Endpoint:      "https://api.openai.com/v1",
		Model:         "gpt-4.1",
		APIKeyPresent: true,
		Target:        SetupTargetSlack,
		Channels: DefaultFirstRunChannels(map[SetupTargetID]ChannelState{
			SetupTargetSlack: {
				Configured:     true,
				Detail:         "slack app configured",
				HandoffCommand: "gormes gateway --channel slack",
			},
		}),
	})
	if !ready.Ready {
		t.Fatalf("Ready = false, want ready Slack channel: %#v", ready)
	}
	if ready.Summary != "Slack is ready" {
		t.Fatalf("Summary = %q, want Slack ready summary", ready.Summary)
	}
	if ready.NextCommand != "gormes gateway --channel slack" {
		t.Fatalf("NextCommand = %q, want channel handoff override", ready.NextCommand)
	}
}

func TestBuildFirstRunPlan_NonTTYGivesExactCommands(t *testing.T) {
	plan := BuildFirstRunPlan(FirstRunPlanInput{
		Interactive: false,
		Provider:    "groq",
		Target:      "wa",
		Channels:    DefaultFirstRunChannels(nil),
	})

	if plan.PromptAllowed {
		t.Fatal("PromptAllowed = true, want false for non-interactive plan")
	}
	if plan.Target != SetupTargetWhatsApp {
		t.Fatalf("Target = %q, want wa alias to whatsapp", plan.Target)
	}
	if plan.NextCommand != "gormes setup --quick --target whatsapp" {
		t.Fatalf("NextCommand = %q, want first core missing command before channel setup", plan.NextCommand)
	}
	provider, ok := plan.Step(FirstRunStepProvider)
	if !ok {
		t.Fatalf("missing provider step: %#v", plan.MissingSteps)
	}
	if provider.Command != "gormes setup --quick --target whatsapp" {
		t.Fatalf("provider Command = %q, want exact non-interactive setup command", provider.Command)
	}
	auth, ok := plan.Step(FirstRunStepAuth)
	if !ok {
		t.Fatalf("missing auth step: %#v", plan.MissingSteps)
	}
	if auth.Command != "gormes auth add groq" {
		t.Fatalf("auth Command = %q, want provider-specific auth command", auth.Command)
	}
	channel, ok := plan.Step(FirstRunStepChannel)
	if !ok {
		t.Fatalf("missing channel step: %#v", plan.MissingSteps)
	}
	if channel.Command != "gormes whatsapp" {
		t.Fatalf("channel Command = %q, want WhatsApp setup command", channel.Command)
	}
	if strings.Contains(strings.ToLower(plan.Summary), "prompt") {
		t.Fatalf("Summary = %q, non-interactive plan must not imply prompting", plan.Summary)
	}
}

func assertActionIDs(t *testing.T, actions []FirstRunAction, want []FirstRunActionID) {
	t.Helper()

	got := make([]FirstRunActionID, 0, len(actions))
	for _, action := range actions {
		got = append(got, action.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("action IDs = %#v, want %#v; actions=%#v", got, want, actions)
	}
}
