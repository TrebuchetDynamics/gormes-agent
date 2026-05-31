package channels

import (
	"context"
	"fmt"
	"sort"
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

	channels := collectOutputs(platforms, platformFilter(args))

	return map[string]interface{}{
		"count":    len(channels),
		"channels": channels,
	}, nil
}

func platformFilter(args map[string]interface{}) string {
	if pf, ok := args["platform"].(string); ok {
		return normalizePlatform(pf)
	}
	return ""
}

func collectOutputs(platforms map[string][]Entry, filterPlatform string) []Output {
	channels := make([]Output, 0)
	for _, platform := range sortedPlatformNames(platforms) {
		if filterPlatform != "" && normalizePlatform(platform) != filterPlatform {
			continue
		}
		for _, entry := range platforms[platform] {
			channels = append(channels, Output{
				ID:       entry.ID,
				Name:     entry.Name,
				ChatID:   entry.ChatID,
				Enabled:  entry.Enabled,
				Platform: platform,
			})
		}
	}
	return channels
}

func sortedPlatformNames(platforms map[string][]Entry) []string {
	names := make([]string, 0, len(platforms))
	for platform := range platforms {
		names = append(names, platform)
	}
	sort.SliceStable(names, func(i, j int) bool {
		left, right := normalizePlatform(names[i]), normalizePlatform(names[j])
		if left != right {
			return left < right
		}
		return names[i] < names[j]
	})
	return names
}

func normalizePlatform(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}
