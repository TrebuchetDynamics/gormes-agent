package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/gateway"

const (
	GatewayBonjourServiceType = gateway.GatewayBonjourServiceType

	GatewayEndpointSourceBonjour = gateway.GatewayEndpointSourceBonjour
	GatewayEndpointSourceManual  = gateway.GatewayEndpointSourceManual

	GatewayDiscoverCodeCompleted   = gateway.GatewayDiscoverCodeCompleted
	GatewayDiscoverCodeNoGateways  = gateway.GatewayDiscoverCodeNoGateways
	GatewayDiscoverCodeUnavailable = gateway.GatewayDiscoverCodeUnavailable

	GatewayProbeCodeCompleted   = gateway.GatewayProbeCodeCompleted
	GatewayProbeCodeUnreachable = gateway.GatewayProbeCodeUnreachable

	GatewayUsageCostCodeCompleted   = gateway.GatewayUsageCostCodeCompleted
	GatewayUsageCostCodeUnavailable = gateway.GatewayUsageCostCodeUnavailable

	GatewayDegradedNoGateways            = gateway.GatewayDegradedNoGateways
	GatewayDegradedBonjourUnavailable    = gateway.GatewayDegradedBonjourUnavailable
	GatewayDegradedProbeTimeout          = gateway.GatewayDegradedProbeTimeout
	GatewayDegradedHTTPProbeUnavailable  = gateway.GatewayDegradedHTTPProbeUnavailable
	GatewayDegradedHTTPUnauthorized      = gateway.GatewayDegradedHTTPUnauthorized
	GatewayDegradedCapabilityUnsupported = gateway.GatewayDegradedCapabilityUnsupported
	GatewayDegradedCapabilityMalformed   = gateway.GatewayDegradedCapabilityMalformed
	GatewayDegradedUsageDataUnavailable  = gateway.GatewayDegradedUsageDataUnavailable

	GatewayHealthTCPReachable              = gateway.GatewayHealthTCPReachable
	GatewayHealthUnreachable               = gateway.GatewayHealthUnreachable
	GatewayHealthHTTPHealthy               = gateway.GatewayHealthHTTPHealthy
	GatewayHealthHTTPUnavailable           = gateway.GatewayHealthHTTPUnavailable
	GatewayHealthHTTPUnauthorized          = gateway.GatewayHealthHTTPUnauthorized
	GatewayHealthHTTPCapabilityUnsupported = gateway.GatewayHealthHTTPCapabilityUnsupported
	GatewayHealthHTTPCapabilityMalformed   = gateway.GatewayHealthHTTPCapabilityMalformed

	GatewayProbeStatusRuntimeRunning        = gateway.GatewayProbeStatusRuntimeRunning
	GatewayProbeStatusUnavailable           = gateway.GatewayProbeStatusUnavailable
	GatewayProbeStatusCapabilityReady       = gateway.GatewayProbeStatusCapabilityReady
	GatewayProbeStatusUnauthorized          = gateway.GatewayProbeStatusUnauthorized
	GatewayProbeStatusUnsupportedCapability = gateway.GatewayProbeStatusUnsupportedCapability
	GatewayProbeStatusMalformedCapabilities = gateway.GatewayProbeStatusMalformedCapabilities
)

type GatewayEndpoint = gateway.GatewayEndpoint
type GatewayDegradedStatus = gateway.GatewayDegradedStatus
type GatewayDiscoverResult = gateway.GatewayDiscoverResult
type GatewayDiscoverRequest = gateway.GatewayDiscoverRequest
type GatewayDiscoverer = gateway.GatewayDiscoverer
type GatewayDiscovererFunc = gateway.GatewayDiscovererFunc
type ShellGatewayDiscoverer = gateway.ShellGatewayDiscoverer
type GatewayRuntimeSummary = gateway.GatewayRuntimeSummary
type GatewayProbeRequest = gateway.GatewayProbeRequest
type GatewayDiscoverySummary = gateway.GatewayDiscoverySummary
type GatewayProbeResult = gateway.GatewayProbeResult
type GatewayProbeTarget = gateway.GatewayProbeTarget
type GatewayEndpointProber = gateway.GatewayEndpointProber
type GatewayHTTPAuth = gateway.GatewayHTTPAuth
type GatewayCapabilitiesSummary = gateway.GatewayCapabilitiesSummary
type GatewayEndpointProberFunc = gateway.GatewayEndpointProberFunc
type TCPGatewayProber = gateway.TCPGatewayProber
type HTTPGatewayProber = gateway.HTTPGatewayProber
type GatewayUsageSession = gateway.GatewayUsageSession
type GatewayUsagePricing = gateway.GatewayUsagePricing
type GatewayUsageCostRequest = gateway.GatewayUsageCostRequest
type GatewayUsageSessionLister = gateway.GatewayUsageSessionLister
type GatewayUsageSessionListerFunc = gateway.GatewayUsageSessionListerFunc
type GatewayUsageCostSession = gateway.GatewayUsageCostSession
type GatewayUsageCostTotals = gateway.GatewayUsageCostTotals
type GatewayUsageCostResult = gateway.GatewayUsageCostResult

var DiscoverGateways = gateway.DiscoverGateways
var NormalizeGatewayEndpoint = gateway.NormalizeGatewayEndpoint
var ParseGatewayEndpoint = gateway.ParseGatewayEndpoint
var NewShellGatewayDiscoverer = gateway.NewShellGatewayDiscoverer
var ProbeGateways = gateway.ProbeGateways
var SummarizeGatewayUsageCost = gateway.SummarizeGatewayUsageCost
