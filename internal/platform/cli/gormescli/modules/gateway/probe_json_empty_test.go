package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestGatewayProbe_JSONEmitsEmptyBeaconsArrayNotNull pins the regression
// observed during a fresh-install probe sweep:
// `gormes gateway probe --json` on a fresh install (no discovered
// beacons) emitted `"beacons": null` inside the discovery sub-object,
// instead of `"beacons": []`. Fleet automation iterating without
// nil-checks then crashes on null.
//
// Convention this enforces: read-only inventory `--json` surfaces
// always emit empty arrays, never null. Same shape as already
// enforced for sessions, kanban list, kanban boards list, and
// status system events.
func TestGatewayProbe_JSONEmitsEmptyBeaconsArrayNotNull(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	stdout, _, _ := executeGatewayProbeCommand(t, "--json")
	// gateway probe on a fresh install (no beacons) returns exit 1
	// with `ok: false`. The exit code is fine — the contract under
	// test is the JSON shape, not the exit code.

	if strings.Contains(stdout, `"beacons": null`) || strings.Contains(stdout, `"beacons":null`) {
		t.Fatalf(`"beacons" must be `+"`[]`"+` on no-discovered-beacons, not `+"`null`"+`; got:\n%s`, stdout)
	}

	var got struct {
		Discovery struct {
			Beacons []any `json:"beacons"`
		} `json:"discovery"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Discovery.Beacons == nil {
		t.Fatalf(`discovery.beacons must decode to []any{}, not nil; got %v\nstdout=%s`, got.Discovery.Beacons, stdout)
	}
}
