package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestGatewayEndpointNormalizationAppliesNonRoutingTXTHints(t *testing.T) {
	t.Run("display name", func(t *testing.T) {
		endpoint := NormalizeGatewayEndpoint(GatewayEndpoint{
			InstanceName: "gormes-gateway",
			Address:      "127.0.0.1",
			Port:         18789,
			TXT: map[string]string{
				"displayName": " Juan Gateway ",
			},
		})
		if endpoint.DisplayName != "Juan Gateway" {
			t.Fatalf("DisplayName = %q, want TXT displayName hint", endpoint.DisplayName)
		}
	})
	t.Run("explicit display name wins", func(t *testing.T) {
		endpoint := NormalizeGatewayEndpoint(GatewayEndpoint{
			DisplayName: "Config Name",
			Address:     "127.0.0.1",
			Port:        18789,
			TXT: map[string]string{
				"displayName": "TXT Name",
			},
		})
		if endpoint.DisplayName != "Config Name" {
			t.Fatalf("DisplayName = %q, want explicit display name preserved", endpoint.DisplayName)
		}
	})
}

func TestGatewayEndpointNormalizationCanonicalizesHTTPAliases(t *testing.T) {
	for _, tc := range []struct {
		name       string
		scheme     string
		wantScheme string
		wantURL    string
	}{
		{name: "http", scheme: "http", wantScheme: "ws", wantURL: "ws://127.0.0.1:18789"},
		{name: "https", scheme: "https", wantScheme: "wss", wantURL: "wss://127.0.0.1:18789"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := NormalizeGatewayEndpoint(GatewayEndpoint{Address: "127.0.0.1", Port: 18789, Scheme: tc.scheme})
			if endpoint.Scheme != tc.wantScheme || endpoint.WSURL != tc.wantURL {
				t.Fatalf("normalized endpoint = %+v, want scheme %q URL %q", endpoint, tc.wantScheme, tc.wantURL)
			}
		})
	}
}

func TestNormalizeGatewayEndpointsDropsInvalidAndKeepsFirstDuplicateCandidate(t *testing.T) {
	endpoints := normalizeGatewayEndpoints([]GatewayEndpoint{
		{InstanceName: "manual", Address: "LOCALHOST", Port: 18789, Scheme: "ws", Source: GatewayEndpointSourceManual},
		{InstanceName: "missing-address", Port: 18789, Scheme: "ws", Source: GatewayEndpointSourceManual},
		{InstanceName: "unsupported-scheme", Address: "localhost", Port: 18800, Scheme: "ftp", Source: GatewayEndpointSourceManual},
		{InstanceName: "invalid-high-port", Address: "localhost", Port: 70000, Scheme: "ws", Source: GatewayEndpointSourceManual},
		{InstanceName: "bonjour-duplicate", Address: "localhost", Port: 18789, Scheme: "ws", Source: GatewayEndpointSourceBonjour},
		{InstanceName: "secure", Address: "127.0.0.1", Port: 18789, Scheme: "https", Source: GatewayEndpointSourceBonjour},
	})

	if len(endpoints) != 2 {
		t.Fatalf("endpoints = %+v, want 2 valid deduped candidates", endpoints)
	}
	for _, endpoint := range endpoints {
		if endpoint.InstanceName == "unsupported-scheme" || endpoint.Scheme == "ftp" || endpoint.Port > 65535 {
			t.Fatalf("endpoints = %+v, want invalid candidates dropped before discovery/probe output", endpoints)
		}
	}
	if endpoints[0].InstanceName != "secure" || endpoints[0].Scheme != "wss" {
		t.Fatalf("first endpoint = %+v, want sorted secure bonjour candidate", endpoints[0])
	}
	if endpoints[1].InstanceName != "manual" || endpoints[1].Address != "LOCALHOST" {
		t.Fatalf("duplicate winner = %+v, want first candidate provenance preserved", endpoints[1])
	}
}

