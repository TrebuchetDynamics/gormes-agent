package tools

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestGatewayDiscoverListsLocalGateways(t *testing.T) {
	ctx := context.Background()
	result := DiscoverGateways(ctx, GatewayDiscoverRequest{
		Discoverer: GatewayDiscovererFunc(func(context.Context) ([]GatewayEndpoint, error) {
			return []GatewayEndpoint{{
				InstanceName: "workstation",
				DisplayName:  "Workstation",
				Address:      "127.0.0.1",
				Port:         18789,
				Source:       GatewayEndpointSourceBonjour,
			}}, nil
		}),
	})

	if !result.OK {
		t.Fatalf("OK = false; result = %+v", result)
	}
	if result.Code != GatewayDiscoverCodeCompleted {
		t.Fatalf("Code = %q, want %q", result.Code, GatewayDiscoverCodeCompleted)
	}
	if len(result.Beacons) != 1 {
		t.Fatalf("Beacons = %d, want 1", len(result.Beacons))
	}
	beacon := result.Beacons[0]
	if beacon.Address != "127.0.0.1" || beacon.Port != 18789 {
		t.Fatalf("beacon address/port = %s/%d, want 127.0.0.1/18789", beacon.Address, beacon.Port)
	}
	if beacon.WSURL != "ws://127.0.0.1:18789" {
		t.Fatalf("WSURL = %q, want ws://127.0.0.1:18789", beacon.WSURL)
	}
}

func TestGatewayDiscoverReportsDegradedNoGateways(t *testing.T) {
	result := DiscoverGateways(context.Background(), GatewayDiscoverRequest{
		Discoverer: GatewayDiscovererFunc(func(context.Context) ([]GatewayEndpoint, error) {
			return nil, nil
		}),
	})

	if result.OK {
		t.Fatalf("OK = true; result = %+v", result)
	}
	if result.Code != GatewayDiscoverCodeNoGateways {
		t.Fatalf("Code = %q, want %q", result.Code, GatewayDiscoverCodeNoGateways)
	}
	if len(result.Degraded) != 1 || result.Degraded[0].Reason != GatewayDegradedNoGateways {
		t.Fatalf("Degraded = %+v, want no_gateways_discovered", result.Degraded)
	}
}

func TestGatewayDiscoverProbeShowsReachabilityDiscoveryHealthAndStatus(t *testing.T) {
	endpoint := GatewayEndpoint{
		InstanceName: "dev-gateway",
		Address:      "127.0.0.1",
		Port:         18789,
		Source:       GatewayEndpointSourceBonjour,
	}
	result := ProbeGateways(context.Background(), GatewayProbeRequest{
		Discoverer: GatewayDiscovererFunc(func(context.Context) ([]GatewayEndpoint, error) {
			return []GatewayEndpoint{endpoint}, nil
		}),
		Prober: GatewayEndpointProberFunc(func(context.Context, GatewayEndpoint) GatewayProbeTarget {
			return GatewayProbeTarget{
				Endpoint:  NormalizeGatewayEndpoint(endpoint),
				Reachable: true,
				Health:    GatewayHealthTCPReachable,
				Status:    GatewayProbeStatusRuntimeRunning,
				LatencyMs: 12,
			}
		}),
		Runtime: GatewayRuntimeSummary{
			State:          "running",
			Validation:     "live",
			ValidationLive: true,
			ActiveAgents:   2,
			Platforms: map[string]string{
				"telegram": "running",
			},
		},
	})

	if !result.OK {
		t.Fatalf("OK = false; result = %+v", result)
	}
	if result.Code != GatewayProbeCodeCompleted {
		t.Fatalf("Code = %q, want %q", result.Code, GatewayProbeCodeCompleted)
	}
	if result.Discovery.Count != 1 {
		t.Fatalf("Discovery.Count = %d, want 1", result.Discovery.Count)
	}
	if got := result.Targets[0]; !got.Reachable || got.Health != GatewayHealthTCPReachable || got.Status != GatewayProbeStatusRuntimeRunning {
		t.Fatalf("target = %+v, want reachable tcp health and running status", got)
	}
	if result.Runtime.State != "running" || result.Runtime.Platforms["telegram"] != "running" {
		t.Fatalf("Runtime = %+v, want running telegram status", result.Runtime)
	}
}

