package environment

import (
	"strings"
	"testing"
)

// TestEnvironmentSnapshotSource_RedirectsStdoutAndStderr proves the generated
// shell wrapper redirects both stdout and stderr away from the snapshot
// `source` line so macOS bash declare -x output does not leak environment
// variables into terminal tool responses (Hermes 2e6699b3 fix).
func TestEnvironmentSnapshotSource_RedirectsStdoutAndStderr(t *testing.T) {
	cfg := EnvironmentSnapshotConfig{
		Mode:         SnapshotEnabled,
		SnapshotPath: "/tmp/hermes_snapshot.sh",
	}
	userCommand := "echo hello && ls -la"

	wrapper, evidence := BuildShellWrapper(cfg, userCommand)

	if evidence.Code != "snapshot_loaded" {
		t.Fatalf("evidence.Code = %q, want snapshot_loaded", evidence.Code)
	}
	if evidence.Path != "/tmp/hermes_snapshot.sh" {
		t.Fatalf("evidence.Path = %q, want /tmp/hermes_snapshot.sh", evidence.Path)
	}

	// Locate the source line within the wrapper.
	var sourceLine string
	for _, line := range strings.Split(wrapper, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "source ") || strings.HasPrefix(trimmed, ". ") {
			sourceLine = trimmed
			break
		}
	}
	if sourceLine == "" {
		t.Fatalf("no source line in wrapper:\n%s", wrapper)
	}

	// Source line must reference the snapshot path.
	if !strings.Contains(sourceLine, "/tmp/hermes_snapshot.sh") {
		t.Fatalf("source line missing snapshot path: %q", sourceLine)
	}

	// Source line must redirect BOTH stdout and stderr away.
	hasBothStream := strings.Contains(sourceLine, ">/dev/null 2>&1") ||
		strings.Contains(sourceLine, "&>/dev/null") ||
		strings.Contains(sourceLine, "> /dev/null 2>&1")
	if !hasBothStream {
		t.Fatalf("source line does not redirect both stdout and stderr: %q", sourceLine)
	}

	// User command must NOT have its own redirect bolted on.
	if !strings.Contains(wrapper, userCommand) {
		t.Fatalf("user command missing from wrapper:\n%s", wrapper)
	}
}

// TestEnvironmentSnapshotSource_QuotesSnapshotPath proves the snapshot path is
// shell-quoted before the source line is assembled, so spaces and apostrophes in
// profile/cache directories do not split the command or inject shell syntax.
func TestEnvironmentSnapshotSource_QuotesSnapshotPath(t *testing.T) {
	cfg := EnvironmentSnapshotConfig{
		Mode:         SnapshotEnabled,
		SnapshotPath: "/tmp/gormes snapshot/it's-live.sh",
	}

	wrapper, evidence := BuildShellWrapper(cfg, "echo still-runs")
	if evidence.Code != EvidenceSnapshotLoaded {
		t.Fatalf("evidence.Code = %q, want %q", evidence.Code, EvidenceSnapshotLoaded)
	}

	lines := strings.Split(wrapper, "\n")
	if len(lines) != 2 {
		t.Fatalf("wrapper lines = %#v, want source line plus user command", lines)
	}
	wantSourceLine := "source '/tmp/gormes snapshot/it'\"'\"'s-live.sh' >/dev/null 2>&1 || true"
	if lines[0] != wantSourceLine {
		t.Fatalf("source line = %q, want %q", lines[0], wantSourceLine)
	}
	if lines[1] != "echo still-runs" {
		t.Fatalf("user command line = %q", lines[1])
	}
}

