package tools

import (
	"context"
	"testing"
)

func TestMCPChannelsListFacadeDelegatesToMCPPackage(t *testing.T) {
	s := &MCPServer{channelDir: fakeFacadeChannelDir{platforms: map[string][]ChannelEntry{
		"telegram": {{ID: "tg-1", Name: "General", ChatID: "123", Enabled: true}},
	}}}
	result, err := s.channelsListHandler(context.Background(), map[string]interface{}{"platform": "telegram"})
	if err != nil {
		t.Fatalf("channelsListHandler: %v", err)
	}
	out := result.(map[string]interface{})
	channels := out["channels"].([]channelOutput)
	if out["count"] != 1 || len(channels) != 1 || channels[0].Platform != "telegram" {
		t.Fatalf("result = %#v, want one telegram channel", result)
	}
}

type fakeFacadeChannelDir struct {
	platforms map[string][]ChannelEntry
}

func (f fakeFacadeChannelDir) Platforms() (map[string][]ChannelEntry, error) {
	return f.platforms, nil
}
