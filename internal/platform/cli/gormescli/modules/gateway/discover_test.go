package gateway

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/commandtest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// TestGatewayDiscoverCommand_JSONIncludesBuildProvenance proves
// `gormes gateway discover --json` returns a `build` envelope so fleet
// log/inventory pipelines can attribute discover beacons to the binary
// version that emitted them. Same convention as the rest of the
// `--json` arc; existing top-level fields (ok/count/beacons) remain
// addressable for backwards compatibility.
func TestGatewayDiscoverCommand_JSONIncludesBuildProvenance(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restore := gatewayDiscovererForTest(t, tools.GatewayDiscovererFunc(func(context.Context) ([]tools.GatewayEndpoint, error) {
		return nil, nil
	}))
	defer restore()

	stdout, stderr, err := executeGatewayDiscoverCommand(t, "--json")
	if err != nil {
		t.Fatalf("gateway discover --json: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		OK    bool `json:"ok"`
		Count int  `json:"count"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\n%s", jsonErr, stdout)
	}
	if got.Build.Version != testGatewayVersion {
		t.Errorf("build.version = %q, want %q", got.Build.Version, testGatewayVersion)
	}
}

func TestGatewayDiscoverCommand_JSONListsBonjourGateways(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	restore := gatewayDiscovererForTest(t, tools.GatewayDiscovererFunc(func(context.Context) ([]tools.GatewayEndpoint, error) {
		return []tools.GatewayEndpoint{{
			InstanceName: "devbox",
			Address:      "127.0.0.1",
			Port:         18789,
			Source:       tools.GatewayEndpointSourceBonjour,
		}}, nil
	}))
	defer restore()

	stdout, stderr, err := executeGatewayDiscoverCommand(t, "--json")
	if err != nil {
		t.Fatalf("gateway discover --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got struct {
		OK      bool                    `json:"ok"`
		Count   int                     `json:"count"`
		Beacons []tools.GatewayEndpoint `json:"beacons"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if !got.OK || got.Count != 1 {
		t.Fatalf("discover result = %+v, want ok count 1", got)
	}
	if got.Beacons[0].Address != "127.0.0.1" || got.Beacons[0].Port != 18789 || got.Beacons[0].WSURL != "ws://127.0.0.1:18789" {
		t.Fatalf("beacon = %+v, want address/port/wsUrl", got.Beacons[0])
	}
}

func TestGatewayDiscoverProbeCommandShowsReachabilityHealthAndStatus(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	tcpAddr := listener.Addr().(*net.TCPAddr)
	restoreDiscoverer := gatewayDiscovererForTest(t, tools.GatewayDiscovererFunc(func(context.Context) ([]tools.GatewayEndpoint, error) {
		return []tools.GatewayEndpoint{{
			InstanceName: "tcp-gateway",
			Address:      tcpAddr.IP.String(),
			Port:         tcpAddr.Port,
			Source:       tools.GatewayEndpointSourceBonjour,
		}}, nil
	}))
	defer restoreDiscoverer()
	restoreRuntime := gatewayProbeRuntimeSummaryForTest(t, tools.GatewayRuntimeSummary{
		State:          "running",
		Validation:     "live",
		ValidationLive: true,
		Platforms: map[string]string{
			"telegram": "running",
		},
	})
	defer restoreRuntime()

	stdout, stderr, err := executeGatewayProbeCommand(t, "--json")
	if err != nil {
		t.Fatalf("gateway probe --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got tools.GatewayProbeResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if !got.OK || len(got.Targets) == 0 {
		t.Fatalf("probe result = %+v, want a reachable target", got)
	}
	var target *tools.GatewayProbeTarget
	for i := range got.Targets {
		if got.Targets[i].Endpoint.Port == tcpAddr.Port {
			target = &got.Targets[i]
			break
		}
	}
	if target == nil || !target.Reachable {
		t.Fatalf("targets = %+v, want discovered TCP target reachable", got.Targets)
	}
	if target.Health != tools.GatewayHealthTCPReachable {
		t.Fatalf("health = %q, want %q", target.Health, tools.GatewayHealthTCPReachable)
	}
	if got.Runtime.State != "running" || got.Runtime.Platforms["telegram"] != "running" {
		t.Fatalf("runtime = %+v, want running telegram summary", got.Runtime)
	}
}

func TestGatewayDiscoverUsageCostCommandShowsSessionAndAggregateCosts(t *testing.T) {
	setupGatewayStatusTestEnv(t)
	smap, err := session.OpenBolt(config.SessionDBPath())
	if err != nil {
		t.Fatalf("OpenBolt: %v", err)
	}
	now := time.Now().UTC()
	if err := smap.PutMetadata(context.Background(), session.Metadata{
		SessionID:      "sess-cost",
		Source:         "telegram",
		ChatID:         "42",
		Title:          "usage work",
		UpdatedAt:      now.Unix(),
		TokensInTotal:  2000,
		TokensOutTotal: 500,
	}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}
	if err := smap.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	stdout, stderr, err := executeGatewayUsageCostCommand(t,
		"--days", "7",
		"--input-cost-per-1k", "0.002",
		"--output-cost-per-1k", "0.006",
	)
	if err != nil {
		t.Fatalf("gateway usage-cost: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gateway usage cost",
		"sessions: 1",
		"tokens: 2000 in / 500 out / 2500 total",
		"estimated_cost_usd: $0.007000",
		"- sess-cost telegram chat=42 tokens=2000/500 cost=$0.007000 title=\"usage work\"",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q\n%s", want, stdout)
		}
	}
}

func gatewayDiscovererForTest(t *testing.T, discoverer tools.GatewayDiscoverer) func() {
	t.Helper()
	old := newGatewayDiscoverer
	newGatewayDiscoverer = func(time.Duration) tools.GatewayDiscoverer {
		return discoverer
	}
	return func() {
		newGatewayDiscoverer = old
	}
}

func gatewayProbeRuntimeSummaryForTest(t *testing.T, summary tools.GatewayRuntimeSummary) func() {
	t.Helper()
	old := readGatewayProbeRuntimeSummary
	readGatewayProbeRuntimeSummary = func(context.Context) (tools.GatewayRuntimeSummary, error) {
		return summary, nil
	}
	return func() {
		readGatewayProbeRuntimeSummary = old
	}
}

func executeGatewayDiscoverCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return commandtest.Execute(t, NewDiscoverCommand(testGatewayOptions()), args...)
}

func executeGatewayProbeCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return commandtest.Execute(t, NewProbeCommand(testGatewayOptions()), args...)
}

func executeGatewayUsageCostCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	return commandtest.Execute(t, NewUsageCostCommand(testGatewayOptions()), args...)
}
