package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const (
	defaultGatewayDiscoverTimeoutMs = 2000
	defaultGatewayProbeTimeoutMs    = 1500
)

var (
	newGatewayDiscoverer = func(timeout time.Duration) tools.GatewayDiscoverer {
		return tools.NewShellGatewayDiscoverer(timeout)
	}
	readGatewayProbeRuntimeSummary = func(ctx context.Context) (tools.GatewayRuntimeSummary, error) {
		store := newGatewayStatusRuntimeStore(config.GatewayRuntimeStatusPath())
		snapshot, err := store.ReadValidatedRuntimeStatusSnapshot(ctx)
		if err != nil {
			return tools.GatewayRuntimeSummary{}, err
		}
		return gatewayRuntimeSummaryFromSnapshot(snapshot), nil
	}
	openGatewayUsageSessionLister = func() (tools.GatewayUsageSessionLister, io.Closer, error) {
		if _, err := os.Stat(config.SessionDBPath()); err != nil {
			return nil, nil, err
		}
		smap, err := session.OpenBolt(config.SessionDBPath())
		if err != nil {
			return nil, nil, err
		}
		return gatewayUsageSessionLister{smap: smap}, smap, nil
	}
)

func newGatewayDiscoverCommand() *cobra.Command {
	var (
		timeoutMs int
		jsonOut   bool
	)
	cmd := &cobra.Command{
		Use:          "discover",
		Short:        "Discover local Gormes gateways via Bonjour/mDNS",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if timeoutMs < 0 {
				return fmt.Errorf("gateway discover: timeout must be non-negative")
			}
			result := tools.DiscoverGateways(cmd.Context(), tools.GatewayDiscoverRequest{
				Discoverer: newGatewayDiscoverer(time.Duration(timeoutMs) * time.Millisecond),
			})
			if jsonOut {
				// Normalize nil beacon slices to empty slices so
				// JSON consumers iterate over `[]` instead of
				// crashing on `null`. Same convention as the
				// probe path.
				if result.Beacons == nil {
					result.Beacons = []tools.GatewayEndpoint{}
				}
				return encodeIndentedJSON(cmd.OutOrStdout(), gatewayDiscoverReportJSON{
					Build:                 newBuildProvenance(),
					GatewayDiscoverResult: result,
				})
			}
			renderGatewayDiscoverText(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().IntVar(&timeoutMs, "timeout", defaultGatewayDiscoverTimeoutMs, "per-command Bonjour browse/resolve timeout in milliseconds")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print gateway discovery as JSON")
	return cmd
}

func newGatewayProbeCommand() *cobra.Command {
	var (
		timeoutMs int
		urlRaw    string
		jsonOut   bool
	)
	cmd := &cobra.Command{
		Use:          "probe",
		Short:        "Probe gateway reachability, discovery, health, and runtime status",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if timeoutMs < 0 {
				return fmt.Errorf("gateway probe: timeout must be non-negative")
			}
			endpoints := []tools.GatewayEndpoint{}
			discoverer := newGatewayDiscoverer(time.Duration(timeoutMs) * time.Millisecond)
			prober := tools.GatewayEndpointProber(tools.TCPGatewayProber{
				Timeout: time.Duration(timeoutMs) * time.Millisecond,
			})
			if strings.TrimSpace(urlRaw) != "" {
				endpoint, err := tools.ParseGatewayEndpoint(urlRaw, tools.GatewayEndpointSourceManual)
				if err != nil {
					return fmt.Errorf("gateway probe: %w", err)
				}
				endpoints = append(endpoints, endpoint)
				discoverer = nil
				cfg, err := config.Load(nil)
				if err != nil {
					return fmt.Errorf("config: %w", err)
				}
				prober = tools.HTTPGatewayProber{
					Timeout: time.Duration(timeoutMs) * time.Millisecond,
					Auth:    gatewayProbeHTTPAuth(cfg, os.Getenv),
				}
			} else {
				endpoints = append(endpoints, tools.NormalizeGatewayEndpoint(tools.GatewayEndpoint{
					Address: "127.0.0.1",
					Port:    18789,
					Source:  tools.GatewayEndpointSourceManual,
				}))
			}
			runtimeSummary, err := readGatewayProbeRuntimeSummary(cmd.Context())
			if err != nil {
				return fmt.Errorf("runtime status: %w", err)
			}
			result := tools.ProbeGateways(cmd.Context(), tools.GatewayProbeRequest{
				Discoverer: discoverer,
				Endpoints:  endpoints,
				Prober:     prober,
				Runtime:    runtimeSummary,
			})
			if jsonOut {
				// Normalize nil beacon slices to empty slices so
				// JSON consumers iterate over `[]` instead of
				// crashing on `null`. Same convention as
				// emitSessionListJSON / collectSystemSnapshotForJSON.
				if result.Discovery.Beacons == nil {
					result.Discovery.Beacons = []tools.GatewayEndpoint{}
				}
				if err := encodeIndentedJSON(cmd.OutOrStdout(), gatewayProbeReportJSON{
					Build:              newBuildProvenance(),
					GatewayProbeResult: result,
				}); err != nil {
					return err
				}
			} else {
				renderGatewayProbeText(cmd.OutOrStdout(), result)
			}
			if !result.OK {
				return newExitCodeError(1, fmt.Errorf("gateway probe: no reachable gateway"))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&timeoutMs, "timeout", defaultGatewayProbeTimeoutMs, "per-target TCP reachability timeout in milliseconds")
	cmd.Flags().StringVar(&urlRaw, "url", "", "explicit gateway URL or host:port to probe")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print gateway probe as JSON")
	return cmd
}

func gatewayProbeHTTPAuth(cfg config.Config, lookupEnv func(string) string) tools.GatewayHTTPAuth {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}
	if token := strings.TrimSpace(lookupEnv("GATEWAY_PROXY_KEY")); token != "" {
		return tools.GatewayHTTPAuth{Token: token, Source: "env:GATEWAY_PROXY_KEY"}
	}
	if token := strings.TrimSpace(lookupEnv("GORMES_DASHBOARD_API_KEY")); token != "" {
		return tools.GatewayHTTPAuth{Token: token, Source: "env:GORMES_DASHBOARD_API_KEY"}
	}
	if token := strings.TrimSpace(cfg.Gateway.ProxyKey); token != "" {
		return tools.GatewayHTTPAuth{Token: token, Source: "config:gateway.proxy_key"}
	}
	return tools.GatewayHTTPAuth{Source: "none"}
}

func newGatewayUsageCostCommand() *cobra.Command {
	var (
		days            int
		jsonOut         bool
		inputCostPer1K  float64
		outputCostPer1K float64
	)
	cmd := &cobra.Command{
		Use:          "usage-cost",
		Short:        "Summarize token usage costs from gateway session metadata",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if days < 0 {
				return fmt.Errorf("gateway usage-cost: days must be non-negative")
			}
			pricing := tools.GatewayUsagePricing{
				Source:             "unpriced",
				InputCostPer1KUSD:  inputCostPer1K,
				OutputCostPer1KUSD: outputCostPer1K,
			}
			if inputCostPer1K > 0 || outputCostPer1K > 0 {
				pricing.Source = "flags"
			}
			lister, closer, err := openGatewayUsageSessionLister()
			if closer != nil {
				defer closer.Close()
			}
			var result tools.GatewayUsageCostResult
			if err != nil {
				result = tools.SummarizeGatewayUsageCost(cmd.Context(), tools.GatewayUsageCostRequest{
					Days:    days,
					Pricing: pricing,
					Lister: tools.GatewayUsageSessionListerFunc(func(context.Context) ([]tools.GatewayUsageSession, error) {
						return nil, err
					}),
				})
			} else {
				result = tools.SummarizeGatewayUsageCost(cmd.Context(), tools.GatewayUsageCostRequest{
					Days:    days,
					Pricing: pricing,
					Lister:  lister,
				})
			}
			if jsonOut {
				return encodeIndentedJSON(cmd.OutOrStdout(), gatewayUsageCostReportJSON{
					Build:                  newBuildProvenance(),
					GatewayUsageCostResult: result,
				})
			}
			renderGatewayUsageCostText(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().IntVar(&days, "days", 30, "number of days to include")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print gateway usage cost as JSON")
	cmd.Flags().Float64Var(&inputCostPer1K, "input-cost-per-1k", 0, "estimated input token cost in USD per 1K tokens")
	cmd.Flags().Float64Var(&outputCostPer1K, "output-cost-per-1k", 0, "estimated output token cost in USD per 1K tokens")
	return cmd
}

func gatewayRuntimeSummaryFromSnapshot(snapshot gateway.RuntimeStatusSnapshot) tools.GatewayRuntimeSummary {
	status := snapshot.Status
	platforms := map[string]string{}
	for name, platform := range status.Platforms {
		platforms[name] = string(platform.State)
	}
	if len(platforms) == 0 {
		platforms = nil
	}
	return tools.GatewayRuntimeSummary{
		State:          string(status.GatewayState),
		Validation:     string(snapshot.Validation.Status),
		ValidationLive: snapshot.Validation.Live,
		ActiveAgents:   status.ActiveAgents,
		Platforms:      platforms,
		Missing:        snapshot.Missing,
	}
}

type gatewayUsageSessionMetadataLister interface {
	ListAllMetadata(context.Context) ([]session.Metadata, error)
}

type gatewayUsageSessionLister struct {
	smap gatewayUsageSessionMetadataLister
}

func (l gatewayUsageSessionLister) ListGatewayUsageSessions(ctx context.Context) ([]tools.GatewayUsageSession, error) {
	items, err := l.smap.ListAllMetadata(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]tools.GatewayUsageSession, 0, len(items))
	for _, item := range items {
		updatedAt := time.Time{}
		if item.UpdatedAt > 0 {
			updatedAt = time.Unix(item.UpdatedAt, 0).UTC()
		}
		out = append(out, tools.GatewayUsageSession{
			SessionID: item.SessionID,
			Source:    item.Source,
			ChatID:    item.ChatID,
			Title:     item.Title,
			UpdatedAt: updatedAt,
			TokensIn:  item.TokensInTotal,
			TokensOut: item.TokensOutTotal,
		})
	}
	return out, nil
}

func renderGatewayDiscoverText(w io.Writer, result tools.GatewayDiscoverResult) {
	fmt.Fprintln(w, "Gateway discovery")
	fmt.Fprintf(w, "found: %d\n", result.Count)
	for _, beacon := range result.Beacons {
		fmt.Fprintf(w, "- %s address=%s port=%d wsUrl=%s source=%s\n", gatewayEndpointLabel(beacon), beacon.Address, beacon.Port, beacon.WSURL, beacon.Source)
	}
	renderGatewayDegradedText(w, result.Degraded)
}

func renderGatewayProbeText(w io.Writer, result tools.GatewayProbeResult) {
	fmt.Fprintln(w, "Gateway probe")
	if result.OK {
		fmt.Fprintln(w, "reachable: yes")
	} else {
		fmt.Fprintln(w, "reachable: no")
	}
	fmt.Fprintf(w, "discovery: %d gateway(s)\n", result.Discovery.Count)
	if result.Runtime.State != "" || result.Runtime.Validation != "" || result.Runtime.Missing {
		fmt.Fprintf(w, "runtime: state=%s validation=%s live=%t active_agents=%d\n", displayValue(result.Runtime.State), displayValue(result.Runtime.Validation), result.Runtime.ValidationLive, result.Runtime.ActiveAgents)
	}
	for _, target := range result.Targets {
		fmt.Fprintf(w, "- %s reachable=%t health=%s status=%s latency_ms=%d", target.Endpoint.WSURL, target.Reachable, target.Health, target.Status, target.LatencyMs)
		if target.Error != "" {
			fmt.Fprintf(w, " error=%q", target.Error)
		}
		fmt.Fprintln(w)
	}
	renderGatewayDegradedText(w, result.Warnings)
}

func renderGatewayUsageCostText(w io.Writer, result tools.GatewayUsageCostResult) {
	fmt.Fprintln(w, "Gateway usage cost")
	fmt.Fprintf(w, "window_days: %d\n", result.Days)
	fmt.Fprintf(w, "pricing: %s input_per_1k_usd=%s output_per_1k_usd=%s\n", result.Pricing.Source, formatFloat(result.Pricing.InputCostPer1KUSD), formatFloat(result.Pricing.OutputCostPer1KUSD))
	fmt.Fprintf(w, "sessions: %d\n", result.Totals.Sessions)
	fmt.Fprintf(w, "tokens: %d in / %d out / %d total\n", result.Totals.TokensIn, result.Totals.TokensOut, result.Totals.TotalTokens)
	fmt.Fprintf(w, "estimated_cost_usd: %s\n", formatGatewayCost(result.Totals.EstimatedCostUSD, result.Totals.Priced))
	if len(result.Sessions) > 0 {
		fmt.Fprintln(w, "session_costs:")
		for _, row := range result.Sessions {
			parts := []string{row.SessionID}
			if row.Source != "" {
				parts = append(parts, row.Source)
			}
			if row.ChatID != "" {
				parts = append(parts, "chat="+row.ChatID)
			}
			parts = append(parts,
				fmt.Sprintf("tokens=%d/%d", row.TokensIn, row.TokensOut),
				"cost="+formatGatewayCost(row.EstimatedCostUSD, row.Priced),
			)
			if row.Title != "" {
				parts = append(parts, "title="+strconv.Quote(row.Title))
			}
			fmt.Fprintf(w, "- %s\n", strings.Join(parts, " "))
		}
	}
	renderGatewayDegradedText(w, result.Degraded)
}

func renderGatewayDegradedText(w io.Writer, degraded []tools.GatewayDegradedStatus) {
	if len(degraded) == 0 {
		return
	}
	fmt.Fprintln(w, "degraded:")
	for _, item := range degraded {
		if item.Message != "" {
			fmt.Fprintf(w, "- %s: %s\n", item.Reason, item.Message)
		} else {
			fmt.Fprintf(w, "- %s\n", item.Reason)
		}
	}
}

func gatewayEndpointLabel(endpoint tools.GatewayEndpoint) string {
	if endpoint.DisplayName != "" {
		return endpoint.DisplayName
	}
	if endpoint.InstanceName != "" {
		return endpoint.InstanceName
	}
	return endpoint.WSURL
}

func displayValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func formatGatewayCost(cost float64, priced bool) string {
	if !priced {
		return "n/a"
	}
	return fmt.Sprintf("$%.6f", cost)
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func encodeIndentedJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

// gatewayDiscoverReportJSON, gatewayProbeReportJSON, and
// gatewayUsageCostReportJSON wrap the internal/tools result types with
// build provenance so fleet log/inventory pipelines can attribute each
// JSON document to the binary version that emitted it. Existing
// top-level fields (ok/count/beacons/etc.) remain addressable through
// struct embedding — JSON unmarshal in callers parsing the old shape
// continues to work because Go's JSON decoder ignores the unknown
// `build` field by default.
type gatewayDiscoverReportJSON struct {
	Build buildProvenanceJSON `json:"build"`
	tools.GatewayDiscoverResult
}

type gatewayProbeReportJSON struct {
	Build buildProvenanceJSON `json:"build"`
	tools.GatewayProbeResult
}

type gatewayUsageCostReportJSON struct {
	Build buildProvenanceJSON `json:"build"`
	tools.GatewayUsageCostResult
}
