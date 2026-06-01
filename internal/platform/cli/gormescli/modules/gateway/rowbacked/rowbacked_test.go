package rowbacked

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func TestGatewayRowBackedCommandsUseInjectedBuildProvenance(t *testing.T) {
	cmd := NewWebhookCommand(Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
	})
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"subscribe", "https://example.invalid/hook", "--json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("webhook subscribe must remain row-backed\nstdout=%s", stdout.String())
	}
	if exit, ok := err.(interface{ ExitCode() int }); !ok || exit.ExitCode() != 2 {
		t.Fatalf("webhook subscribe exit = %#v, want ExitCode() == 2", err)
	}
	var got gormescli.RowBackedReportJSON
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("webhook stdout must be JSON: %v\nstdout=%s", err, stdout.String())
	}
	if got.Build.Version != "test-version" || got.Build.GitCommit != "test-sha" {
		t.Fatalf("build provenance = %+v, want injected test values", got.Build)
	}
	if got.Command != "webhook subscribe" || got.Status != gormescli.RowBackedStatus || got.Row != GatewayCronRow {
		t.Fatalf("webhook report = %+v, want gateway cron row-backed report", got)
	}
}

func TestGatewayRowBackedCommandTreesExposeExpectedChildren(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  any
		want []string
	}{
		{name: "webhook", cmd: NewWebhookCommand(Options{}), want: []string{"subscribe", "list", "remove", "test"}},
		{name: "hooks", cmd: NewHooksCommand(Options{}), want: []string{"list", "test", "revoke", "doctor"}},
		{name: "pairing", cmd: NewPairingCommand(Options{}), want: []string{"list", "approve", "revoke", "clear-pending"}},
	} {
		root, ok := tc.cmd.(interface{ Commands() []*cobra.Command })
		if !ok {
			t.Fatalf("%s command has unexpected type %T", tc.name, tc.cmd)
		}
		seen := map[string]bool{}
		for _, child := range root.Commands() {
			seen[child.Name()] = true
		}
		for _, want := range tc.want {
			if !seen[want] {
				t.Fatalf("%s command missing child %q", tc.name, want)
			}
		}
	}
}
