package gormescli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGatewayCommandsRejectStartupProfileFlag(t *testing.T) {
	for _, args := range [][]string{
		{"-p", "profile-name", "gateway"},
		{"-p", "profile-name", "gateway", "status"},
		{"-p", "profile-name", "gateway", "run"},
		{"-p", "profile-name", "gateway", "setup"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			setupOneshotFlagTestEnv(t)
			cmd := newRootCommandWithGatewayProfileGuardForTest()
			stdout, stderr, err := executeRootCommandForTest(cmd, args...)
			if err == nil {
				t.Fatalf("%v succeeded, want profile flag rejection\nstdout=%s\nstderr=%s", args, stdout, stderr)
			}
			if code := exitCodeFromError(err); code != 2 {
				t.Fatalf("exit code = %d, want 2; err=%v\nstdout=%s\nstderr=%s", code, err, stdout, stderr)
			}
			combined := err.Error() + stderr
			for _, want := range []string{"gateway", "--profile", "process-scoped"} {
				if !strings.Contains(combined, want) {
					t.Fatalf("gateway profile rejection missing %q:\n%s", want, combined)
				}
			}
		})
	}
}

func TestGatewayProfileStartupGuardAllowsNonGatewayProfileFlag(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithGatewayProfileGuardForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "-p", "profile-name", "chat")
	if err != nil {
		t.Fatalf("chat profile flag rejected unexpectedly: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
}

func TestGatewayProfileStartupGuardAllowsUnprofiledGateway(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithGatewayProfileGuardForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "gateway", "status")
	if err != nil {
		t.Fatalf("gateway status without profile rejected unexpectedly: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
}

func newRootCommandWithGatewayProfileGuardForTest() *cobra.Command {
	factories := stubRootFactories()
	factories["chat"] = func() *cobra.Command { return &cobra.Command{Use: "chat", RunE: noopRunE} }
	factories["gateway"] = func() *cobra.Command {
		cmd := &cobra.Command{Use: "gateway", RunE: noopRunE}
		cmd.AddCommand(
			&cobra.Command{Use: "status", RunE: noopRunE},
			&cobra.Command{Use: "run", RunE: noopRunE},
			&cobra.Command{Use: "setup", RunE: noopRunE},
		)
		return cmd
	}
	return NewRootCommand(RootOptions{
		Version: Version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return RejectGatewayProfileStartupFlag(cmd, GatewayProfileStartupGuardOptions{ExitCodeError: NewExitCodeError})
		},
	}, factories)
}

func noopRunE(*cobra.Command, []string) error { return nil }
