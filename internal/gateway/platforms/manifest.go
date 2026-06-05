package platforms

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/inventory"

// PlatformImplementationStatus classifies how much of one upstream Hermes
// gateway platform has a native Gormes surface today.
type PlatformImplementationStatus = inventory.PlatformImplementationStatus

const (
	PlatformStatusImplemented = inventory.PlatformStatusImplemented
	PlatformStatusPartial     = inventory.PlatformStatusPartial
	PlatformStatusRowBacked   = inventory.PlatformStatusRowBacked
	PlatformStatusExcluded    = inventory.PlatformStatusExcluded
	PlatformStatusOwned       = inventory.PlatformStatusOwned
)

// PlatformSurfaceStatus records whether one observable platform capability is
// already implemented, partially implemented, row-backed, intentionally not
// applicable, or owned by a Go-native Gormes surface.
type PlatformSurfaceStatus = inventory.PlatformSurfaceStatus

const (
	PlatformSurfaceImplemented   = inventory.PlatformSurfaceImplemented
	PlatformSurfacePartial       = inventory.PlatformSurfacePartial
	PlatformSurfaceRowBacked     = inventory.PlatformSurfaceRowBacked
	PlatformSurfaceNotApplicable = inventory.PlatformSurfaceNotApplicable
	PlatformSurfaceOwned         = inventory.PlatformSurfaceOwned
)

// PlatformKind distinguishes real operator channels from local/runtime-only
// gateway surfaces that Hermes still exposes through the same Platform enum.
type PlatformKind = inventory.PlatformKind

const (
	PlatformKindChannel = inventory.PlatformKindChannel
	PlatformKindRuntime = inventory.PlatformKindRuntime
	PlatformKindWebhook = inventory.PlatformKindWebhook
	PlatformKindLocal   = inventory.PlatformKindLocal
)

// PlatformManifestEntry is the source-backed platform inventory used by
// gateway/channel planning. It is deliberately data-only: tests compare it to
// upstream Hermes source without starting live SDKs, sockets, webhooks, QR
// flows, or credential readers.
type PlatformManifestEntry = inventory.PlatformManifestEntry

// HermesGatewayPlatformManifest returns a copy of the current Hermes gateway
// Platform enum and connector inventory as understood by Gormes. Unsupported
// platforms remain explicit row-backed entries instead of disappearing from
// status, docs, or follow-up planning.
func HermesGatewayPlatformManifest() []PlatformManifestEntry {
	return inventory.HermesGatewayPlatformManifest()
}
