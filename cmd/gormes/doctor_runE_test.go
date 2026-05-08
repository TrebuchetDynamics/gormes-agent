package main

import (
	"bytes"
	"encoding/json"
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

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline"})
	_ = cmd.Execute()

	combined := stdout.String() + stderr.String()
	if combined == "" {
		t.Fatalf("doctor --offline produced no captured output; output likely went to os.Stdout/Stderr instead of cobra writers")
	}
	if !strings.Contains(combined, "Toolbox") {
		t.Fatalf("doctor --offline output should mention Toolbox check; got:\n%s", combined)
	}
}

// TestDoctorCommand_JSONIncludesBuildProvenance proves
// `gormes doctor --json` carries the running binary's build SHA and
// version. Same contract as `update --json`'s `build` block — fleet
// health snapshots stay attributable to a specific binary even when
// captured and shipped off-host.
func TestDoctorCommand_JSONIncludesBuildProvenance(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	if got.Build.Version != Version {
		t.Fatalf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Build.GitCommit == "" {
		t.Fatalf("got.build.git_commit must be non-empty")
	}
}

// TestDoctorCommand_JSONFieldOrderPutsFailedBeforeChecks proves the
// JSON output uses a stable field order with summary fields (`failed`)
// before the per-check array. This matches `update --json`'s
// convention so downstream tooling that pretty-prints / diffs JSON
// reports gets a predictable structure across surfaces. Relying on
// `map[string]any` alphabetic sort would put `checks` before `failed`
// — inconsistent with the rest of the --json arc.
func TestDoctorCommand_JSONFieldOrderPutsFailedBeforeChecks(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	body := stdout.String()
	failedIdx := strings.Index(body, `"failed"`)
	checksIdx := strings.Index(body, `"checks"`)
	if failedIdx < 0 || checksIdx < 0 {
		t.Fatalf("output missing failed/checks fields:\n%s", body)
	}
	if failedIdx >= checksIdx {
		t.Fatalf("`failed` must precede `checks` in JSON for stable consumer rendering; got failedIdx=%d checksIdx=%d", failedIdx, checksIdx)
	}
}

// TestDoctorCommand_JSONReportsFailedFieldFromWorstCheck proves the
// JSON document carries a top-level "failed" boolean derived from the
// worst-status check encountered. Monitoring consumers branch on this
// field rather than scanning every entry — same contract as
// `gormes update --json` and friends.
func TestDoctorCommand_JSONReportsFailedFieldFromWorstCheck(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	// Parse as a generic map first to assert the `failed` key is
	// PRESENT (zero-value false would otherwise let a missing field
	// silently pass through json.Unmarshal into a struct).
	var raw map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	gotFailedRaw, hasFailed := raw["failed"]
	if !hasFailed {
		t.Fatalf("JSON must include top-level `failed` boolean; got keys=%v", mapKeys(raw))
	}
	gotFailed, ok := gotFailedRaw.(bool)
	if !ok {
		t.Fatalf("`failed` must be a bool; got %T %v", gotFailedRaw, gotFailedRaw)
	}

	// Recompute expected from collected check statuses so the test
	// stays correct across host-environment variation.
	checksRaw, _ := raw["checks"].([]any)
	wantFailed := false
	for _, entry := range checksRaw {
		m, _ := entry.(map[string]any)
		if status, _ := m["status"].(string); status == "FAIL" {
			wantFailed = true
			break
		}
	}
	if gotFailed != wantFailed {
		t.Fatalf("got.failed = %t, want %t (any FAIL status implies failed=true)", gotFailed, wantFailed)
	}
}

// TestDoctorCommand_SourceBuildIdentitySummary proves that when the
// binary was built without ldflags-injected provenance (the default
// `go run` / `go build` path leaves GitCommit at the literal sentinel
// "unknown"), the doctor summary labels it as a "source build". A bare
// `commit=unknown` summary is technically accurate but cryptic — it
// looks like a malformed value. The explicit label tells operators
// "this binary wasn't built by the release pipeline" without forcing
// them to know what `unknown` means in this codebase.
func TestDoctorCommand_SourceBuildIdentitySummary(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prevDirty := GitDirty
	prevCommit := GitCommit
	GitDirty = "false"
	GitCommit = "unknown"
	t.Cleanup(func() {
		GitDirty = prevDirty
		GitCommit = prevCommit
	})

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	for _, c := range got.Checks {
		if c.Name != "build identity" {
			continue
		}
		if c.Status != "PASS" {
			t.Fatalf("source build identity status = %q, want PASS", c.Status)
		}
		if !strings.Contains(c.Summary, "source build") {
			t.Fatalf("summary must label sentinel-commit builds as `source build`; got %q", c.Summary)
		}
		if strings.Contains(c.Summary, "commit=unknown") {
			t.Fatalf("summary must NOT show bare `commit=unknown`; replaced with `source build` label; got %q", c.Summary)
		}
		return
	}
	t.Fatalf("doctor must emit `build identity` check; got checks=%+v", got.Checks)
}

// TestDoctorCommand_DirtyBuildEmitsBuildIdentityWarning proves that when
// the binary was built from a dirty source tree (`-X main.GitDirty=true`
// at build time), `gormes doctor` surfaces an explicit warn-status
// "build identity" check. Operators reading doctor output should know
// they are NOT running a clean release artifact — otherwise stale or
// uncommitted local changes silently ride along into production with no
// signal to the operator.
func TestDoctorCommand_DirtyBuildEmitsBuildIdentityWarning(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := GitDirty
	GitDirty = "true"
	t.Cleanup(func() { GitDirty = prev })

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	var found bool
	for _, c := range got.Checks {
		if c.Name == "build identity" {
			found = true
			if c.Status != "WARN" {
				t.Fatalf("build identity status = %q, want WARN for dirty build", c.Status)
			}
			if !strings.Contains(c.Summary, "dirty") {
				t.Fatalf("build identity summary should mention 'dirty'; got %q", c.Summary)
			}
		}
	}
	if !found {
		t.Fatalf("doctor must emit `build identity` check on dirty builds; got checks=%+v", got.Checks)
	}
}

// TestDoctorCommand_CleanBuildEmitsBuildIdentityPass proves that on a
// clean build (the default), `gormes doctor` reports a PASS-status
// "build identity" check naming the version + short SHA. The check
// must be present in BOTH dirty and clean states so consumers always
// see binary identity in the snapshot — not only when something is
// wrong.
func TestDoctorCommand_CleanBuildEmitsBuildIdentityPass(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	prev := GitDirty
	GitDirty = "false"
	t.Cleanup(func() { GitDirty = prev })

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	var found bool
	for _, c := range got.Checks {
		if c.Name == "build identity" {
			found = true
			if c.Status != "PASS" {
				t.Fatalf("build identity status = %q, want PASS for clean build", c.Status)
			}
			if !strings.Contains(c.Summary, Version) {
				t.Fatalf("build identity summary must name version %q; got %q", Version, c.Summary)
			}
		}
	}
	if !found {
		t.Fatalf("doctor must emit `build identity` check on clean builds; got checks=%+v", got.Checks)
	}
}

func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestDoctorCommand_OfflineJSONEmitsCheckArray proves
// `gormes doctor --offline --json` emits a parseable
// `{"checks": [...]}` document where each entry has the same fields
// the human surface renders. Monitoring/CI consumers can ingest
// fleet-wide doctor results without scraping the bracketed text.
func TestDoctorCommand_OfflineJSONEmitsCheckArray(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"doctor", "--offline", "--json"})
	_ = cmd.Execute()

	var got struct {
		Checks []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout.String(), err)
	}
	if len(got.Checks) == 0 {
		t.Fatalf("got 0 checks, want at least the SecretRef runtime + Toolbox + provider-skipped entries")
	}
	wantNames := map[string]bool{"Toolbox": false, "SecretRef runtime": false, "provider health": false}
	for _, c := range got.Checks {
		if _, ok := wantNames[c.Name]; ok {
			wantNames[c.Name] = true
		}
	}
	for name, found := range wantNames {
		if !found {
			t.Fatalf("checks array missing entry %q; got names=%v", name, checkNames(got.Checks))
		}
	}
	if strings.Contains(stdout.String(), "[PASS]") || strings.Contains(stdout.String(), "[WARN]") {
		t.Fatalf("--json must not emit bracketed human lines; got:\n%s", stdout.String())
	}
}

func checkNames(checks []struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}) []string {
	out := make([]string, len(checks))
	for i, c := range checks {
		out[i] = c.Name
	}
	return out
}
