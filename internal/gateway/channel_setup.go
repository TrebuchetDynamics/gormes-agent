package gateway

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	gatewaychannelsetup "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/channelsetup"
)

type ChannelSetupStatus = gatewaychannelsetup.ChannelSetupStatus

const (
	ChannelSetupStatusUnconfigured ChannelSetupStatus = gatewaychannelsetup.ChannelSetupStatusUnconfigured
	ChannelSetupStatusPartial      ChannelSetupStatus = gatewaychannelsetup.ChannelSetupStatusPartial
	ChannelSetupStatusConfigured   ChannelSetupStatus = gatewaychannelsetup.ChannelSetupStatusConfigured
	ChannelSetupStatusPaired       ChannelSetupStatus = gatewaychannelsetup.ChannelSetupStatusPaired
	ChannelSetupStatusRunning      ChannelSetupStatus = gatewaychannelsetup.ChannelSetupStatusRunning
	ChannelSetupStatusFailed       ChannelSetupStatus = gatewaychannelsetup.ChannelSetupStatusFailed
)

type ChannelSetupPlan = gatewaychannelsetup.ChannelSetupPlan

type ChannelSetupEntry = gatewaychannelsetup.ChannelSetupEntry

// ChannelSetupPlanOptions carries optional read-only runtime evidence used to
// enrich setup guidance without reading live state from disk.
type ChannelSetupPlanOptions struct {
	Pairing PairingStatus
	// CredentialHashes carries caller-supplied redacted token hashes keyed by
	// credential id. Setup planning uses them only for duplicate ownership
	// evidence and never resolves live secret values.
	CredentialHashes map[string]string
}

func BuildChannelSetupPlan(cfg config.Config) ChannelSetupPlan {
	return gatewaychannelsetup.BuildChannelSetupPlan(cfg)
}

// BuildChannelSetupPlanWithOptions builds setup guidance from config plus
// caller-supplied read-model evidence such as gateway pairing status.
func BuildChannelSetupPlanWithOptions(cfg config.Config, opts ChannelSetupPlanOptions) ChannelSetupPlan {
	return gatewaychannelsetup.BuildChannelSetupPlanWithOptions(cfg, gatewaychannelsetup.ChannelSetupPlanOptions{
		Pairing:          convertChannelSetupPairingStatus(opts.Pairing),
		CredentialHashes: opts.CredentialHashes,
	})
}

func convertChannelSetupPairingStatus(status PairingStatus) gatewaychannelsetup.PairingStatus {
	out := gatewaychannelsetup.PairingStatus{
		Platforms: make([]gatewaychannelsetup.PairingPlatformStatus, 0, len(status.Platforms)),
		Degraded:  make([]gatewaychannelsetup.PairingDegradedEvidence, 0, len(status.Degraded)),
	}
	for _, platform := range status.Platforms {
		out.Platforms = append(out.Platforms, gatewaychannelsetup.PairingPlatformStatus{
			Platform:      platform.Platform,
			State:         gatewaychannelsetup.PairingPlatformState(platform.State),
			PendingCount:  platform.PendingCount,
			ApprovedCount: platform.ApprovedCount,
		})
	}
	for _, evidence := range status.Degraded {
		out.Degraded = append(out.Degraded, gatewaychannelsetup.PairingDegradedEvidence{
			Platform: evidence.Platform,
			Reason:   gatewaychannelsetup.PairingDegradedReason(evidence.Reason),
		})
	}
	return out
}
