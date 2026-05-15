# Gormes First-Run CLI/TUI UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make fresh `gormes` launches route through target-aware setup instead of opening a broken chat, with shared readiness rules for root, setup, onboard, and doctor.

**Architecture:** Add a pure first-run planner in `internal/cli`, then keep Cobra commands as adapters around that planner. Root `gormes` checks the plan before Bubble Tea startup; setup quick uses the same plan to ask target first, run channel setup, run a tiny provider test, and hand off safely.

**Tech Stack:** Go 1.26, Cobra, existing Bubble Tea TUI boundary, existing Gormes config/seams/test helpers.

---

## Current Branch And Scope

- Work in the existing `development` branch only.
- Do not create a feature branch or worktree.
- Leave unrelated dirty files alone. At plan time the unrelated untracked file is `internal/installtest/local_cli_e2e_test.go`.
- Primary design spec: `webpages/docs/superpowers/specs/2026-05-14-gormes-first-run-cli-tui-ux-design.md`.

## Interface Decision

Two interface shapes were considered:

1. Cobra-local checks in `cmd/gormes/main.go`, `setup.go`, `onboard.go`, and `doctor.go`.
2. A pure planner in `internal/cli` with small command adapters.

Use option 2. It keeps readiness rules testable without provider, terminal, or gateway processes; it prevents root/setup/onboard/doctor from drifting; and it preserves the existing Bubble Tea boundary by keeping setup prompts outside the chat model.

Rejected option 1 because it duplicates "provider/auth/model/channel ready" checks across command files and makes non-TTY behavior harder to prove.

## File Map

- Create `internal/cli/first_run_plan.go`: pure target/readiness/router planner.
- Create `internal/cli/first_run_plan_test.go`: table tests for fresh install, migrations, target readiness, and non-TTY commands.
- Create `cmd/gormes/first_run.go`: root-command adapter, config-to-planner adapter, first-run guidance rendering.
- Create `cmd/gormes/root_first_run_test.go`: root command behavior tests.
- Create `cmd/gormes/setup_first_run.go`: first-run router menu, quick target menu, live provider test, target handoff.
- Create `cmd/gormes/setup_gateway_platform.go`: minimal channel config prompts for Telegram, Discord, Slack, and WhatsApp command routing.
- Modify `cmd/gormes/setup.go`: add `--target`, new seams, conditional migration options, platform target dispatch.
- Modify `cmd/gormes/setup_minimal_test.go`: extend fake seams for target selection/live test.
- Modify `cmd/gormes/setup_entry_mode_test.go`: update fresh install menu expectations for conditional migrations.
- Modify `cmd/gormes/setup_gateway_test.go`: include WhatsApp target and platform routing assertions.
- Modify `cmd/gormes/onboard.go`: render planner-backed readiness and next command.
- Modify `cmd/gormes/onboard_wizard_test.go` and `cmd/gormes/onboard_wizard_json_test.go`: assert planner wording stays parseable.
- Modify `cmd/gormes/doctor.go`: add `--target` and planner-backed target readiness summary.
- Modify `cmd/gormes/doctor_runE_test.go`: assert target readiness text and JSON.
- Modify `cmd/gormes/fresh_install_e2e_test.go`: add clean-home first-run checks.
- Modify `webpages/docs/superpowers/specs/2026-05-14-gormes-first-run-cli-tui-ux-design.md` only if implementation discovers an approved spec correction.

## Task 1: Add Pure First-Run Planner

**Files:**
- Create: `internal/cli/first_run_plan.go`
- Create: `internal/cli/first_run_plan_test.go`

- [ ] **Step 1: Write planner tests first**

Create `internal/cli/first_run_plan_test.go`:

```go
package cli

import "testing"

func TestBuildFirstRunPlan_FreshInstallRouterOptions(t *testing.T) {
	plan := BuildFirstRunPlan(FirstRunPlanInput{
		Interactive: true,
		Channels:   DefaultFirstRunChannels(nil),
	})

	if plan.Ready {
		t.Fatal("fresh install plan Ready = true, want false")
	}
	if plan.DefaultAction != FirstRunActionQuick {
		t.Fatalf("DefaultAction = %q, want %q", plan.DefaultAction, FirstRunActionQuick)
	}
	assertActionIDs(t, plan.Actions, []FirstRunActionID{FirstRunActionQuick, FirstRunActionFull})
	if plan.NextCommand != "gormes setup --quick --target terminal" {
		t.Fatalf("NextCommand = %q", plan.NextCommand)
	}
}

func TestBuildFirstRunPlan_MigrationsAppearOnlyWhenAvailable(t *testing.T) {
	plan := BuildFirstRunPlan(FirstRunPlanInput{
		Interactive:        true,
		HermesSourcePath:   "/tmp/hermes",
		OpenClawSourcePath: "/tmp/openclaw",
		Channels:          DefaultFirstRunChannels(nil),
	})

	assertActionIDs(t, plan.Actions, []FirstRunActionID{
		FirstRunActionQuick,
		FirstRunActionFull,
		FirstRunActionMigrateHermes,
		FirstRunActionMigrateOpenClaw,
	})
}

func TestBuildFirstRunPlan_ReadyTerminalNeedsCoreProviderPieces(t *testing.T) {
	plan := BuildFirstRunPlan(FirstRunPlanInput{
		Interactive:   true,
		Target:        SetupTargetTerminal,
		Endpoint:      "https://provider.example/v1",
		Model:         "test-model",
		APIKeyPresent: true,
		Channels:      DefaultFirstRunChannels(nil),
	})

	if !plan.Ready {
		t.Fatalf("Ready = false, missing=%+v", plan.MissingSteps)
	}
	if len(plan.MissingSteps) != 0 {
		t.Fatalf("MissingSteps = %+v, want none", plan.MissingSteps)
	}
}

func TestBuildFirstRunPlan_ChannelTargetIncludesChannelStep(t *testing.T) {
	plan := BuildFirstRunPlan(FirstRunPlanInput{
		Interactive:   true,
		Target:        SetupTargetTelegram,
		Endpoint:      "https://provider.example/v1",
		Model:         "test-model",
		APIKeyPresent: true,
		Channels: DefaultFirstRunChannels(map[SetupTargetID]ChannelState{
			SetupTargetTelegram: {Target: SetupTargetTelegram, Label: "Telegram", Configured: false},
		}),
	})

	if plan.Ready {
		t.Fatal("telegram plan Ready = true, want false while token is missing")
	}
	step, ok := plan.Step(FirstRunStepChannel)
	if !ok {
		t.Fatalf("MissingSteps = %+v, want channel step", plan.MissingSteps)
	}
	if step.Command != "gormes setup --quick --target telegram" {
		t.Fatalf("channel Command = %q", step.Command)
	}
}

func TestBuildFirstRunPlan_NonTTYGivesExactCommands(t *testing.T) {
	plan := BuildFirstRunPlan(FirstRunPlanInput{
		Interactive: false,
		Target:      SetupTargetWhatsApp,
		Channels:    DefaultFirstRunChannels(nil),
	})

	if plan.PromptAllowed {
		t.Fatal("PromptAllowed = true, want false")
	}
	if plan.NextCommand != "gormes setup --quick --target whatsapp" {
		t.Fatalf("NextCommand = %q", plan.NextCommand)
	}
	if len(plan.Targets) == 0 {
		t.Fatal("Targets empty, want terminal and channel targets")
	}
}

func assertActionIDs(t *testing.T, actions []FirstRunAction, want []FirstRunActionID) {
	t.Helper()
	got := make([]FirstRunActionID, 0, len(actions))
	for _, action := range actions {
		got = append(got, action.ID)
	}
	if len(got) != len(want) {
		t.Fatalf("actions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("actions = %v, want %v", got, want)
		}
	}
}
```

- [ ] **Step 2: Run planner tests and confirm red**

Run:

```bash
go test ./internal/cli -run 'TestBuildFirstRunPlan' -count=1
```

Expected:

```text
FAIL
undefined: BuildFirstRunPlan
```

- [ ] **Step 3: Implement the planner**

Create `internal/cli/first_run_plan.go`:

```go
package cli

import (
	"fmt"
	"strings"
)

type SetupTargetID string

const (
	SetupTargetTerminal SetupTargetID = "terminal"
	SetupTargetTelegram SetupTargetID = "telegram"
	SetupTargetWhatsApp SetupTargetID = "whatsapp"
	SetupTargetDiscord  SetupTargetID = "discord"
	SetupTargetSlack    SetupTargetID = "slack"
	SetupTargetNavivox  SetupTargetID = "navivox"
)

type FirstRunActionID string

const (
	FirstRunActionQuick           FirstRunActionID = "quick"
	FirstRunActionFull            FirstRunActionID = "full"
	FirstRunActionMigrateHermes   FirstRunActionID = "migrate_hermes"
	FirstRunActionMigrateOpenClaw FirstRunActionID = "migrate_openclaw"
)

type FirstRunStepID string

const (
	FirstRunStepProvider FirstRunStepID = "provider"
	FirstRunStepAuth     FirstRunStepID = "auth"
	FirstRunStepModel    FirstRunStepID = "model"
	FirstRunStepChannel  FirstRunStepID = "channel"
)

type FirstRunPlanInput struct {
	Interactive        bool
	Provider           string
	Endpoint           string
	Model              string
	APIKeyPresent      bool
	Target             SetupTargetID
	Channels           []ChannelState
	HermesSourcePath   string
	OpenClawSourcePath string
}

type ChannelState struct {
	Target         SetupTargetID
	Label          string
	Configured     bool
	Detail         string
	SetupCommand   string
	HandoffCommand string
}

type SetupTargetOption struct {
	ID             SetupTargetID
	Label          string
	Channel        bool
	Configured     bool
	Detail         string
	SetupCommand   string
	HandoffCommand string
}

type FirstRunAction struct {
	ID        FirstRunActionID
	Label     string
	Available bool
	Detail    string
	Command   string
}

type FirstRunStep struct {
	ID      FirstRunStepID
	Label   string
	Detail  string
	Command string
}

type FirstRunPlan struct {
	Ready         bool
	PromptAllowed bool
	Target        SetupTargetID
	TargetLabel   string
	DefaultTarget SetupTargetID
	Targets       []SetupTargetOption
	Actions       []FirstRunAction
	DefaultAction FirstRunActionID
	MissingSteps  []FirstRunStep
	NextCommand   string
	Summary       string
}

func BuildFirstRunPlan(input FirstRunPlanInput) FirstRunPlan {
	targets := targetOptions(input.Channels)
	target := normalizeSetupTarget(input.Target)
	if target == "" {
		target = SetupTargetTerminal
	}

	plan := FirstRunPlan{
		PromptAllowed: input.Interactive,
		Target:        target,
		TargetLabel:   setupTargetLabel(target),
		DefaultTarget: SetupTargetTerminal,
		Targets:       targets,
		Actions:       firstRunActions(input),
		DefaultAction: FirstRunActionQuick,
		NextCommand:   fmt.Sprintf("gormes setup --quick --target %s", target),
	}

	endpoint := strings.TrimSpace(input.Endpoint)
	model := strings.TrimSpace(input.Model)
	provider := strings.TrimSpace(input.Provider)
	if endpoint == "" {
		plan.MissingSteps = append(plan.MissingSteps, FirstRunStep{
			ID:      FirstRunStepProvider,
			Label:   "Provider endpoint",
			Detail:  "No provider endpoint is configured.",
			Command: "gormes setup provider",
		})
	}
	if !input.APIKeyPresent {
		command := "gormes setup provider"
		if provider != "" {
			command = "gormes auth add " + provider
		}
		plan.MissingSteps = append(plan.MissingSteps, FirstRunStep{
			ID:      FirstRunStepAuth,
			Label:   "Provider credential",
			Detail:  "No provider credential was found.",
			Command: command,
		})
	}
	if model == "" {
		plan.MissingSteps = append(plan.MissingSteps, FirstRunStep{
			ID:      FirstRunStepModel,
			Label:   "Default model",
			Detail:  "No default model is configured.",
			Command: "gormes setup model",
		})
	}

	if target != SetupTargetTerminal {
		option := findTargetOption(targets, target)
		if !option.Configured {
			command := option.SetupCommand
			if strings.TrimSpace(command) == "" {
				command = fmt.Sprintf("gormes setup --quick --target %s", target)
			}
			plan.MissingSteps = append(plan.MissingSteps, FirstRunStep{
				ID:      FirstRunStepChannel,
				Label:   setupTargetLabel(target) + " channel",
				Detail:  setupTargetLabel(target) + " is not configured.",
				Command: command,
			})
		}
	}

	plan.Ready = len(plan.MissingSteps) == 0
	if plan.Ready {
		if target == SetupTargetTerminal {
			plan.Summary = "terminal chat is ready"
			plan.NextCommand = "gormes"
		} else {
			handoff := findTargetOption(targets, target).HandoffCommand
			if strings.TrimSpace(handoff) == "" {
				handoff = "gormes gateway"
			}
			plan.Summary = setupTargetLabel(target) + " is ready"
			plan.NextCommand = handoff
		}
	} else {
		plan.Summary = "setup needed: " + plan.MissingSteps[0].Detail
	}
	return plan
}

func (p FirstRunPlan) Step(id FirstRunStepID) (FirstRunStep, bool) {
	for _, step := range p.MissingSteps {
		if step.ID == id {
			return step, true
		}
	}
	return FirstRunStep{}, false
}

func DefaultFirstRunChannels(overrides map[SetupTargetID]ChannelState) []ChannelState {
	base := []ChannelState{
		{Target: SetupTargetTelegram, Label: "Telegram", SetupCommand: "gormes setup --quick --target telegram", HandoffCommand: "gormes gateway"},
		{Target: SetupTargetWhatsApp, Label: "WhatsApp", SetupCommand: "gormes whatsapp", HandoffCommand: "gormes gateway"},
		{Target: SetupTargetDiscord, Label: "Discord", SetupCommand: "gormes setup --quick --target discord", HandoffCommand: "gormes gateway"},
		{Target: SetupTargetSlack, Label: "Slack", SetupCommand: "gormes setup --quick --target slack", HandoffCommand: "gormes gateway"},
		{Target: SetupTargetNavivox, Label: "Navivox", SetupCommand: "gormes setup --quick --target navivox", HandoffCommand: "gormes gateway"},
	}
	for i, row := range base {
		if override, ok := overrides[row.Target]; ok {
			if override.Label == "" {
				override.Label = row.Label
			}
			if override.SetupCommand == "" {
				override.SetupCommand = row.SetupCommand
			}
			if override.HandoffCommand == "" {
				override.HandoffCommand = row.HandoffCommand
			}
			base[i] = override
		}
	}
	return base
}

func firstRunActions(input FirstRunPlanInput) []FirstRunAction {
	actions := []FirstRunAction{
		{ID: FirstRunActionQuick, Label: "Quick setup", Available: true, Command: "gormes setup --quick"},
		{ID: FirstRunActionFull, Label: "Full setup", Available: true, Command: "gormes setup --reconfigure"},
	}
	if strings.TrimSpace(input.HermesSourcePath) != "" {
		actions = append(actions, FirstRunAction{
			ID: FirstRunActionMigrateHermes, Label: "Migrate from Hermes", Available: true,
			Detail: input.HermesSourcePath, Command: "gormes migrate hermes --dry-run",
		})
	}
	if strings.TrimSpace(input.OpenClawSourcePath) != "" {
		actions = append(actions, FirstRunAction{
			ID: FirstRunActionMigrateOpenClaw, Label: "Migrate from OpenClaw", Available: true,
			Detail: input.OpenClawSourcePath, Command: "gormes migrate openclaw --dry-run",
		})
	}
	return actions
}

func targetOptions(channels []ChannelState) []SetupTargetOption {
	options := []SetupTargetOption{{
		ID:             SetupTargetTerminal,
		Label:          "Terminal chat",
		Channel:        false,
		Configured:     true,
		HandoffCommand: "gormes",
	}}
	if len(channels) == 0 {
		channels = DefaultFirstRunChannels(nil)
	}
	for _, channel := range channels {
		id := normalizeSetupTarget(channel.Target)
		if id == "" || id == SetupTargetTerminal {
			continue
		}
		label := channel.Label
		if label == "" {
			label = setupTargetLabel(id)
		}
		setupCommand := channel.SetupCommand
		if setupCommand == "" {
			setupCommand = fmt.Sprintf("gormes setup --quick --target %s", id)
		}
		handoffCommand := channel.HandoffCommand
		if handoffCommand == "" {
			handoffCommand = "gormes gateway"
		}
		options = append(options, SetupTargetOption{
			ID:             id,
			Label:          label,
			Channel:        true,
			Configured:     channel.Configured,
			Detail:         channel.Detail,
			SetupCommand:   setupCommand,
			HandoffCommand: handoffCommand,
		})
	}
	return options
}

func findTargetOption(options []SetupTargetOption, target SetupTargetID) SetupTargetOption {
	for _, option := range options {
		if option.ID == target {
			return option
		}
	}
	return SetupTargetOption{ID: target, Label: setupTargetLabel(target), Channel: target != SetupTargetTerminal}
}

func normalizeSetupTarget(target SetupTargetID) SetupTargetID {
	switch SetupTargetID(strings.ToLower(strings.TrimSpace(string(target)))) {
	case SetupTargetTerminal, "chat", "tui":
		return SetupTargetTerminal
	case SetupTargetTelegram:
		return SetupTargetTelegram
	case SetupTargetWhatsApp, "wa":
		return SetupTargetWhatsApp
	case SetupTargetDiscord:
		return SetupTargetDiscord
	case SetupTargetSlack:
		return SetupTargetSlack
	case SetupTargetNavivox:
		return SetupTargetNavivox
	default:
		return ""
	}
}

func setupTargetLabel(target SetupTargetID) string {
	switch target {
	case SetupTargetTerminal:
		return "Terminal chat"
	case SetupTargetTelegram:
		return "Telegram"
	case SetupTargetWhatsApp:
		return "WhatsApp"
	case SetupTargetDiscord:
		return "Discord"
	case SetupTargetSlack:
		return "Slack"
	case SetupTargetNavivox:
		return "Navivox"
	default:
		return string(target)
	}
}
```

