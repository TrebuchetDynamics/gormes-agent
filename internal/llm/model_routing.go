package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"

type ModelRouteReason = routing.ModelRouteReason

const (
	ModelRouteReasonPrimary             = routing.ModelRouteReasonPrimary
	ModelRouteReasonTurnOverride        = routing.ModelRouteReasonTurnOverride
	ModelRouteReasonConfigOverride      = routing.ModelRouteReasonConfigOverride
	ModelRouteReasonAutomaticSimpleTurn = routing.ModelRouteReasonAutomaticSimpleTurn
	ModelRouteReasonFallbackSelected    = routing.ModelRouteReasonFallbackSelected
	ModelRouteReasonFallbackDisabled    = routing.ModelRouteReasonFallbackDisabled
	ModelRouteReasonFallbackUnavailable = routing.ModelRouteReasonFallbackUnavailable
)

type ModelRoutingStatusCode = routing.ModelRoutingStatusCode

const (
	ModelRoutingStatusProviderUnavailable   = routing.ModelRoutingStatusProviderUnavailable
	ModelRoutingStatusMetadataGap           = routing.ModelRoutingStatusMetadataGap
	ModelRoutingStatusInvalidOverride       = routing.ModelRoutingStatusInvalidOverride
	ModelRoutingStatusFallbackDisabled      = routing.ModelRoutingStatusFallbackDisabled
	ModelRoutingStatusFallbackUnavailable   = routing.ModelRoutingStatusFallbackUnavailable
	ModelRoutingStatusAutomaticRouteSkipped = routing.ModelRoutingStatusAutomaticRouteSkipped
)

type ModelRoute = routing.ModelRoute
type ProviderAvailability = routing.ProviderAvailability
type AutomaticModelRoutingPolicy = routing.AutomaticModelRoutingPolicy
type FallbackModelPolicy = routing.FallbackModelPolicy

type ModelRoutingRequest = routing.ModelRoutingRequest
type ModelRoutingStatus = routing.ModelRoutingStatus
type ModelRoutingDecision = routing.ModelRoutingDecision
type ModelRouterConfig = routing.ModelRouterConfig
type ModelRouter = routing.ModelRouter

func NormalizeFallbackModelConfig(value any) FallbackModelPolicy {
	return routing.NormalizeFallbackModelConfig(value)
}

func NewModelRouter(config ModelRouterConfig) ModelRouter {
	return routing.NewModelRouter(config)
}
