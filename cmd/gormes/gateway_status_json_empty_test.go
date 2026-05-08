package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestGatewayStatus_JSONEmitsEmptyMapsAndArraysNotNull pins the
// regression observed during a fresh-install probe sweep:
// `gormes gateway status --json` on a fresh install (no runtime
// running, no pairing state) emitted four `null` fields:
// `runtime.platforms`, `pairing.platforms`, `pairing.pending`, and
// `pairing.approved`. Fleet automation iterating without nil-checks
// then crashes on `null`.
//
// Convention enforced: `--json` surfaces always emit empty
// arrays/maps, never null. Same as session list, kanban list,
// kanban boards list, gateway probe/discover beacons, and status
// system events.
func TestGatewayStatus_JSONEmitsEmptyMapsAndArraysNotNull(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "gateway", "status", "--json")
	if err != nil {
		t.Fatalf("gateway status --json: %v\nstderr=%s", err, stderr)
	}

	for _, banned := range []string{
		`"platforms": null`,
		`"platforms":null`,
		`"pending": null`,
		`"pending":null`,
		`"approved": null`,
		`"approved":null`,
	} {
		if strings.Contains(stdout, banned) {
			t.Fatalf(`gateway status --json must not emit %s; got:\n%s`, banned, stdout)
		}
	}

	var got struct {
		Runtime struct {
			Platforms map[string]any `json:"platforms"`
		} `json:"runtime"`
		Pairing struct {
			Platforms []any `json:"platforms"`
			Pending   []any `json:"pending"`
			Approved  []any `json:"approved"`
		} `json:"pairing"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Runtime.Platforms == nil {
		t.Errorf("runtime.platforms must decode to map[string]any{}, not nil")
	}
	if got.Pairing.Platforms == nil {
		t.Errorf("pairing.platforms must decode to []any{}, not nil")
	}
	if got.Pairing.Pending == nil {
		t.Errorf("pairing.pending must decode to []any{}, not nil")
	}
	if got.Pairing.Approved == nil {
		t.Errorf("pairing.approved must decode to []any{}, not nil")
	}
}
