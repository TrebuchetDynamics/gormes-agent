package gateway

import gatewayplatforms "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms"

// PlatformImplementationStatus classifies how much of one upstream Hermes
// gateway platform has a native Gormes surface today.
type PlatformImplementationStatus = gatewayplatforms.PlatformImplementationStatus

const (
	PlatformStatusImplemented = gatewayplatforms.PlatformStatusImplemented
	PlatformStatusPartial     = gatewayplatforms.PlatformStatusPartial
	PlatformStatusRowBacked   = gatewayplatforms.PlatformStatusRowBacked
	PlatformStatusExcluded    = gatewayplatforms.PlatformStatusExcluded
	PlatformStatusOwned       = gatewayplatforms.PlatformStatusOwned
)

// PlatformSurfaceStatus records whether one observable platform capability is
// already implemented, partially implemented, row-backed, intentionally not
// applicable, or owned by a Go-native Gormes surface.
type PlatformSurfaceStatus = gatewayplatforms.PlatformSurfaceStatus

const (
	PlatformSurfaceImplemented   = gatewayplatforms.PlatformSurfaceImplemented
	PlatformSurfacePartial       = gatewayplatforms.PlatformSurfacePartial
	PlatformSurfaceRowBacked     = gatewayplatforms.PlatformSurfaceRowBacked
	PlatformSurfaceNotApplicable = gatewayplatforms.PlatformSurfaceNotApplicable
	PlatformSurfaceOwned         = gatewayplatforms.PlatformSurfaceOwned
)

// PlatformKind distinguishes real operator channels from local/runtime-only
// gateway surfaces that Hermes still exposes through the same Platform enum.
type PlatformKind = gatewayplatforms.PlatformKind

const (
	PlatformKindChannel = gatewayplatforms.PlatformKindChannel
	PlatformKindRuntime = gatewayplatforms.PlatformKindRuntime
	PlatformKindWebhook = gatewayplatforms.PlatformKindWebhook
	PlatformKindLocal   = gatewayplatforms.PlatformKindLocal
)

// PlatformManifestEntry is the source-backed platform inventory used by
// gateway/channel planning.
type PlatformManifestEntry = gatewayplatforms.PlatformManifestEntry

// HermesGatewayPlatformManifest returns a copy of the current Hermes gateway
// Platform enum and connector inventory as understood by Gormes.
func HermesGatewayPlatformManifest() []PlatformManifestEntry {
	return gatewayplatforms.HermesGatewayPlatformManifest()
}

// GormesOwnedPlatformManifest returns first-party Gormes operator channels
// that are intentionally outside Hermes' upstream platform inventory.
func GormesOwnedPlatformManifest() []PlatformManifestEntry {
	return gatewayplatforms.GormesOwnedPlatformManifest()
}

// OperatorPlatformManifest combines Hermes parity rows with Gormes-owned
// operator channels for user-facing capability/status reports.
func OperatorPlatformManifest() []PlatformManifestEntry {
	return gatewayplatforms.OperatorPlatformManifest()
}
