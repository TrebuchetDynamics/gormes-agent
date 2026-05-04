package main

import (
	"strings"
	"testing"
)

func TestHermesCommandAliasFidelity_RootUnknownAndTypoSuggestions(t *testing.T) {
	t.Run("unknown top level command stays nonzero guidance", func(t *testing.T) {
		cmd := newRootCommandWithRuntime(rootRuntime{})
		stdout, stderr, err := executeRootCommandForTest(cmd, "no-such-command-xyzzy")
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
	})

	t.Run("removed login command is an explicit suggestion", func(t *testing.T) {
		cmd := newRootCommandWithRuntime(rootRuntime{})
		stdout, stderr, err := executeRootCommandForTest(cmd, "login", "--provider", "plain-secret-provider")
		if err == nil {
			t.Fatalf("login error = nil; stdout=%s stderr=%s", stdout, stderr)
		}
		combined := stdout + stderr + err.Error()
		if !strings.Contains(combined, "did you mean \"gormes auth add <provider> --type oauth\"?") {
			t.Fatalf("login output missing auth-add suggestion:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
		}
		if strings.Contains(combined, "plain-secret-provider") {
			t.Fatalf("login suggestion leaked provider argument:\n%s", combined)
		}
	})

	t.Run("migrate ooenclaw remains typo guidance not alias", func(t *testing.T) {
		root := setupMigrateOpenClawEnv(t)
		src := root + "/src"
		writeOpenClawCLIFixture(t, src)

		_, stdout, stderr, err := executeMigrateOpenClaw("ooenclaw", "--dry-run", "--source", src)
		if err == nil {
			t.Fatalf("migrate ooenclaw error = nil; stdout=%s stderr=%s", stdout, stderr)
		}
		combined := stdout + stderr + err.Error()
		if !strings.Contains(combined, "openclaw") || !strings.Contains(strings.ToLower(combined), "unknown command") {
			t.Fatalf("migrate ooenclaw output missing explicit openclaw typo guidance:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
		}
	})
}
