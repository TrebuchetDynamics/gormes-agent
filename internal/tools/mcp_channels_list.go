package tools

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

type channelOutput struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ChatID   string `json:"chat_id"`
	Enabled  bool   `json:"enabled"`
	Platform string `json:"platform"`
}

func (s *MCPServer) channelsListHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	if s.channelDir == nil {
		return nil, fmt.Errorf("channel directory unavailable")
	}
	platforms, err := s.channelDir.Platforms()
	if err != nil {
		return nil, fmt.Errorf("channel directory unavailable: %w", err)
	}

	filterPlatform := ""
	if pf, ok := args["platform"].(string); ok {
		filterPlatform = strings.ToLower(strings.TrimSpace(pf))
	}

	channels := make([]channelOutput, 0)
	for platform, entries := range platforms {
		if filterPlatform != "" && strings.ToLower(platform) != filterPlatform {
			continue
		}
		for _, entry := range entries {
			channels = append(channels, channelOutput{
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