- [ ] **Step 4: Run planner tests and confirm green**

Run:

```bash
go test ./internal/cli -run 'TestBuildFirstRunPlan' -count=1
```

Expected:

```text
ok  	github.com/TrebuchetDynamics/gormes-agent/internal/cli
```

- [ ] **Step 5: Commit planner slice**

Run:

```bash
git add internal/cli/first_run_plan.go internal/cli/first_run_plan_test.go
git commit -m "feat: add first-run readiness planner"
```

Expected:

```text
[development <sha>] feat: add first-run readiness planner
```

## Task 2: Route Plain `gormes` Through First-Run Setup

**Files:**
- Create: `cmd/gormes/first_run.go`
- Create: `cmd/gormes/root_first_run_test.go`
- Modify: `cmd/gormes/main.go`

- [ ] **Step 1: Write root first-run tests**

Create `cmd/gormes/root_first_run_test.go`:

```go
package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootFreshInteractiveLaunchRoutesToFirstRunSetup(t *testing.T) {
	freshInstallE2EHome(t)
	setupCalls := 0
	tuiCalls := 0

	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return true },
		runFirstRunSetup: func(*cobra.Command) error {
			setupCalls++
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			tuiCalls++
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd)
	if err != nil {
		t.Fatalf("gormes: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if setupCalls != 1 || tuiCalls != 0 {
		t.Fatalf("setupCalls=%d tuiCalls=%d, want setup=1 tui=0", setupCalls, tuiCalls)
	}
}

func TestRootFreshNonTTYPrintsSetupGuidance(t *testing.T) {
	freshInstallE2EHome(t)
	tuiCalls := 0

	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return false },
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			tuiCalls++
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd)
	if err != nil {
		t.Fatalf("gormes: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if tuiCalls != 0 {
		t.Fatalf("tuiCalls=%d, want 0", tuiCalls)
	}
	for _, want := range []string{
		"Gormes setup needed",
		"Next: gormes setup --quick --target terminal",
		"Non-interactive mode will not prompt.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestRootReadyConfigStillOpensTUI(t *testing.T) {
	freshInstallE2EHome(t)
	writeOneshotFlagConfig(t, []byte(`
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o"
api_key = "sk-test-root-ready"
`))

	tuiCalls := 0
	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return true },
		runFirstRunSetup: func(*cobra.Command) error {
			t.Fatal("ready root launch invoked first-run setup")
			return nil
		},
		runResolvedTUI: func(*cobra.Command, tuiInvocation) error {
			tuiCalls++
			return nil
		},
	})

	stdout, stderr, err := executeRootCommandForTest(cmd)
	if err != nil {
		t.Fatalf("gormes: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if tuiCalls != 1 {
		t.Fatalf("tuiCalls=%d, want 1", tuiCalls)
	}
}
```

- [ ] **Step 2: Run root tests and confirm red**

Run:

```bash
go test ./cmd/gormes -run 'TestRootFresh|TestRootReady' -count=1
```

Expected:

```text
FAIL
unknown field isTTY in struct literal of type rootRuntime
```

- [ ] **Step 3: Add root runtime seams**

Modify `cmd/gormes/main.go`:

```go
type rootRuntime struct {
	runTUI                 func(*cobra.Command, []string) error
	runResolvedTUI         func(*cobra.Command, tuiInvocation) error
	runOneshot             func(*cobra.Command, oneshotInvocation) error
	newOneshotClient       oneshotClientFactory
	configureOneshotKernel oneshotKernelConfigurer
	tuiProgramFactory      tuiProgramFactory
	isTTY                  func() bool
	runFirstRunSetup       func(*cobra.Command) error
}
```

In `newRootCommandWithRuntime`, add defaults before building `runResolvedTUI`:

```go
if runtime.isTTY == nil {
	runtime.isTTY = isStdinTTY
}
if runtime.runFirstRunSetup == nil {
	runtime.runFirstRunSetup = runFirstRunSetupCommand
}
```

In `runRootCommand`, insert the first-run check after `resolveTUIInvocation` and before `runtime.runResolvedTUI`:

```go
	invocation, err := resolveTUIInvocation(cmd)
	if err != nil {
		return err
	}
	if handled, err := maybeHandleRootFirstRun(cmd, invocation, runtime); handled || err != nil {
		return err
	}
	return runtime.runResolvedTUI(cmd, invocation)
```

- [ ] **Step 4: Add root first-run adapter**

