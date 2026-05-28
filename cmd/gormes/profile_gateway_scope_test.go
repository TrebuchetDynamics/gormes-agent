package main

import (
	"strings"
	"testing"
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
			cmd := newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeOneshotFlagCommand(cmd, args...)
			if err == nil {
				t.Fatalf("%v succeeded, want profile flag rejection\nstdout=%s\nstderr=%s", args, stdout, stderr)
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
