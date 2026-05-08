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