Create `cmd/gormes/first_run.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

func maybeHandleRootFirstRun(cmd *cobra.Command, invocation tuiInvocation, runtime rootRuntime) (bool, error) {
	if rootFirstRunBypass(cmd, invocation) {
		return false, nil
	}
	plan := buildFirstRunPlanFromConfig(invocation.Config, cli.SetupTargetTerminal, runtime.isTTY())
	if plan.Ready {
		return false, nil
	}
	if runtime.isTTY() {
		return true, runtime.runFirstRunSetup(cmd)
	}
	printFirstRunGuidance(cmd, plan)
	return true, nil
}

func rootFirstRunBypass(cmd *cobra.Command, invocation tuiInvocation) bool {
	if strings.TrimSpace(invocation.RemoteURL) != "" {
		return true
	}
	offline, _ := cmd.Flags().GetBool("offline")
	return offline
}

func runFirstRunSetupCommand(cmd *cobra.Command) error {
	setup := newSetupCommand()
	setup.SetIn(cmd.InOrStdin())
	setup.SetOut(cmd.OutOrStdout())
	setup.SetErr(cmd.ErrOrStderr())
	setup.SetArgs([]string{})
	setup.SilenceUsage = true
	setup.SilenceErrors = true
	return setup.ExecuteContext(cmd.Context())
}

func buildFirstRunPlanFromConfig(cfg config.Config, target cli.SetupTargetID, interactive bool) cli.FirstRunPlan {
	return cli.BuildFirstRunPlan(cli.FirstRunPlanInput{
		Interactive:        interactive,
		Provider:           cfg.Hermes.Provider,
		Endpoint:           cfg.Hermes.Endpoint,
		Model:              cfg.Hermes.Model,
		APIKeyPresent:      configuredProviderAuthPresent(cfg),
		Target:             target,
		Channels:           firstRunChannelStates(cfg),
		HermesSourcePath:   detectHermesMigrationSource(),
		OpenClawSourcePath: detectOpenClawMigrationSource(),
	})
}

func firstRunChannelStates(cfg config.Config) []cli.ChannelState {
	overrides := map[cli.SetupTargetID]cli.ChannelState{}
	overrides[cli.SetupTargetTelegram] = cli.ChannelState{
		Target:         cli.SetupTargetTelegram,
		Label:          "Telegram",
		Configured:     strings.TrimSpace(cfg.Telegram.BotToken) != "",
		Detail:         configuredTelegramGatewayStatusDetail(cfg.Telegram),
		SetupCommand:   "gormes setup --quick --target telegram",
		HandoffCommand: "gormes gateway",
	}
	overrides[cli.SetupTargetWhatsApp] = cli.ChannelState{
		Target:         cli.SetupTargetWhatsApp,
		Label:          "WhatsApp",
		Configured:     strings.EqualFold(strings.TrimSpace(os.Getenv("WHATSAPP_ENABLED")), "true"),
		SetupCommand:   "gormes whatsapp",
		HandoffCommand: "gormes gateway",
	}
	overrides[cli.SetupTargetDiscord] = cli.ChannelState{
		Target:         cli.SetupTargetDiscord,
		Label:          "Discord",
		Configured:     cfg.Discord.Enabled(),
		SetupCommand:   "gormes setup --quick --target discord",
		HandoffCommand: "gormes gateway",
	}
	overrides[cli.SetupTargetSlack] = cli.ChannelState{
		Target:         cli.SetupTargetSlack,
		Label:          "Slack",
		Configured:     cfg.Slack.Enabled,
		Detail:         configuredSlackGatewayStatusDetail(cfg.Slack),
		SetupCommand:   "gormes setup --quick --target slack",
		HandoffCommand: "gormes gateway",
	}
	overrides[cli.SetupTargetNavivox] = cli.ChannelState{
		Target:         cli.SetupTargetNavivox,
		Label:          "Navivox",
		Configured:     cfg.Navivox.Enabled,
		Detail:         configuredNavivoxGatewayStatusDetail(cfg.Navivox),
		SetupCommand:   "gormes setup --quick --target navivox",
		HandoffCommand: "gormes gateway",
	}
	return cli.DefaultFirstRunChannels(overrides)
}

func printFirstRunGuidance(cmd *cobra.Command, plan cli.FirstRunPlan) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Gormes setup needed")
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Target: %s\n", plan.TargetLabel)
	fmt.Fprintf(out, "Status: %s\n", plan.Summary)
	if len(plan.MissingSteps) > 0 {
		fmt.Fprintln(out, "Missing:")
		for _, step := range plan.MissingSteps {
			fmt.Fprintf(out, "  - %s: %s\n", step.Label, step.Detail)
		}
	}
	fmt.Fprintf(out, "Next: %s\n", plan.NextCommand)
	fmt.Fprintln(out, "Non-interactive mode will not prompt.")
}

func detectHermesMigrationSource() string {
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".hermes")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate
		}
	}
	return strings.TrimSpace(os.Getenv("HERMES_HOME"))
}

func detectOpenClawMigrationSource() string {
	if home, err := os.UserHomeDir(); err == nil {
		for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
			candidate := filepath.Join(home, dir)
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				return candidate
			}
		}
	}
	return ""
}
```

- [ ] **Step 5: Run root tests and confirm green**

Run:

```bash
go test ./cmd/gormes -run 'TestRootFresh|TestRootReady' -count=1
```

Expected:

```text
ok  	github.com/TrebuchetDynamics/gormes-agent/cmd/gormes
```

- [ ] **Step 6: Commit root slice**

Run:

```bash
git add cmd/gormes/main.go cmd/gormes/first_run.go cmd/gormes/root_first_run_test.go
git commit -m "feat: route fresh root launches through setup"
```

Expected:

```text
[development <sha>] feat: route fresh root launches through setup
```

## Task 3: Make Setup Quick Target-First

**Files:**
- Create: `cmd/gormes/setup_first_run.go`
- Modify: `cmd/gormes/setup.go`
- Modify: `cmd/gormes/setup_minimal_test.go`
- Modify: `cmd/gormes/setup_entry_mode_test.go`

- [ ] **Step 1: Extend setup tests for target-first quick flow**

Add these tests to `cmd/gormes/setup_entry_mode_test.go`:

```go
func TestSetupQuickPromptsTargetBeforeProviderWork(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true, current: cli.ProviderModel{}}
	var events []string
	seams := fake.seams()
	seams.ChooseSetupTarget = func(_ *cobra.Command, _ []cli.SetupTargetOption, defaultOption int) (cli.SetupTargetID, error) {
		events = append(events, "target")
		if defaultOption != 0 {
			t.Fatalf("default target index = %d, want 0", defaultOption)
		}
		return cli.SetupTargetTerminal, nil
	}
	seams.RunModelPicker = func(*cobra.Command) error {
		events = append(events, "model")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		events = append(events, "chat")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick")
	if err != nil {
		t.Fatalf("setup --quick: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Join(events, ",") != "target,model,live-test,chat" {
		t.Fatalf("events=%v, want target,model,live-test,chat", events)
	}
}

func TestSetupQuickNonInteractivePrintsTargetCommands(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: false, freshInstall: true}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "--quick", "--non-interactive")
	if err != nil {
		t.Fatalf("setup --quick --non-interactive: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Quick setup targets:",
		"gormes setup --quick --target terminal",
		"gormes setup --quick --target telegram",
		"gormes whatsapp",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupFirstRunRouterShowsConditionalMigrations(t *testing.T) {
	fake := &setupCommandFakeSeams{isTTY: true, freshInstall: true}
	seams := fake.seams()
	seams.DetectHermesMigrationSource = func() string { return "/tmp/hermes" }
	seams.DetectOpenClawMigrationSource = func() string { return "/tmp/openclaw" }
	seams.ChooseSetupAction = func(_ *cobra.Command, options []setupMenuOption, defaultOption int) (setupAction, error) {
		if defaultOption != 0 {
			t.Fatalf("default option = %d, want 0", defaultOption)
		}
		if len(options) != 4 {
			t.Fatalf("options=%#v, want quick/full/hermes/openclaw", options)
		}
		if options[2].Action != setupActionMigrateHermes || options[3].Action != setupActionMigrateOpenClaw {
			t.Fatalf("options=%#v, migrations not conditional tail", options)
		}
		return setupActionExit, nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams)
	if err != nil {
		t.Fatalf("setup: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Migrate from Hermes") || !strings.Contains(stdout, "Migrate from OpenClaw") {
		t.Fatalf("stdout missing migrations:\n%s", stdout)
	}
}
```

