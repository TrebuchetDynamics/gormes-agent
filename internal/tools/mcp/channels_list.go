package mcp

import (
	"context"
	"fmt"
	"strings"
)

// ChannelEntry represents one platform channel for MCP channels_list output.
type ChannelEntry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	ChatID  string `json:"chat_id"`
	Enabled bool   `json:"enabled"`
}

// ChannelDirectoryProvider supplies platform channel data to MCP tools.
type ChannelDirectoryProvider interface {
	Platforms() (map[string][]ChannelEntry, error)
}

// ChannelOutput is one normalized channels_list entry.
type ChannelOutput struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ChatID   string `json:"chat_id"`
	Enabled  bool   `json:"enabled"`
	Platform string `json:"platform"`
}

// ListChannels returns the MCP channels_list payload for a directory provider.
func ListChannels(ctx context.Context, dir ChannelDirectoryProvider, args map[string]interface{}) (interface{}, error) {
	_ = ctx
	if dir == nil {
		return nil, fmt.Errorf("channel directory unavailable")
	}
	platforms, err := dir.Platforms()
	if err != nil {
		return nil, fmt.Errorf("channel directory unavailable: %w", err)
	}

	filterPlatform := ""
	if pf, ok := args["platform"].(string); ok {
		filterPlatform = strings.ToLower(strings.TrimSpace(pf))
	}

	channels := make([]ChannelOutput, 0)
	for platform, entries := range platforms {
		if filterPlatform != "" && strings.ToLower(platform) != filterPlatform {
			continue
		}
		for _, entry := range entries {
			channels = append(channels, ChannelOutput{
				ID:       entry.ID,
				Name:     entry.Name,
				ChatID:   entry.ChatID,
				Enabled:  entry.Enabled,
				Platform: platform,
			})
		}
	}

	return map[string]interface{}{
		"count":    len(channels),
		"channels": channels,
	}, nil
}
