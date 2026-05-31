package channels

import (
	"context"
	"fmt"
	"strings"
)

// Entry represents one platform channel for MCP channels_list output.
type Entry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	ChatID  string `json:"chat_id"`
	Enabled bool   `json:"enabled"`
}

// DirectoryProvider supplies platform channel data to MCP tools.
type DirectoryProvider interface {
	Platforms() (map[string][]Entry, error)
}

// Output is one normalized channels_list entry.
type Output struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ChatID   string `json:"chat_id"`
	Enabled  bool   `json:"enabled"`
	Platform string `json:"platform"`
}

// List returns the MCP channels_list payload for a directory provider.
func List(ctx context.Context, dir DirectoryProvider, args map[string]interface{}) (interface{}, error) {
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

	channels := make([]Output, 0)
	for platform, entries := range platforms {
		if filterPlatform != "" && strings.ToLower(platform) != filterPlatform {
			continue
		}
		for _, entry := range entries {
			channels = append(channels, Output{
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
