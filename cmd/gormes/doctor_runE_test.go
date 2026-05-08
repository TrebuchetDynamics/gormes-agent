package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestDoctorCommand_OfflineRoutedThroughCobra proves that `gormes
// doctor --offline` writes its output through cmd.OutOrStdout() (so
// tests can capture it via cmd.SetOut) and returns a normal RunE error
// instead of calling os.Exit on failure paths. This is the
// testability-enabling refactor: previously the command hard-exited
// the test process and bypassed cobra's stdout writer, so end-to-end
// fixtures were impossible.
func TestDoctorCommand_OfflineRoutedThroughCobra(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	var stdout, stderr bytes.Buffer
	doctorCmd.SetOut(&stdout)
	doctorCmd.SetErr(&stderr)
	doctorCmd.SetArgs([]string{"--offline"})
	// Defer-restore default writers so other tests aren't contaminated
	// by this buffer assignment.
	t.Cleanup(func() {
		doctorCmd.SetOut(nil)
		doctorCmd.SetErr(nil)
		doctorCmd.SetArgs(nil)
	})

	// We don't pin the exact pass/fail outcome — that depends on host
	// state (chrome present, etc). What we DO pin: the command must
	// return without panicking AND its output must reach the buffer
	// rather than the host's real stdout.
	_ = doctorCmd.Execute()

	combined := stdout.String() + stderr.String()
	if combined == "" {
		t.Fatalf("doctor --offline produced no captured output; output likely went to os.Stdout/Stderr instead of cobra writers")
	}
	// Toolbox check always runs in offline mode and emits a `[PASS]`/
	// `[FAIL]`/`[WARN]` line with `Toolbox` in the name, so it's a
	// stable presence-marker.
	if !strings.Contains(combined, "Toolbox") {
		t.Fatalf("doctor --offline output should mention Toolbox check; got:\n%s", combined)
	}
}
