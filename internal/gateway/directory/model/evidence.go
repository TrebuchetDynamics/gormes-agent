package model

import modelevidence "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/directory/model/evidence"

const (
	// EvidenceChannelDirectoryMissing reports that no usable cached directory was available.
	EvidenceChannelDirectoryMissing = modelevidence.ChannelDirectoryMissing
	// EvidenceChannelDirectoryInvalid reports that the cached directory JSON could not be decoded.
	EvidenceChannelDirectoryInvalid = modelevidence.ChannelDirectoryInvalid
	// EvidenceChannelDirectorySourcesInvalid reports that remembered-source JSON could not be decoded.
	EvidenceChannelDirectorySourcesInvalid = modelevidence.ChannelDirectorySourcesInvalid
	// EvidenceChannelDirectoryRefreshFailed reports that refresh could not produce or persist a new directory.
	EvidenceChannelDirectoryRefreshFailed = modelevidence.ChannelDirectoryRefreshFailed
	// EvidenceChannelTargetAmbiguous reports that a human target query matched multiple cached entries.
	EvidenceChannelTargetAmbiguous = modelevidence.ChannelTargetAmbiguous
	// EvidenceChannelTargetStale reports that an explicit cached target disappeared from a refreshed directory.
	EvidenceChannelTargetStale = modelevidence.ChannelTargetStale
)