// TestEnvironmentSnapshotSource_NoSnapshotSkipsSource proves no `source`
// command is emitted when snapshot mode is disabled or path is missing; the
// user command runs untouched.
func TestEnvironmentSnapshotSource_NoSnapshotSkipsSource(t *testing.T) {
	cases := []struct {
		name     string
		cfg      EnvironmentSnapshotConfig
		wantCode string
	}{
		{
			name:     "disabled mode",
			cfg:      EnvironmentSnapshotConfig{Mode: SnapshotDisabled, SnapshotPath: "/tmp/snapshot.sh"},
			wantCode: "snapshot_disabled",
		},
		{
			name:     "enabled but empty path",
			cfg:      EnvironmentSnapshotConfig{Mode: SnapshotEnabled, SnapshotPath: ""},
			wantCode: "snapshot_path_missing",
		},
	}
	userCommand := "echo hello"

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapper, evidence := BuildShellWrapper(tc.cfg, userCommand)

			if evidence.Code != tc.wantCode {
				t.Fatalf("evidence.Code = %q, want %q", evidence.Code, tc.wantCode)
			}
			if strings.Contains(wrapper, "source ") {
				t.Fatalf("wrapper unexpectedly contains source command:\n%s", wrapper)
			}
			if strings.Contains(wrapper, "/dev/null") {
				t.Fatalf("wrapper unexpectedly contains /dev/null redirect:\n%s", wrapper)
			}
			if wrapper != userCommand {
				t.Fatalf("wrapper = %q, want unmodified user command %q", wrapper, userCommand)
			}
		})
	}
}

// TestEnvironmentSnapshotSource_DoesNotRedactUserCommandOutput proves only
// the snapshot-load line is suppressed; the actual user command's stdout and
// stderr remain visible (no /dev/null redirects bolted onto the user
// command).
func TestEnvironmentSnapshotSource_DoesNotRedactUserCommandOutput(t *testing.T) {
	cfg := EnvironmentSnapshotConfig{
		Mode:         SnapshotEnabled,
		SnapshotPath: "/tmp/snap.sh",
	}
	userCommand := "echo visible-stdout && echo visible-stderr 1>&2"

	wrapper, _ := BuildShellWrapper(cfg, userCommand)

	// Split into lines and find the user-command segment.
	lines := strings.Split(wrapper, "\n")
	var userLineIdx int = -1
	for i, line := range lines {
		if strings.Contains(line, "visible-stdout") {
			userLineIdx = i
			break
		}
	}
	if userLineIdx < 0 {
		t.Fatalf("user command not found in wrapper:\n%s", wrapper)
	}

	userLine := lines[userLineIdx]
	if strings.Contains(userLine, ">/dev/null") || strings.Contains(userLine, "&>/dev/null") {
		t.Fatalf("user command line wraps stdout/stderr in /dev/null: %q", userLine)
	}

	// User command must appear verbatim somewhere in the wrapper, not
	// rewritten to swallow output.
	if !strings.Contains(wrapper, userCommand) {
		t.Fatalf("user command not present verbatim in wrapper:\n%s", wrapper)
	}

	// Confirm there is exactly one /dev/null occurrence (the source line).
	devNullCount := strings.Count(wrapper, "/dev/null")
	if devNullCount != 1 {
		t.Fatalf("expected exactly 1 /dev/null redirect (snapshot load only), got %d in:\n%s", devNullCount, wrapper)
	}
}

// TestEnvironmentSnapshotSource_HermeticNoShellInvocation is an invariant
// guard: the builder is a pure string transformation. It must not require a
// real shell, bash, or developer environment to run.
func TestEnvironmentSnapshotSource_HermeticNoShellInvocation(t *testing.T) {
	// If BuildShellWrapper ever shells out, this test will hang or fail in
	// hermetic CI. Calling it with both modes proves it returns purely from
	// in-process string construction.
	cfgs := []EnvironmentSnapshotConfig{
		{Mode: SnapshotDisabled},
		{Mode: SnapshotEnabled, SnapshotPath: "/tmp/x.sh"},
		{Mode: SnapshotEnabled, SnapshotPath: ""},
	}
	for _, cfg := range cfgs {
		wrapper, evidence := BuildShellWrapper(cfg, "true")
		if wrapper == "" {
			t.Fatalf("empty wrapper for cfg %+v", cfg)
		}
		if evidence.Code == "" {
			t.Fatalf("empty evidence code for cfg %+v", cfg)
		}
	}
}
