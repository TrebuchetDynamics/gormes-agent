package evidence

const (
	// ChannelDirectoryMissing reports that no usable cached directory was available.
	ChannelDirectoryMissing = "channel_directory_missing"
	// ChannelDirectoryInvalid reports that the cached directory JSON could not be decoded.
	ChannelDirectoryInvalid = "channel_directory_invalid"
	// ChannelDirectorySourcesInvalid reports that remembered-source JSON could not be decoded.
	ChannelDirectorySourcesInvalid = "channel_directory_sources_invalid"
	// ChannelDirectoryRefreshFailed reports that refresh could not produce or persist a new directory.
	ChannelDirectoryRefreshFailed = "channel_directory_refresh_failed"
	// ChannelTargetAmbiguous reports that a human target query matched multiple cached entries.
	ChannelTargetAmbiguous = "channel_target_ambiguous"
	// ChannelTargetStale reports that an explicit cached target disappeared from a refreshed directory.
	ChannelTargetStale = "channel_target_stale"
)