func TestGatewayEndpointCandidateClassificationExplainsDroppedCandidates(t *testing.T) {
	seen := map[string]bool{
		"ws://localhost:18789": true,
	}
	cases := []struct {
		name          string
		endpoint      GatewayEndpoint
		wantAccepted  bool
		wantRejection string
	}{
		{
			name:          "missing address",
			endpoint:      GatewayEndpoint{Port: 18789, Scheme: "ws"},
			wantRejection: gatewayEndpointCandidateRejectedMissingAddress,
		},
		{
			name:          "invalid port",
			endpoint:      GatewayEndpoint{Address: "localhost", Scheme: "ws", Port: 70000},
			wantRejection: gatewayEndpointCandidateRejectedInvalidPort,
		},
		{
			name:          "unsupported scheme",
			endpoint:      GatewayEndpoint{Address: "localhost", Scheme: "ftp", Port: 18789},
			wantRejection: gatewayEndpointCandidateRejectedUnsupportedScheme,
		},
		{
			name:          "duplicate endpoint",
			endpoint:      GatewayEndpoint{Address: "LOCALHOST", Scheme: "ws", Port: 18789},
			wantRejection: gatewayEndpointCandidateRejectedDuplicate,
		},
		{
			name:         "accepted endpoint",
			endpoint:     GatewayEndpoint{Address: "127.0.0.1", Scheme: "wss", Port: 18789},
			wantAccepted: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyGatewayEndpointCandidate(tc.endpoint, seen)
			if got.Accepted != tc.wantAccepted || got.Rejection != tc.wantRejection {
				t.Fatalf("candidate = %+v, want accepted=%v rejection=%q", got, tc.wantAccepted, tc.wantRejection)
			}
			if got.Accepted && got.Key == "" {
				t.Fatalf("accepted candidate missing dedupe key: %+v", got)
			}
		})
	}
}

func TestParseDNSSDResolveGatewayKeepsQuotedTXTValuesWithSpaces(t *testing.T) {
	stdout := `Lookup Gormes Gateway._openclaw-gw._tcp.local.
Gormes Gateway._openclaw-gw._tcp.local. can be reached at workstation.local.:18789 (interface 4)
 "gatewayTls=true" "displayName=Juan Gateway" "wsPath=/gateway socket"
`

	endpoints := parseDNSSDResolveGateway(stdout, "Gormes Gateway")
	if len(endpoints) != 1 {
		t.Fatalf("endpoints = %+v, want one resolved endpoint", endpoints)
	}
	got := endpoints[0].TXT
	if got["gatewayTls"] != "true" || got["displayName"] != "Juan Gateway" || got["wsPath"] != "/gateway socket" {
		t.Fatalf("TXT = %+v, want quoted values with spaces preserved", got)
	}
	if endpoints[0].Scheme != "wss" {
		t.Fatalf("Scheme = %q, want gatewayTls TXT to normalize to wss", endpoints[0].Scheme)
	}
}