- [ ] **Step 2: Extend setup fake seams**

Modify `cmd/gormes/setup_minimal_test.go`:

```go
type setupCommandFakeSeams struct {
	isTTY        bool
	freshInstall bool
	current      cli.ProviderModel

	modelPickerCalls int
	loadedCurrent    int

	chooseSetupAction           func(*cobra.Command, []setupMenuOption, int) (setupAction, error)
	chooseSetupTarget           func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error)
	runFullWizard               func(*cobra.Command, bool) error
	runSetupGateway             func(*cobra.Command, bool) error
	runGatewayPlatform          func(*cobra.Command, string) error
	runProviderLiveTest         func(*cobra.Command) error
	detectHermesMigrationSource func() string
	detectOpenClawMigrationSource func() string
}
```

In `seams()` return these fields:

```go
		ChooseSetupTarget:            f.chooseSetupTarget,
		RunProviderLiveTest:          f.runProviderLiveTest,
		DetectHermesMigrationSource:  f.detectHermesMigrationSource,
		DetectOpenClawMigrationSource: f.detectOpenClawMigrationSource,
```

- [ ] **Step 3: Run setup tests and confirm red**

Run:

```bash
go test ./cmd/gormes -run 'TestSetupQuick|TestSetupFirstRunRouter|TestSetupEntryMode_FreshInstall' -count=1
```

Expected:

```text
FAIL
unknown field ChooseSetupTarget in struct literal of type setupCommandSeams
```

- [ ] **Step 4: Add setup seams and flags**

Modify `cmd/gormes/setup.go` `setupCommandSeams`:

```go
type setupCommandSeams struct {
	IsTTY                       func() bool
	HasExistingInstall          func() (bool, error)
	ResetConfig                 func() (string, error)
	RunModelPicker              func(*cobra.Command) error
	LoadCurrentModel            func() (cli.ProviderModel, error)
	ChooseSetupAction           func(*cobra.Command, []setupMenuOption, int) (setupAction, error)
	ChooseSetupTarget           func(*cobra.Command, []cli.SetupTargetOption, int) (cli.SetupTargetID, error)
	RunFullWizard               func(*cobra.Command, bool) error
	RunSetupGateway             func(*cobra.Command, bool) error
	RunSetupTools               func(*cobra.Command, bool) error
	RunGatewayPlatform          func(*cobra.Command, string) error
	RunProviderLiveTest         func(*cobra.Command) error
	DetectHermesMigrationSource func() string
	DetectOpenClawMigrationSource func() string
	LaunchChat                  func(*cobra.Command) error
}
```

In `newSetupCommandWithSeams`, add defaults:

```go
	if seams.ChooseSetupTarget == nil {
		seams.ChooseSetupTarget = promptSetupTarget
	}
	if seams.RunProviderLiveTest == nil {
		seams.RunProviderLiveTest = runSetupProviderLiveTest
	}
	if seams.DetectHermesMigrationSource == nil {
		seams.DetectHermesMigrationSource = detectHermesMigrationSource
	}
	if seams.DetectOpenClawMigrationSource == nil {
		seams.DetectOpenClawMigrationSource = detectOpenClawMigrationSource
	}
```

Add `targetFlag` beside existing setup flags:

```go
	var targetFlag string
```

Register it:

```go
	cmd.Flags().StringVar(&targetFlag, "target", "", "setup target for --quick: terminal, telegram, whatsapp, discord, slack, or navivox")
```

Change quick calls:

```go
if quick {
	return runSetupQuick(cmd, seams, nonInteractive || !seams.IsTTY(), cli.SetupTargetID(targetFlag))
}
```

Change the function signature:

```go
func runSetupQuick(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool, requestedTarget cli.SetupTargetID) error
```

Update existing callers in `runSetupRoot` and `runSetupFirstTimeChoice` to pass `""`.

- [ ] **Step 5: Implement first-run router and quick target flow**

Create `cmd/gormes/setup_first_run.go`:

```go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

func firstRunSetupOptions(seams setupCommandSeams) []setupMenuOption {
	options := []setupMenuOption{
		{Action: setupActionQuick, Label: "Quick setup - provider, model, and messaging target"},
		{Action: setupActionFull, Label: "Full setup - configure everything"},
	}
	if source := strings.TrimSpace(seams.DetectHermesMigrationSource()); source != "" {
		options = append(options, setupMenuOption{Action: setupActionMigrateHermes, Label: "Migrate from Hermes"})
	}
	if source := strings.TrimSpace(seams.DetectOpenClawMigrationSource()); source != "" {
		options = append(options, setupMenuOption{Action: setupActionMigrateOpenClaw, Label: "Migrate from OpenClaw"})
	}
	return options
}

func printQuickSetupTargets(cmd *cobra.Command, targets []cli.SetupTargetOption) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Quick setup targets:")
	for _, target := range targets {
		command := target.SetupCommand
		if command == "" {
			command = fmt.Sprintf("gormes setup --quick --target %s", target.ID)
		}
		if target.ID == cli.SetupTargetTerminal {
			command = "gormes setup --quick --target terminal"
		}
		fmt.Fprintf(out, "  - %-10s %s\n", target.ID, command)
	}
}

func promptSetupTarget(cmd *cobra.Command, targets []cli.SetupTargetOption, defaultOption int) (cli.SetupTargetID, error) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Where do you want to use Gormes first?")
	fmt.Fprintln(out)
	for i, target := range targets {
		marker := "( )"
		if i == defaultOption {
			marker = "(*)"
		}
		fmt.Fprintf(out, "  %d. %s %-12s", i+1, marker, target.Label)
		if target.Configured && target.Detail != "" {
			fmt.Fprintf(out, " configured (%s)", target.Detail)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out)
	answer, err := promptString(cmd, fmt.Sprintf("Select target [%d]: ", defaultOption+1), strconv.Itoa(defaultOption+1))
	if err != nil {
		return "", err
	}
	answer = strings.ToLower(strings.TrimSpace(cli.StripANSI(answer)))
	if answer == "" {
		return targets[defaultOption].ID, nil
	}
	if n, err := strconv.Atoi(answer); err == nil && n >= 1 && n <= len(targets) {
		return targets[n-1].ID, nil
	}
	for _, target := range targets {
		if answer == string(target.ID) || strings.Contains(strings.ToLower(target.Label), answer) {
			return target.ID, nil
		}
	}
	return "", newExitCodeError(2, fmt.Errorf("setup_target_invalid_selection: %s", answer))
}

func runSetupQuick(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool, requestedTarget cli.SetupTargetID) error {
	out := cmd.OutOrStdout()
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Quick Setup")

	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("quick setup: load config: %w", err)
	}
	target := requestedTarget
	plan := buildFirstRunPlanFromConfig(cfg, target, !nonInteractive)
	if target == "" {
		if nonInteractive {
			printQuickSetupTargets(cmd, plan.Targets)
			fmt.Fprintln(out)
			printFirstRunGuidance(cmd, plan)
			return nil
		}
		selected, err := seams.ChooseSetupTarget(cmd, plan.Targets, 0)
		if err != nil {
			return err
		}
		target = selected
		plan = buildFirstRunPlanFromConfig(cfg, target, true)
	}
	if target == "" {
		target = cli.SetupTargetTerminal
	}

	if err := runSetupQuickCore(cmd, seams, nonInteractive); err != nil {
		return err
	}
	if target != cli.SetupTargetTerminal {
		if err := runSetupQuickChannel(cmd, seams, target, nonInteractive); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "Running provider live test...")
	if err := seams.RunProviderLiveTest(cmd); err != nil {
		fmt.Fprintln(out, "Provider live test failed. Chat was not opened.")
		fmt.Fprintf(out, "Repair: %s\n", buildFirstRunPlanFromConfig(cfg, target, false).NextCommand)
		return newExitCodeError(1, fmt.Errorf("quick setup live test failed: %w", err))
	}
	return runSetupQuickHandoff(cmd, seams, target)
}

func runSetupQuickCore(cmd *cobra.Command, seams setupCommandSeams, nonInteractive bool) error {
	current, err := seams.LoadCurrentModel()
	if err != nil {
		return fmt.Errorf("quick setup: load current model: %w", err)
	}
	if strings.TrimSpace(current.Provider) == "" || strings.TrimSpace(current.Model) == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "Model/provider defaults are missing.")
		return runSetupModelSection(cmd, seams, nonInteractive)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Current model/provider: %s via %s\n", current.Model, current.Provider)
	return nil
}

func runSetupQuickChannel(cmd *cobra.Command, seams setupCommandSeams, target cli.SetupTargetID, nonInteractive bool) error {
	switch target {
	case cli.SetupTargetWhatsApp:
		if nonInteractive {
			fmt.Fprintln(cmd.OutOrStdout(), "WhatsApp setup command: gormes whatsapp --plan")
			return nil
		}
		return runSetupWhatsAppTarget(cmd)
	case cli.SetupTargetTelegram, cli.SetupTargetDiscord, cli.SetupTargetSlack, cli.SetupTargetNavivox:
		return seams.RunGatewayPlatform(cmd, string(target))
	default:
		return newExitCodeError(2, fmt.Errorf("setup_target_unsupported: %s", target))
	}
}

func runSetupQuickHandoff(cmd *cobra.Command, seams setupCommandSeams, target cli.SetupTargetID) error {
	if target == cli.SetupTargetTerminal {
		return seams.LaunchChat(cmd)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Channel setup checked. Start messaging with: gormes gateway")
	return nil
}

func runSetupProviderLiveTest(cmd *cobra.Command) error {
	root := newRootCommand()
	root.SetIn(cmd.InOrStdin())
	root.SetOut(cmd.OutOrStdout())
	root.SetErr(cmd.ErrOrStderr())
	root.SetArgs([]string{"--oneshot", "Reply with OK for Gormes setup."})
	root.SilenceUsage = true
	root.SilenceErrors = true
	return root.ExecuteContext(cmd.Context())
}

func runSetupWhatsAppTarget(cmd *cobra.Command) error {
	wa := newWhatsAppCommand()
	wa.SetIn(cmd.InOrStdin())
	wa.SetOut(cmd.OutOrStdout())
	wa.SetErr(cmd.ErrOrStderr())
	wa.SetArgs([]string{})
	wa.SilenceUsage = true
	wa.SilenceErrors = true
	return wa.ExecuteContext(cmd.Context())
}
```

