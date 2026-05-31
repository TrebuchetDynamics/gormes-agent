package model

const (
	// EvidenceChannelDirectoryMissing reports that no usable cached directory was available.
	EvidenceChannelDirectoryMissing = "channel_directory_missing"
	// EvidenceChannelDirectoryInvalid reports that the cached directory JSON could not be decoded.
	EvidenceChannelDirectoryInvalid = "channel_directory_invalid"
	// EvidenceChannelDirectorySourcesInvalid reports that remembered-source JSON could not be decoded.
	EvidenceChannelDirectorySourcesInvalid = "channel_directory_sources_invalid"
	// EvidenceChannelDirectoryRefreshFailed reports that refresh could not produce or persist a new directory.
	EvidenceChannelDirectoryRefreshFailed = "channel_directory_refresh_failed"
	// EvidenceChannelTargetAmbiguous reports that a human target query matched multiple cached entries.
	EvidenceChannelTargetAmbiguous = "channel_target_ambiguous"
	// EvidenceChannelTargetStale reports that an explicit cached target disappeared from a refreshed directory.
	EvidenceChannelTargetStale = "channel_target_stale"
)