func TestGatewayDiscoverProbeReportsPerEndpointFailureReason(t *testing.T) {
	endpoint := GatewayEndpoint{Address: "127.0.0.1", Port: 19999, Source: GatewayEndpointSourceManual}
	result := ProbeGateways(context.Background(), GatewayProbeRequest{
		Endpoints: []GatewayEndpoint{endpoint},
		Prober: GatewayEndpointProberFunc(func(context.Context, GatewayEndpoint) GatewayProbeTarget {
			return GatewayProbeTarget{
				Endpoint: NormalizeGatewayEndpoint(endpoint),
				Health:   GatewayHealthUnreachable,
				Status:   GatewayProbeStatusUnavailable,
				Error:    "connect: connection refused",
			}
		}),
	})

	if result.OK {
		t.Fatalf("OK = true; result = %+v", result)
	}
	if result.Code != GatewayProbeCodeUnreachable {
		t.Fatalf("Code = %q, want %q", result.Code, GatewayProbeCodeUnreachable)
	}
	if len(result.Targets) != 1 || result.Targets[0].Error == "" {
		t.Fatalf("Targets = %+v, want per-endpoint error", result.Targets)
	}
}

func TestGatewayDiscoverUsageCostSummarizesSessionAndAggregateCosts(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	result := SummarizeGatewayUsageCost(context.Background(), GatewayUsageCostRequest{
		Days: 7,
		Now:  func() time.Time { return now },
		Sessions: []GatewayUsageSession{{
			SessionID: "sess-active",
			Source:    "telegram",
			ChatID:    "42",
			Title:     "active work",
			UpdatedAt: now.Add(-time.Hour),
			TokensIn:  1200,
			TokensOut: 300,
		}, {
			SessionID: "sess-old",
			UpdatedAt: now.Add(-30 * 24 * time.Hour),
			TokensIn:  999,
			TokensOut: 999,
		}},
		Pricing: GatewayUsagePricing{
			Source:             "test",
			InputCostPer1KUSD:  0.002,
			OutputCostPer1KUSD: 0.006,
		},
	})

	if !result.OK {
		t.Fatalf("OK = false; result = %+v", result)
	}
	if result.Code != GatewayUsageCostCodeCompleted {
		t.Fatalf("Code = %q, want %q", result.Code, GatewayUsageCostCodeCompleted)
	}
	if len(result.Sessions) != 1 || result.Sessions[0].SessionID != "sess-active" {
		t.Fatalf("Sessions = %+v, want only recent active session", result.Sessions)
	}
	if result.Totals.TokensIn != 1200 || result.Totals.TokensOut != 300 || result.Totals.TotalTokens != 1500 {
		t.Fatalf("Totals = %+v, want 1200/300/1500 tokens", result.Totals)
	}
	wantCost := 1200*0.002/1000 + 300*0.006/1000
	if math.Abs(result.Totals.EstimatedCostUSD-wantCost) > 0.0000001 {
		t.Fatalf("EstimatedCostUSD = %.9f, want %.9f", result.Totals.EstimatedCostUSD, wantCost)
	}
	if !result.Sessions[0].Priced {
		t.Fatalf("session Priced = false, want true")
	}
}

func TestGatewayDiscoverUsageCostReportsUnavailableData(t *testing.T) {
	result := SummarizeGatewayUsageCost(context.Background(), GatewayUsageCostRequest{
		Lister: GatewayUsageSessionListerFunc(func(context.Context) ([]GatewayUsageSession, error) {
			return nil, errors.New("session db missing")
		}),
	})

	if result.OK {
		t.Fatalf("OK = true; result = %+v", result)
	}
	if result.Code != GatewayUsageCostCodeUnavailable {
		t.Fatalf("Code = %q, want %q", result.Code, GatewayUsageCostCodeUnavailable)
	}
	if len(result.Degraded) != 1 || result.Degraded[0].Reason != GatewayDegradedUsageDataUnavailable {
		t.Fatalf("Degraded = %+v, want usage_data_unavailable", result.Degraded)
	}
}