- [ ] **Step 6: Update `runSetupFirstTimeChoice` to use conditional options**

Replace the fixed quick/full options in `cmd/gormes/setup.go` with:

```go
	options := firstRunSetupOptions(seams)
```

Extend the switch:

```go
	case setupActionMigrateHermes:
		return runSetupMigrate(cmd, "hermes")
	case setupActionMigrateOpenClaw:
		return runSetupMigrate(cmd, "openclaw")
```

- [ ] **Step 7: Run setup tests and confirm green**

Run:

```bash
go test ./cmd/gormes -run 'TestSetupQuick|TestSetupFirstRunRouter|TestSetupEntryMode_FreshInstall' -count=1
```

Expected:

```text
ok  	github.com/TrebuchetDynamics/gormes-agent/cmd/gormes
```

- [ ] **Step 8: Commit quick setup slice**

Run:

```bash
git add cmd/gormes/setup.go cmd/gormes/setup_first_run.go cmd/gormes/setup_minimal_test.go cmd/gormes/setup_entry_mode_test.go
git commit -m "feat: make quick setup target first"
```

Expected:

```text
[development <sha>] feat: make quick setup target first
```

## Task 4: Add Minimal Channel Configuration Targets

**Files:**
- Create: `cmd/gormes/setup_gateway_platform.go`
- Modify: `cmd/gormes/setup.go`
- Modify: `cmd/gormes/setup_gateway_test.go`

- [ ] **Step 1: Write gateway target tests**

Add to `cmd/gormes/setup_gateway_test.go`:

```go
func TestSetupGatewayChecklistIncludesWhatsApp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway", "--non-interactive")
	if err != nil {
		t.Fatalf("setup gateway: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"WhatsApp", "whatsapp"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupGatewayTelegramWritesTokenWithoutLeaking(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	stdout, stderr, err := runSetupTestCommandWithInput(t, setupCommandSeams{
		IsTTY: func() bool { return true },
	}, "telegram\n123456:secret-token\n4242\n", "gateway")
	if err != nil {
		t.Fatalf("setup gateway telegram: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "123456:secret-token") {
		t.Fatalf("telegram token leaked:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	cfg, loadErr := config.Load(nil)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if cfg.Telegram.BotToken != "123456:secret-token" || cfg.Telegram.AllowedChatID != 4242 {
		t.Fatalf("telegram cfg = %+v", cfg.Telegram)
	}
}

func TestSetupQuickWhatsAppTargetUsesWhatsAppCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	calls := 0
	seams := setupCommandSeams{
		IsTTY: func() bool { return true },
		HasExistingInstall: func() (bool, error) { return true, nil },
		LoadCurrentModel: func() (cli.ProviderModel, error) {
			return cli.ProviderModel{Provider: "openai", Model: "gpt-4o"}, nil
		},
		RunProviderLiveTest: func(*cobra.Command) error { return nil },
		LaunchChat: func(*cobra.Command) error {
			t.Fatal("whatsapp quick setup launched terminal chat")
			return nil
		},
		RunGatewayPlatform: func(*cobra.Command, string) error {
			t.Fatal("whatsapp quick setup used generic gateway platform")
			return nil
		},
		RunWhatsAppSetup: func(*cobra.Command) error {
			calls++
			return nil
		},
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "whatsapp")
	if err != nil {
		t.Fatalf("setup --quick --target whatsapp: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if calls != 1 {
		t.Fatalf("RunWhatsAppSetup calls = %d, want 1", calls)
	}
	if !strings.Contains(stdout, "Start messaging with: gormes gateway") {
		t.Fatalf("stdout missing gateway handoff:\n%s", stdout)
	}
}
```

- [ ] **Step 2: Add WhatsApp seam**

Modify `setupCommandSeams` in `cmd/gormes/setup.go`:

```go
	RunWhatsAppSetup func(*cobra.Command) error
```

Default it in `newSetupCommandWithSeams`:

```go
	if seams.RunWhatsAppSetup == nil {
		seams.RunWhatsAppSetup = runSetupWhatsAppTarget
	}
```

Update `runSetupQuickChannel` in `cmd/gormes/setup_first_run.go`:

```go
	case cli.SetupTargetWhatsApp:
		if nonInteractive {
			fmt.Fprintln(cmd.OutOrStdout(), "WhatsApp setup command: gormes whatsapp --plan")
			return nil
		}
		return seams.RunWhatsAppSetup(cmd)
```

- [ ] **Step 3: Run tests and confirm red on missing platform code**

Run:

```bash
go test ./cmd/gormes -run 'TestSetupGatewayChecklistIncludesWhatsApp|TestSetupGatewayTelegramWritesTokenWithoutLeaking|TestSetupQuickWhatsAppTargetUsesWhatsAppCommand' -count=1
```

Expected:

```text
FAIL
stdout missing "WhatsApp"
```

- [ ] **Step 4: Implement minimal gateway platform setup**

Create `cmd/gormes/setup_gateway_platform.go`:

