package tools

import (
	"context"
	"fmt"
	"testing"
)

// fakeChannelDir implements ChannelDirectoryProvider for tests.
type fakeChannelDir struct {
	platforms map[string][]ChannelEntry
	err       error
}

func (f fakeChannelDir) Platforms() (map[string][]ChannelEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.platforms, nil
}

func TestMCPChannelsList_ReturnsPerPlatformChannels(t *testing.T) {
	dir := fakeChannelDir{
		platforms: map[string][]ChannelEntry{
			"telegram": {
				{ID: "tg-1", Name: "General", ChatID: "123", Enabled: true},
				{ID: "tg-2", Name: "Dev", ChatID: "456", Enabled: true},
			},
			"discord": {
				{ID: "dc-1", Name: "general", ChatID: "789", Enabled: true},
			},
		},
	}
	s := &MCPServer{channelDir: dir}
	result, err := s.channelsListHandler(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	channels, ok := r["channels"].([]channelOutput)
	if !ok {
		t.Fatalf("expected channels array, got %T", r["channels"])
	}
	if len(channels) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(channels))
	}
	platforms := map[string]bool{}
	for _, ch := range channels {
		platforms[ch.Platform] = true
	}
	if !platforms["telegram"] || !platforms["discord"] {
		t.Fatalf("expected telegram and discord channels, got platforms: %v", platforms)
	}
	if cnt, ok := r["count"]; !ok || cnt.(int) != 3 {
		t.Fatalf("expected count=3, got %v", cnt)
	}
}

func TestMCPChannelsList_PlatformFilter(t *testing.T) {
	dir := fakeChannelDir{
		platforms: map[string][]ChannelEntry{
			"telegram": {
				{ID: "tg-1", Name: "General", ChatID: "123", Enabled: true},
			},
			"discord": {
				{ID: "dc-1", Name: "general", ChatID: "789", Enabled: true},
			},
		},
	}
	s := &MCPServer{channelDir: dir}
	result, err := s.channelsListHandler(context.Background(), map[string]interface{}{
		"platform": "telegram",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := result.(map[string]interface{})
	channels := r["channels"].([]channelOutput)
	if len(channels) != 1 {
		t.Fatalf("expected 1 channel after filtering, got %d", len(channels))
	}
	if channels[0].Platform != "telegram" {
		t.Fatalf("expected telegram channel, got %v", channels[0].Platform)
	}
}

func TestMCPChannelsList_EmptyDirectory(t *testing.T) {
	dir := fakeChannelDir{
		platforms: map[string][]ChannelEntry{},
	}
	s := &MCPServer{channelDir: dir}
	result, err := s.channelsListHandler(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := result.(map[string]interface{})
	channels := r["channels"].([]channelOutput)
	if len(channels) != 0 {
		t.Fatalf("expected 0 channels for empty directory, got %d", len(channels))
	}
	if cnt, ok := r["count"]; !ok || cnt.(int) != 0 {
		t.Fatalf("expected count=0, got %v", cnt)
	}
}

func TestMCPChannelsList_DirectoryUnavailable(t *testing.T) {
	dir := fakeChannelDir{err: fmt.Errorf("directory store unavailable")}
	s := &MCPServer{channelDir: dir}
	_, err := s.channelsListHandler(context.Background(), map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unavailable directory, got nil")
	}
}

func TestMCPChannelsList_ChannelFields(t *testing.T) {
	dir := fakeChannelDir{
		platforms: map[string][]ChannelEntry{
			"telegram": {
				{ID: "tg-1", Name: "General Chat", ChatID: "12345", Enabled: true},
			},
		},
	}
	s := &MCPServer{channelDir: dir}
	result, err := s.channelsListHandler(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := result.(map[string]interface{})
	channels := r["channels"].([]channelOutput)
	ch := channels[0]
	if ch.ID != "tg-1" {
		t.Errorf("expected id='tg-1', got %v", ch.ID)
	}
	if ch.Name != "General Chat" {
		t.Errorf("expected name='General Chat', got %v", ch.Name)
	}
	if ch.ChatID != "12345" {
		t.Errorf("expected chat_id='12345', got %v", ch.ChatID)
	}
	if ch.Enabled != true {
		t.Errorf("expected enabled=true, got %v", ch.Enabled)
	}
	if ch.Platform != "telegram" {
		t.Errorf("expected platform='telegram', got %v", ch.Platform)
	}
}
