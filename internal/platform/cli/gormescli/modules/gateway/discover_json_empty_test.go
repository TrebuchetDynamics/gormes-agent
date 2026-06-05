package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestGatewayDiscover_JSONEmitsEmptyBeaconsArrayNotNull pins the
// regression observed during a fresh-install probe sweep:
// `gormes gateway discover --json` on a fresh install (no beacons
// available) emitted `"beacons": null` instead of `"beacons": []`.
// Same shape as the probe-side fix; this fence covers the discover
// surface specifically.
//
// Convention enforced: read-only inventory `--json` surfaces always
// emit empty arrays, never null. Same as the probe, kanban list,
// kanban boards list, and status system events.
func TestGatewayDiscover_JSONEmitsEmptyBeaconsArrayNotNull(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	stdout, _, err := executeGatewayDiscoverCommand(t, "--json", "--timeout", "50")
	if err != nil {
		t.Fatalf("gateway discover --json: %v", err)
	}

	if strings.Contains(stdout, `"beacons": null`) || strings.Contains(stdout, `"beacons":null`) {
		t.Fatalf(`"beacons" must be `+"`[]`"+` on no-discovery, not `+"`null`"+`; got:\n%s`, stdout)
	}

	var got struct {
		Beacons []any `json:"beacons"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Beacons == nil {
		t.Fatalf(`beacons must decode to []any{}, not nil; got %v\nstdout=%s`, got.Beacons, stdout)
	}
}