```go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

func runSetupGatewayPlatform(cmd *cobra.Command, platform string) error {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "telegram":
		return runSetupTelegramPlatform(cmd)
	case "discord":
		return runSetupDiscordPlatform(cmd)
	case "slack":
		return runSetupSlackPlatform(cmd)
	case "whatsapp":
		return runSetupWhatsAppTarget(cmd)
	default:
		return runSetupGatewayPlatformRowBacked(cmd, platform)
	}
}

func runSetupTelegramPlatform(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Telegram setup")
	token, err := promptSecret(cmd, "Bot token: ")
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return newExitCodeError(2, fmt.Errorf("setup_telegram_token_required"))
	}
	chatIDText, err := promptString(cmd, "Allowed chat ID (blank for first-run discovery): ", "")
	if err != nil {
		return err
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.bot_token", token); err != nil {
		return fmt.Errorf("write telegram token: %w", err)
	}
	chatIDText = strings.TrimSpace(chatIDText)
	if chatIDText != "" {
		chatID, parseErr := strconv.ParseInt(chatIDText, 10, 64)
		if parseErr != nil {
			return newExitCodeError(2, fmt.Errorf("setup_telegram_allowed_chat_id_invalid: %s", chatIDText))
		}
		if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.allowed_chat_id", strconv.FormatInt(chatID, 10)); err != nil {
			return fmt.Errorf("write telegram allowed chat: %w", err)
		}
	} else if err := config.WriteTOMLValue(config.ConfigPath(), "telegram.first_run_discovery", "true"); err != nil {
		return fmt.Errorf("write telegram discovery: %w", err)
	}
	fmt.Fprintln(out, "Telegram configured. Token was stored and not printed.")
	return nil
}

func runSetupDiscordPlatform(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Discord setup")
	token, err := promptSecret(cmd, "Bot token: ")
	if err != nil {
		return err
	}
	channelID, err := promptString(cmd, "Allowed channel ID: ", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" || strings.TrimSpace(channelID) == "" {
		return newExitCodeError(2, fmt.Errorf("setup_discord_token_and_channel_required"))
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "discord.token", token); err != nil {
		return fmt.Errorf("write discord token: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "discord.allowed_channel_id", channelID); err != nil {
		return fmt.Errorf("write discord channel: %w", err)
	}
	fmt.Fprintln(out, "Discord configured. Token was stored and not printed.")
	return nil
}

func runSetupSlackPlatform(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Slack setup")
	botToken, err := promptSecret(cmd, "Bot token (xoxb-...): ")
	if err != nil {
		return err
	}
	appToken, err := promptSecret(cmd, "App token (xapp-...): ")
	if err != nil {
		return err
	}
	channelID, err := promptString(cmd, "Allowed channel ID: ", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(botToken) == "" || strings.TrimSpace(appToken) == "" || strings.TrimSpace(channelID) == "" {
		return newExitCodeError(2, fmt.Errorf("setup_slack_tokens_and_channel_required"))
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.enabled", "true"); err != nil {
		return fmt.Errorf("write slack enabled: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.bot_token", botToken); err != nil {
		return fmt.Errorf("write slack bot token: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.app_token", appToken); err != nil {
		return fmt.Errorf("write slack app token: %w", err)
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "slack.allowed_channel_id", channelID); err != nil {
		return fmt.Errorf("write slack channel: %w", err)
	}
	fmt.Fprintln(out, "Slack configured. Tokens were stored and not printed.")
	return nil
}
```

Modify `cmd/gormes/setup.go` defaults:

```go
	if seams.RunGatewayPlatform == nil {
		seams.RunGatewayPlatform = runSetupGatewayPlatform
	}
```

Modify `setupGatewayPlatformOptions` platform list:

```go
	for _, key := range []string{"telegram", "whatsapp", "discord", "slack", "navivox"} {
```

Modify `setupGatewayPlatformFallbackLabel`:

```go
	case "whatsapp":
		return "WhatsApp"
```

- [ ] **Step 5: Run gateway setup tests and confirm green**

Run:

```bash
go test ./cmd/gormes -run 'TestSetupGateway' -count=1
```

Expected:

```text
ok  	github.com/TrebuchetDynamics/gormes-agent/cmd/gormes
```

- [ ] **Step 6: Commit channel slice**

Run:

```bash
git add cmd/gormes/setup.go cmd/gormes/setup_gateway_platform.go cmd/gormes/setup_gateway_test.go cmd/gormes/setup_minimal_test.go
git commit -m "feat: add target channel setup prompts"
```

Expected:

```text
[development <sha>] feat: add target channel setup prompts
```

## Task 5: Add Planner Language To Onboard And Doctor

**Files:**
- Modify: `cmd/gormes/onboard.go`
- Modify: `cmd/gormes/onboard_wizard_test.go`
- Modify: `cmd/gormes/onboard_wizard_json_test.go`
- Modify: `cmd/gormes/doctor.go`
- Modify: `cmd/gormes/doctor_runE_test.go`

- [ ] **Step 1: Add onboard assertions**

Add to `cmd/gormes/onboard_wizard_test.go`:

```go
func TestOnboardStatusPrintsFirstRunNextCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "onboard")
	if err != nil {
		t.Fatalf("onboard: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"First-run readiness: setup needed",
		"Next: gormes setup --quick --target terminal",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}
```

Add to `cmd/gormes/onboard_wizard_json_test.go`:

