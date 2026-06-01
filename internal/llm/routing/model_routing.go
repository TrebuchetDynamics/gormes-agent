package routing

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/routepolicy"

type ModelRouteReason = routepolicy.ModelRouteReason

const (
	ModelRouteReasonPrimary             ModelRouteReason = routepolicy.ModelRouteReasonPrimary
	ModelRouteReasonTurnOverride        ModelRouteReason = routepolicy.ModelRouteReasonTurnOverride
	ModelRouteReasonConfigOverride      ModelRouteReason = routepolicy.ModelRouteReasonConfigOverride
	ModelRouteReasonAutomaticSimpleTurn ModelRouteReason = routepolicy.ModelRouteReasonAutomaticSimpleTurn
	ModelRouteReasonFallbackSelected    ModelRouteReason = routepolicy.ModelRouteReasonFallbackSelected
	ModelRouteReasonFallbackDisabled    ModelRouteReason = routepolicy.ModelRouteReasonFallbackDisabled
	ModelRouteReasonFallbackUnavailable ModelRouteReason = routepolicy.ModelRouteReasonFallbackUnavailable
)

type ModelRoutingStatusCode = routepolicy.ModelRoutingStatusCode

const (
	ModelRoutingStatusProviderUnavailable   ModelRoutingStatusCode = routepolicy.ModelRoutingStatusProviderUnavailable
	ModelRoutingStatusMetadataGap           ModelRoutingStatusCode = routepolicy.ModelRoutingStatusMetadataGap
	ModelRoutingStatusInvalidOverride       ModelRoutingStatusCode = routepolicy.ModelRoutingStatusInvalidOverride
	ModelRoutingStatusFallbackDisabled      ModelRoutingStatusCode = routepolicy.ModelRoutingStatusFallbackDisabled
	ModelRoutingStatusFallbackUnavailable   ModelRoutingStatusCode = routepolicy.ModelRoutingStatusFallbackUnavailable
	ModelRoutingStatusAutomaticRouteSkipped ModelRoutingStatusCode = routepolicy.ModelRoutingStatusAutomaticRouteSkipped
)

type ModelRoute = routepolicy.ModelRoute
type ProviderAvailability = routepolicy.ProviderAvailability
type AutomaticModelRoutingPolicy = routepolicy.AutomaticModelRoutingPolicy
type FallbackModelPolicy = routepolicy.FallbackModelPolicy
type ModelRoutingRequest = routepolicy.ModelRoutingRequest
type ModelRoutingStatus = routepolicy.ModelRoutingStatus
type ModelRoutingDecision = routepolicy.ModelRoutingDecision
type ModelRouterConfig = routepolicy.ModelRouterConfig
type ModelRouter = routepolicy.ModelRouter

func NormalizeFallbackModelConfig(value any) FallbackModelPolicy {
	return routepolicy.NormalizeFallbackModelConfig(value)
}

func NewModelRouter(config ModelRouterConfig) ModelRouter {
	return routepolicy.NewModelRouter(config)
}
