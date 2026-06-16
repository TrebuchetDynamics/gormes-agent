package gormescli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootExecutionUnknownTopLevelCommandStaysNonzeroGuidance(t *testing.T) {
	cmd := newRootExecutionCommandForTest()
	stdout, stderr, err := executeRootExecutionForTest(cmd, "no-such-command-xyzzy")
	if err == nil {
		t.Fatalf("unknown command error = nil; stdout=%s stderr=%s", stdout, stderr)
	}
	if exitCodeFromError(err) == 0 {
		t.Fatalf("unknown command exit code = 0, want nonzero")
	}
	combined := stdout + stderr + err.Error()
	if !strings.Contains(strings.ToLower(combined), "unknown command") {
		t.Fatalf("unknown command output missing guidance:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
}

func TestRootExecutionRemovedTopLevelEntrypointsReturnReplacementGuidance(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "login",
			args: []string{"login", "--provider", "plain-secret-provider"},
			want: []string{"unknown command \"login\"", "gormes auth add <provider> --type oauth"},
		},
		{
			name: "onboard",
			args: []string{"onboard", "--json"},
			want: []string{"unknown command \"onboard\"", "gormes setup", "gormes doctor --offline --target terminal --json"},
		},
		{
			name: "oneshot long",
			args: []string{"--oneshot", "hello"},
			want: []string{"unknown flag: --oneshot", "gormes chat -q \"hello\""},
		},
		{
			name: "oneshot long equals",
			args: []string{"--oneshot=hello"},
			want: []string{"unknown flag: --oneshot", "gormes chat -q \"hello\""},
		},
		{
			name: "oneshot short",
			args: []string{"-z", "hello"},
			want: []string{"unknown shorthand flag: -z", "gormes chat -q \"hello\""},
		},
		{
			name: "oneshot short compact",
			args: []string{"-zhello"},
			want: []string{"unknown shorthand flag: -z", "gormes chat -q \"hello\""},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootExecutionCommandForTest()
			stdout, stderr, err := executeRootExecutionForTest(cmd, tc.args...)
			if err == nil {
				t.Fatalf("%v error = nil\nstdout=%s\nstderr=%s", tc.args, stdout, stderr)
			}
			combined := stdout + stderr + err.Error()
			for _, want := range tc.want {
				if !strings.Contains(combined, want) {
					t.Fatalf("%v output missing %q:\nstdout=%s\nstderr=%s\nerr=%v", tc.args, want, stdout, stderr, err)
				}
			}
			for _, forbidden := range []string{"Deprecated", "deprecated", "plain-secret-provider"} {
				if strings.Contains(combined, forbidden) {
					t.Fatalf("%v output leaked %q:\nstdout=%s\nstderr=%s\nerr=%v", tc.args, forbidden, stdout, stderr, err)
				}
			}
		})
	}
}

func TestRootExecutionUnknownSubcommandWithJSONEmitsStructuredInputError(t *testing.T) {
	root := newRootCommandWithFactoryForTest("config", func() *cobra.Command {
		cmd := &cobra.Command{Use: "config"}
		cmd.AddCommand(&cobra.Command{Use: "get", RunE: func(*cobra.Command, []string) error { return nil }})
		return cmd
	})
	InstallParentUnknownSubcommandGuards(root, ParentUnknownSubcommandGuardOptions{BuildProvenance: testBuildProvenance, ExitCodeError: NewExitCodeError})
	stdout, stderr, err := executeRootExecutionForTest(root, "config", "gat", "--json")
	if err == nil {
		t.Fatalf("config gat --json error = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	var got struct {
		Build  BuildProvenance `json:"build"`
		Action string          `json:"action"`
		Error  string          `json:"error"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s\nstderr=%s\nerr=%v", jsonErr, stdout, stderr, err)
	}
	if got.Action != "unknown_subcommand" || !strings.Contains(got.Error, `unknown command "gat"`) || !strings.Contains(got.Error, "get") {
		t.Fatalf("unexpected JSON input error: %+v", got)
	}
	if got.Build.Version != Version || got.Build.GitCommit == "" {
		t.Fatalf("build provenance = %+v, want test version/commit", got.Build)
	}
}

func newRootExecutionCommandForTest() *cobra.Command {
	return NewRootCommand(RootOptions{Version: Version}, stubRootFactories())
}

func executeRootExecutionForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	return executeRootCommandForTestWithOptions(cmd, RootExecutionOptions{BuildProvenance: testBuildProvenance, ExitCodeError: NewExitCodeError}, args...)
}

func executeRootCommandForTestWithOptions(cmd *cobra.Command, opts RootExecutionOptions, args ...string) (string, string, error) {
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := ExecuteRootCommand(cmd, args, opts)
	return stdout.String(), stderr.String(), err
}