```go
func TestOnboardJSONIncludesFirstRunReadiness(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "onboard", "--json")
	if err != nil {
		t.Fatalf("onboard --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		FirstRun struct {
			Ready       bool   `json:"ready"`
			Target      string `json:"target"`
			NextCommand string `json:"next_command"`
		} `json:"first_run"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", err, stdout)
	}
	if got.FirstRun.Ready {
		t.Fatal("first_run.ready = true, want false for fresh install")
	}
	if got.FirstRun.Target != "terminal" || got.FirstRun.NextCommand != "gormes setup --quick --target terminal" {
		t.Fatalf("first_run = %+v", got.FirstRun)
	}
}
```

- [ ] **Step 2: Add doctor assertions**

Add to `cmd/gormes/doctor_runE_test.go`:

```go
func TestDoctorTargetTelegramReportsMissingChannelCommand(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("GORMES_API_KEY", "")

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--target", "telegram")
	if err != nil {
		t.Fatalf("doctor --offline --target telegram: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"target readiness",
		"Telegram is not configured",
		"gormes setup --quick --target telegram",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestDoctorJSONIncludesTargetReadiness(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "doctor", "--offline", "--target", "whatsapp", "--json")
	if err != nil {
		t.Fatalf("doctor --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		Target struct {
			Name        string `json:"name"`
			Ready       bool   `json:"ready"`
			NextCommand string `json:"next_command"`
		} `json:"target"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Target.Name != "whatsapp" || got.Target.Ready || got.Target.NextCommand != "gormes setup --quick --target whatsapp" {
		t.Fatalf("target = %+v", got.Target)
	}
}
```

- [ ] **Step 3: Run onboard and doctor tests and confirm red**

Run:

```bash
go test ./cmd/gormes -run 'TestOnboard.*FirstRun|TestDoctorTarget|TestDoctorJSONIncludesTarget' -count=1
```

Expected:

```text
FAIL
stdout missing "First-run readiness"
```

- [ ] **Step 4: Add onboard JSON field and text**

Modify `cmd/gormes/onboard.go` `onboardStatusReportJSON`:

```go
	FirstRun onboardFirstRunJSON `json:"first_run"`
```

Add type:

```go
type onboardFirstRunJSON struct {
	Ready       bool     `json:"ready"`
	Target      string   `json:"target"`
	NextCommand string   `json:"next_command"`
	Missing     []string `json:"missing"`
}
```

In `writeOnboardStatusJSON`, build and assign:

```go
	firstRun := buildFirstRunPlanFromConfig(cfg, cli.SetupTargetTerminal, false)
	report.FirstRun = onboardFirstRunJSON{
		Ready:       firstRun.Ready,
		Target:      string(firstRun.Target),
		NextCommand: firstRun.NextCommand,
	}
	for _, step := range firstRun.MissingSteps {
		report.FirstRun.Missing = append(report.FirstRun.Missing, string(step.ID))
	}
```

In `printOnboardStatus`, after runtime skills output and before provider details:

```go
	firstRun := buildFirstRunPlanFromConfig(cfg, cli.SetupTargetTerminal, false)
	fmt.Fprintf(out, "First-run readiness: %s\n", firstRun.Summary)
	fmt.Fprintf(out, "Next: %s\n", firstRun.NextCommand)
	fmt.Fprintln(out)
```

- [ ] **Step 5: Add doctor target field and check**

Modify `cmd/gormes/doctor.go` flags:

```go
	cmd.Flags().String("target", "terminal", "target readiness to check: terminal, telegram, whatsapp, discord, slack, or navivox")
```

Modify `doctorReportJSON`:

```go
	Target doctorTargetJSON `json:"target,omitempty"`
```

Add types and reporter field:

```go
type doctorTargetJSON struct {
	Name        string   `json:"name"`
	Ready       bool     `json:"ready"`
	Summary     string   `json:"summary"`
	NextCommand string   `json:"next_command"`
	Missing     []string `json:"missing"`
}

type doctorReporter struct {
	w         io.Writer
	asJSON    bool
	target    doctorTargetJSON
	collected []doctor.CheckResult
	failed    bool
}
```

In `Finalize`, include target:

```go
	body, err := json.MarshalIndent(doctorReportJSON{
		Build:  newBuildProvenance(),
		Failed: r.failed,
		Target: r.target,
		Checks: checks,
	}, "", "  ")
```

After config load and secret activation, add:

```go
		targetFlag, _ := cmd.Flags().GetString("target")
		targetPlan := buildFirstRunPlanFromConfig(cfg, cli.SetupTargetID(targetFlag), false)
		reporter.target = doctorTargetFromPlan(targetPlan)
		status := doctor.StatusPass
		if !targetPlan.Ready {
			status = doctor.StatusWarn
		}
		reporter.Add(doctor.CheckResult{
			Name:    "target readiness",
			Status:  status,
			Summary: fmt.Sprintf("%s; next=%s", targetPlan.Summary, targetPlan.NextCommand),
		})
```

Add helper:

```go
func doctorTargetFromPlan(plan cli.FirstRunPlan) doctorTargetJSON {
	out := doctorTargetJSON{
		Name:        string(plan.Target),
		Ready:       plan.Ready,
		Summary:     plan.Summary,
		NextCommand: plan.NextCommand,
	}
	for _, step := range plan.MissingSteps {
		out.Missing = append(out.Missing, string(step.ID))
	}
	return out
}
```

- [ ] **Step 6: Run onboard and doctor tests and confirm green**

Run:

```bash
go test ./cmd/gormes -run 'TestOnboard.*FirstRun|TestDoctorTarget|TestDoctorJSONIncludesTarget|TestOnboardWizard_JSON|TestOnboard_JSON' -count=1
```

Expected:

```text
ok  	github.com/TrebuchetDynamics/gormes-agent/cmd/gormes
```

- [ ] **Step 7: Commit onboard/doctor slice**

Run:

```bash
git add cmd/gormes/onboard.go cmd/gormes/onboard_wizard_test.go cmd/gormes/onboard_wizard_json_test.go cmd/gormes/doctor.go cmd/gormes/doctor_runE_test.go
git commit -m "feat: share first-run readiness in onboard and doctor"
```

Expected:

```text
[development <sha>] feat: share first-run readiness in onboard and doctor
```

## Task 6: Extend Fresh-Install E2E Coverage

**Files:**
- Modify: `cmd/gormes/fresh_install_e2e_test.go`

- [ ] **Step 1: Add fresh-install first-run e2e tests**

Add to `cmd/gormes/fresh_install_e2e_test.go`:

```go
func TestFreshInstallRootNoTTYPrintsFirstRunGuidance(t *testing.T) {
	freshInstallE2EHome(t)

	cmd := newRootCommandWithRuntime(rootRuntime{
		isTTY: func() bool { return false },
	})
	stdout, stderr, err := executeRootCommandForTest(cmd)
	if err != nil {
		t.Fatalf("fresh root no tty: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes setup needed",
		"Next: gormes setup --quick --target terminal",
		"Non-interactive mode will not prompt.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestFreshInstallSetupQuickNonInteractiveDoesNotPrompt(t *testing.T) {
	freshInstallE2EHome(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "setup", "--quick", "--non-interactive")
	if err != nil {
		t.Fatalf("setup quick noninteractive: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Quick setup targets:",
		"gormes setup --quick --target terminal",
		"gormes setup --quick --target telegram",
		"gormes whatsapp",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}
```

- [ ] **Step 2: Run fresh-install tests and confirm red or green**

Run:

```bash
go test ./cmd/gormes -run 'TestFreshInstallRootNoTTYPrintsFirstRunGuidance|TestFreshInstallSetupQuickNonInteractiveDoesNotPrompt' -count=1
```

Expected after Tasks 1-5:

```text
ok  	github.com/TrebuchetDynamics/gormes-agent/cmd/gormes
```

- [ ] **Step 3: Run broader fresh-install e2e battery**

Run:

```bash
go test ./cmd/gormes -run 'TestFreshInstall' -count=1
```

Expected:

```text
ok  	github.com/TrebuchetDynamics/gormes-agent/cmd/gormes
```

- [ ] **Step 4: Commit e2e slice**

Run:

```bash
git add cmd/gormes/fresh_install_e2e_test.go
git commit -m "test: cover first-run setup e2e"
```

Expected:

```text
[development <sha>] test: cover first-run setup e2e
```

## Task 7: Full Verification And Cleanup

**Files:**
- No planned source edits unless verification exposes a concrete failure.

- [ ] **Step 1: Run focused tests**

Run:

```bash
go test ./internal/cli ./cmd/gormes -count=1
```

Expected:

```text
ok  	github.com/TrebuchetDynamics/gormes-agent/internal/cli
ok  	github.com/TrebuchetDynamics/gormes-agent/cmd/gormes
```

- [ ] **Step 2: Run repository test gate**

Run:

```bash
go test ./... -count=1
```

Expected:

```text
ok  	...
```

- [ ] **Step 3: Validate progress metadata**

Run:

```bash
go run ./cmd/progress validate
```

Expected:

```text
progress: validated 9 phases
```

- [ ] **Step 4: Check whitespace**

Run:

```bash
git diff --check
```

Expected: no output and exit code 0.

- [ ] **Step 5: Inspect status for unrelated files**

Run:

```bash
git status --short
```

Expected:

```text
?? internal/installtest/local_cli_e2e_test.go
```

Additional tracked modifications are acceptable only if they are from this plan and already committed. Do not add the unrelated installtest file in these commits.

## Self-Review

- Spec coverage:
  - Plain `gormes` first-run routing: Task 2 and Task 6.
  - `setup --quick` target-first flow: Task 3.
  - Shared readiness planner: Task 1, Task 5.
  - Telegram/WhatsApp/Discord/Slack targets: Task 1, Task 4.
  - Non-TTY no-prompt behavior: Task 2, Task 3, Task 6.
  - Tiny provider live test before chat handoff: Task 3.
  - Doctor target summary: Task 5.
  - Bubble Tea boundary preserved: Task 2 routes before `runResolvedTUI`; no TUI model edits.
- Placeholder scan:
  - No deferred-detail markers.
  - No unspecified test steps.
  - Each task names files, code entry points, and commands.
- Type consistency:
  - `SetupTargetID`, `FirstRunPlan`, `FirstRunStepID`, and `BuildFirstRunPlan` are introduced in Task 1 before command code uses them.
  - `rootRuntime.isTTY` and `rootRuntime.runFirstRunSetup` are introduced in Task 2 before tests assert root behavior.
  - `setupCommandSeams.ChooseSetupTarget`, `RunProviderLiveTest`, and `RunWhatsAppSetup` are introduced before setup tests rely on them.
