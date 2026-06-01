package listing

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
	if dir == nil {
		return nil, fmt.Errorf("channel directory unavailable")
	}
	platforms, err := loadPlatforms(ctx, dir)
	if err != nil {
		return nil, err
	}

	channels := collectOutputs(platforms, newPlatformSelection(args))

	return map[string]interface{}{
		"count":    len(channels),
		"channels": channels,
	}, nil
}

func loadPlatforms(ctx context.Context, dir DirectoryProvider) (map[string][]Entry, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	platforms, err := dir.Platforms()
	if err != nil {
		return nil, fmt.Errorf("channel directory unavailable: %w", err)
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	return platforms, nil
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

type platformSelection struct {
	Filter    platformKey
	HasFilter bool
}

func newPlatformSelection(args map[string]interface{}) platformSelection {
	if pf, ok := args["platform"].(string); ok {
		key := newPlatformKey(pf)
		return platformSelection{Filter: key, HasFilter: key.Normalized != ""}
	}
	return platformSelection{}
}

func (s platformSelection) matches(key platformKey) bool {
	return !s.HasFilter || key.Normalized == s.Filter.Normalized
}

func collectOutputs(platforms map[string][]Entry, selection platformSelection) []Output {
	channels := make([]Output, 0)
	for _, candidate := range platformCandidates(platforms, selection) {
		for _, entry := range candidate.Entries {
			channels = append(channels, Output{
				ID:       entry.ID,
				Name:     entry.Name,
				ChatID:   entry.ChatID,
				Enabled:  entry.Enabled,
				Platform: candidate.Normalized,
			})
		}
	}
	return channels
}

type platformCandidate struct {
	platformKey
	Entries []Entry
}

type platformKey struct {
	Raw        string
	Normalized string
}

func newPlatformKey(platform string) platformKey {
	return platformKey{
		Raw:        platform,
		Normalized: strings.ToLower(strings.TrimSpace(platform)),
	}
}

func platformCandidates(platforms map[string][]Entry, selection platformSelection) []platformCandidate {
	candidates := make([]platformCandidate, 0, len(platforms))
	for platform, entries := range platforms {
		key := newPlatformKey(platform)
		if !selection.matches(key) {
			continue
		}
		candidates = append(candidates, platformCandidate{
			platformKey: key,
			Entries:     entries,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Normalized != candidates[j].Normalized {
			return candidates[i].Normalized < candidates[j].Normalized
		}
		return candidates[i].Raw < candidates[j].Raw
	})
	return candidates
}
