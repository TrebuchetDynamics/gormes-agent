package main

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestGatewayDiscoverCommand_JSONListsBonjourGateways(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	restore := gatewayDiscovererForTest(t, tools.GatewayDiscovererFunc(func(context.Context) ([]tools.GatewayEndpoint, error) {
		return []tools.GatewayEndpoint{{
			InstanceName: "devbox",
			Address:      "127.0.0.1",
			Port:         18789,
			Source:       tools.GatewayEndpointSourceBonjour,
		}}, nil
	}))
	defer restore()

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "gateway", "discover", "--json")
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
	setupOneshotFlagTestEnv(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	restoreRuntime := gatewayProbeRuntimeSummaryForTest(t, tools.GatewayRuntimeSummary{
		State:          "running",
		Validation:     "live",
		ValidationLive: true,
		Platforms: map[string]string{
			"telegram": "running",
		},
	})
	defer restoreRuntime()

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "gateway", "probe", "--url", "ws://"+listener.Addr().String(), "--json")
	if err != nil {
		t.Fatalf("gateway probe --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var got tools.GatewayProbeResult
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if !got.OK || len(got.Targets) != 1 || !got.Targets[0].Reachable {
		t.Fatalf("probe result = %+v, want one reachable target", got)
	}
	if got.Targets[0].Health != tools.GatewayHealthTCPReachable {
		t.Fatalf("health = %q, want %q", got.Targets[0].Health, tools.GatewayHealthTCPReachable)
	}
	if got.Runtime.State != "running" || got.Runtime.Platforms["telegram"] != "running" {
		t.Fatalf("runtime = %+v, want running telegram summary", got.Runtime)
	}
}

func TestGatewayDiscoverUsageCostCommandShowsSessionAndAggregateCosts(t *testing.T) {
	setupOneshotFlagTestEnv(t)
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

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"gateway", "usage-cost",
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