func TestParseDNSSDResolveGatewayDropsAmbiguousUnbracketedIPv6HostPort(t *testing.T) {
	stdout := `Lookup Gormes Gateway._openclaw-gw._tcp.local.
Gormes Gateway._openclaw-gw._tcp.local. can be reached at fe80::1:18789 (interface 4)
 "displayName=Juan Gateway"
`

	if got := parseDNSSDResolveGateway(stdout, "Gormes Gateway"); len(got) != 0 {
		t.Fatalf("endpoints = %+v, want ambiguous unbracketed IPv6 literal dropped", got)
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

func TestGatewayProbeHTTPVerifiesHealthDetailedHealthCapabilities(t *testing.T) {
	const secret = "sk-gateway-secret"
	server := newGatewayProbeHTTPFixture(t, secret, func(w http.ResponseWriter, _ *http.Request) {
		writeGatewayProbeCapabilitiesFixture(t, w, true)
	})
	defer server.Close()
	endpoint := mustParseGatewayProbeEndpoint(t, server.URL)

	result := ProbeGateways(context.Background(), GatewayProbeRequest{
		Endpoints: []GatewayEndpoint{endpoint},
		Prober: HTTPGatewayProber{
			Client: server.Client(),
			Auth: GatewayHTTPAuth{
				Token:  secret,
				Source: "env:GATEWAY_PROXY_KEY",
			},
		},
	})

	if !result.OK {
		t.Fatalf("OK = false; result = %+v", result)
	}
	if len(result.Targets) != 1 {
		t.Fatalf("Targets = %d, want 1", len(result.Targets))
	}
	target := result.Targets[0]
	if !target.Reachable || target.Health != GatewayHealthHTTPHealthy || target.Status != GatewayProbeStatusCapabilityReady {
		t.Fatalf("target = %+v, want healthy HTTP capability target", target)
	}
	if target.Capabilities == nil {
		t.Fatalf("Capabilities = nil; target = %+v", target)
	}
	if target.Capabilities.Object != "hermes.api_server.capabilities" || target.Capabilities.Platform != "gormes-agent" {
		t.Fatalf("Capabilities = %+v, want Hermes-compatible object and Gormes platform", target.Capabilities)
	}
	if target.Capabilities.AuthSource != "env:GATEWAY_PROXY_KEY" || !target.Capabilities.AuthRequired {
		t.Fatalf("Capabilities auth = %+v, want env source and required auth", target.Capabilities)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(raw), secret) {
		t.Fatalf("probe JSON leaked auth token: %s", raw)
	}
}

func TestGatewayProbeHTTPClassifiesAuthUnsupportedAndMalformedCapabilities(t *testing.T) {
	const secret = "sk-gateway-secret"
	t.Run("unauthorized", func(t *testing.T) {
		server := newGatewayProbeHTTPFixture(t, secret, func(w http.ResponseWriter, _ *http.Request) {
			writeGatewayProbeCapabilitiesFixture(t, w, true)
		})
		defer server.Close()
		endpoint := mustParseGatewayProbeEndpoint(t, server.URL)

		result := ProbeGateways(context.Background(), GatewayProbeRequest{
			Endpoints: []GatewayEndpoint{endpoint},
			Prober: HTTPGatewayProber{
				Client: server.Client(),
				Auth:   GatewayHTTPAuth{Source: "none"},
			},
		})

		if result.OK {
			t.Fatalf("OK = true; result = %+v", result)
		}
		target := result.Targets[0]
		if target.Health != GatewayHealthHTTPUnauthorized || target.Status != GatewayProbeStatusUnauthorized {
			t.Fatalf("target = %+v, want unauthorized classification", target)
		}
		if !strings.Contains(target.Error, "auth_source=none") {
			t.Fatalf("error = %q, want auth source classification", target.Error)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		server := newGatewayProbeHTTPFixture(t, "", func(w http.ResponseWriter, _ *http.Request) {
			writeGatewayProbeCapabilitiesFixture(t, w, false)
		})
		defer server.Close()
		endpoint := mustParseGatewayProbeEndpoint(t, server.URL)

		result := ProbeGateways(context.Background(), GatewayProbeRequest{
			Endpoints: []GatewayEndpoint{endpoint},
			Prober:    HTTPGatewayProber{Client: server.Client(), Auth: GatewayHTTPAuth{Source: "none"}},
		})

		if result.OK {
			t.Fatalf("OK = true; result = %+v", result)
		}
		target := result.Targets[0]
		if target.Health != GatewayHealthHTTPCapabilityUnsupported || target.Status != GatewayProbeStatusUnsupportedCapability {
			t.Fatalf("target = %+v, want unsupported capability classification", target)
		}
	})

	t.Run("endpoint missing method unsupported", func(t *testing.T) {
		server := newGatewayProbeHTTPFixture(t, "", func(w http.ResponseWriter, _ *http.Request) {
			payload := buildGatewayProbeCapabilitiesFixture(true)
			payload["endpoints"].(map[string]any)["run_stop"] = map[string]string{"path": "/v1/runs/{run_id}/stop"}
			writeJSONResponse(t, w, payload)
		})
		defer server.Close()
		endpoint := mustParseGatewayProbeEndpoint(t, server.URL)

		result := ProbeGateways(context.Background(), GatewayProbeRequest{
			Endpoints: []GatewayEndpoint{endpoint},
			Prober:    HTTPGatewayProber{Client: server.Client(), Auth: GatewayHTTPAuth{Source: "none"}},
		})

		if result.OK {
			t.Fatalf("OK = true; result = %+v", result)
		}
		target := result.Targets[0]
		if target.Health != GatewayHealthHTTPCapabilityUnsupported || target.Status != GatewayProbeStatusUnsupportedCapability {
			t.Fatalf("target = %+v, want unsupported capability classification for endpoint missing method", target)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		server := newGatewayProbeHTTPFixture(t, "", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"object":`))
		})
		defer server.Close()
		endpoint := mustParseGatewayProbeEndpoint(t, server.URL)

		result := ProbeGateways(context.Background(), GatewayProbeRequest{
			Endpoints: []GatewayEndpoint{endpoint},
			Prober:    HTTPGatewayProber{Client: server.Client(), Auth: GatewayHTTPAuth{Source: "none"}},
		})

		if result.OK {
			t.Fatalf("OK = true; result = %+v", result)
		}
		target := result.Targets[0]
		if target.Health != GatewayHealthHTTPCapabilityMalformed || target.Status != GatewayProbeStatusMalformedCapabilities {
			t.Fatalf("target = %+v, want malformed capability classification", target)
		}
	})
}

func TestGatewayProbeHTTPUnavailableRedactsAuthMaterial(t *testing.T) {
	const secret = "sk-gateway-secret"
	endpoint := GatewayEndpoint{Address: "127.0.0.1", Port: 9, Scheme: "ws", Source: GatewayEndpointSourceManual}
	result := ProbeGateways(context.Background(), GatewayProbeRequest{
		Endpoints: []GatewayEndpoint{endpoint},
		Prober: HTTPGatewayProber{
			Client: &http.Client{Transport: gatewayProbeRoundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("dial failed with Bearer " + secret + " token=" + secret + " password=plain-gateway-password")
			})},
			Auth: GatewayHTTPAuth{Token: secret, Source: "env:GATEWAY_PROXY_KEY"},
		},
	})

	if result.OK {
		t.Fatalf("OK = true; result = %+v", result)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	body := string(raw)
	for _, leaked := range []string{secret, "plain-gateway-password"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("probe result leaked %q: %s", leaked, body)
		}
	}
	if !strings.Contains(body, "[redacted]") {
		t.Fatalf("probe result = %s, want redacted marker", body)
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

func TestGatewayDiscoverUsageCostUnavailableUsesSameDurationWindowAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	anchor := time.Date(2026, 3, 8, 12, 0, 0, 0, loc)
	result := SummarizeGatewayUsageCost(context.Background(), GatewayUsageCostRequest{
		Days: 1,
		Now:  func() time.Time { return anchor },
		Lister: GatewayUsageSessionListerFunc(func(context.Context) ([]GatewayUsageSession, error) {
			return nil, errors.New("session db missing")
		}),
	})

	wantSince := anchor.Add(-24 * time.Hour)
	if !result.Since.Equal(wantSince) {
		t.Fatalf("Since = %s, want exact 24h lookback %s", result.Since, wantSince)
	}
}

func newGatewayProbeHTTPFixture(t *testing.T, secret string, capabilities http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			writeJSONResponse(t, w, map[string]any{"status": "ok", "platform": "gormes-agent"})
		case "/health/detailed":
			writeJSONResponse(t, w, map[string]any{"status": "ok", "platform": "gormes-agent"})
		case "/v1/capabilities":
			if secret != "" && r.Header.Get("Authorization") != "Bearer "+secret {
				writeJSONResponseStatus(t, w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"code": "invalid_api_key"}})
				return
			}
			capabilities(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func writeGatewayProbeCapabilitiesFixture(t *testing.T, w http.ResponseWriter, supported bool) {
	t.Helper()
	writeJSONResponse(t, w, buildGatewayProbeCapabilitiesFixture(supported))
}

func buildGatewayProbeCapabilitiesFixture(supported bool) map[string]any {
	return map[string]any{
		"object":   "hermes.api_server.capabilities",
		"platform": "gormes-agent",
		"model":    "gormes-agent",
		"auth": map[string]any{
			"type":     "bearer",
			"required": true,
		},
		"features": map[string]any{
			"chat_completions":           true,
			"chat_completions_streaming": true,
			"responses_api":              true,
			"responses_streaming":        true,
			"run_submission":             true,
			"run_status":                 supported,
			"run_events_sse":             supported,
			"run_stop":                   true,
			"tool_progress_events":       true,
			"session_continuity_header":  "X-Hermes-Session-Id",
		},
		"endpoints": map[string]any{
			"health":           map[string]string{"method": "GET", "path": "/health"},
			"health_detailed":  map[string]string{"method": "GET", "path": "/health/detailed"},
			"chat_completions": map[string]string{"method": "POST", "path": "/v1/chat/completions"},
			"runs":             map[string]string{"method": "POST", "path": "/v1/runs"},
			"run_status":       map[string]string{"method": "GET", "path": "/v1/runs/{run_id}"},
			"run_events":       map[string]string{"method": "GET", "path": "/v1/runs/{run_id}/events"},
			"run_stop":         map[string]string{"method": "POST", "path": "/v1/runs/{run_id}/stop"},
		},
	}
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()
	writeJSONResponseStatus(t, w, http.StatusOK, body)
}

func writeJSONResponseStatus(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode fixture response: %v", err)
	}
}

func mustParseGatewayProbeEndpoint(t *testing.T, raw string) GatewayEndpoint {
	t.Helper()
	endpoint, err := ParseGatewayEndpoint(raw, GatewayEndpointSourceManual)
	if err != nil {
		t.Fatalf("ParseGatewayEndpoint(%q): %v", raw, err)
	}
	return endpoint
}

type gatewayProbeRoundTripFunc func(*http.Request) (*http.Response, error)

func (f gatewayProbeRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
