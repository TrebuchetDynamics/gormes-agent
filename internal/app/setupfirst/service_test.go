package setupfirst

import (
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

func TestFirstRunSetupOptionsIncludesDetectedMigrations(t *testing.T) {
	got := FirstRunSetupOptions(SourceSeams{
		DetectHermesMigrationSource:   func() string { return "/home/operator/.hermes" },
		DetectOpenClawMigrationSource: func() string { return "/home/operator/.openclaw" },
	})
	wantActions := []Action{ActionQuick, ActionFull, ActionMigrateHermes, ActionMigrateOpenClaw}
	if len(got) != len(wantActions) {
		t.Fatalf("options len = %d, want %d: %#v", len(got), len(wantActions), got)
	}
	for i, want := range wantActions {
		if got[i].Action != want {
			t.Fatalf("option[%d].Action = %q, want %q", i, got[i].Action, want)
		}
	}
}

func TestParseQuickTargetAliases(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want cli.SetupTargetID
	}{
		{raw: "", want: cli.SetupTargetTerminal},
		{raw: "chat", want: cli.SetupTargetTerminal},
		{raw: "tui", want: cli.SetupTargetTerminal},
		{raw: "wa", want: cli.SetupTargetWhatsApp},
		{raw: "NAVIVOX", want: cli.SetupTargetNavivox},
	} {
		got, ok := ParseQuickTarget(tc.raw)
		if !ok || got != tc.want {
			t.Fatalf("ParseQuickTarget(%q) = %q, %v; want %q, true", tc.raw, got, ok, tc.want)
		}
	}
	if got, ok := ParseQuickTarget("browser"); ok || got != "" {
		t.Fatalf("ParseQuickTarget(browser) = %q, %v; want invalid", got, ok)
	}
}

func TestRunQuickNonInteractivePrintsTargetsAndGuidance(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	var out strings.Builder
	err := RunQuick(QuickRuntime{
		Out:            &out,
		NonInteractive: true,
		BuildFirstRunPlan: func(config.Config, cli.SetupTargetID, bool) cli.FirstRunPlan {
			return cli.FirstRunPlan{
				DefaultTarget: cli.SetupTargetTerminal,
				NextCommand:   "gormes setup --quick --target terminal",
				Targets: []cli.SetupTargetOption{{
					ID:           cli.SetupTargetTerminal,
					Label:        "Terminal/TUI chat",
					SetupCommand: "gormes setup --quick --target terminal",
				}},
			}
		},
	})
	if err != nil {
		t.Fatalf("RunQuick: %v", err)
	}
	for _, want := range []string{"Quick setup targets:", "Terminal/TUI chat: gormes setup --quick --target terminal", "Gormes setup needed", "Next: gormes setup --quick --target terminal"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRedactedLiveTestErrorRedactsEnvSecret(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("GORMES_API_KEY", "sk-env-secret")
	err := RedactedLiveTestError(errors.New("failed sk-env-secret"))
	message := err.Error()
	if strings.Contains(message, "sk-env-secret") {
		t.Fatalf("redacted error leaked secret: %s", message)
	}
	if !strings.Contains(message, "[REDACTED]") {
		t.Fatalf("redacted error missing redaction marker: %s", message)
	}
}
