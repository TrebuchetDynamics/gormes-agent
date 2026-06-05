package discord

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestDiscordChannelScopeMessageAddsSkillsAndPrompt(t *testing.T) {
	b := New(Config{
		AllowedChannelID: "forum-100",
		ChannelSkillBindings: []gateway.ChannelSkillBinding{
			{ID: "forum-100", Skills: []string{"forum-skill", "shared"}},
			{ID: "thread-200", Skill: "thread-skill"},
		},
		ChannelPrompts: map[string]string{
			"forum-100":  "Forum prompt",
			"thread-200": "Thread prompt",
		},
	}, newMockSession(), nil)
	b.rememberThread(&discordgo.Channel{
		ID:       "thread-200",
		ParentID: "forum-100",
		Name:     "Work thread",
		GuildID:  "guild-1",
		Type:     discordgo.ChannelTypeGuildPublicThread,
	})

	ev, ok := b.toInboundEvent(&discordgo.Message{
		ID:        "msg-1",
		ChannelID: "thread-200",
		GuildID:   "guild-1",
		Content:   "hello from thread",
		Author:    &discordgo.User{ID: "user-1"},
	})
	if !ok {
		t.Fatal("toInboundEvent returned ok=false")
	}
	if !reflect.DeepEqual(ev.AutoSkills, []string{"thread-skill"}) {
		t.Fatalf("AutoSkills = %#v, want thread exact skill", ev.AutoSkills)
	}
	if ev.ChannelPrompt != "Thread prompt" {
		t.Fatalf("ChannelPrompt = %q, want exact thread prompt", ev.ChannelPrompt)
	}
	if ev.ChatID != "forum-100" || ev.ParentChatID != "forum-100" || ev.ThreadID != "thread-200" {
		t.Fatalf("routing ids = chat:%q parent:%q thread:%q", ev.ChatID, ev.ParentChatID, ev.ThreadID)
	}
}

func TestDiscordChannelScopeFallsBackToParent(t *testing.T) {
	b := New(Config{
		AllowedChannelID: "forum-100",
		ChannelSkillBindings: []gateway.ChannelSkillBinding{
			{ID: "forum-100", Skills: []string{"forum-skill", "shared"}},
		},
		ChannelPrompts: map[string]string{"forum-100": "Forum prompt"},
	}, newMockSession(), nil)
	b.rememberThread(&discordgo.Channel{
		ID:       "thread-200",
		ParentID: "forum-100",
		Name:     "Work thread",
		GuildID:  "guild-1",
		Type:     discordgo.ChannelTypeGuildPublicThread,
	})

	ev, ok := b.toInboundEvent(&discordgo.Message{
		ID:        "msg-1",
		ChannelID: "thread-200",
		GuildID:   "guild-1",
		Content:   "hello from thread",
		Author:    &discordgo.User{ID: "user-1"},
	})
	if !ok {
		t.Fatal("toInboundEvent returned ok=false")
	}
	if !reflect.DeepEqual(ev.AutoSkills, []string{"forum-skill", "shared"}) {
		t.Fatalf("AutoSkills = %#v, want parent skills", ev.AutoSkills)
	}
	if ev.ChannelPrompt != "Forum prompt" {
		t.Fatalf("ChannelPrompt = %q, want parent prompt", ev.ChannelPrompt)
	}
}

func TestReloadSkillsDiscordResyncPreservesPreviousOnCollectorError(t *testing.T) {
	commands := []gateway.PlatformCommand{
		{Name: "zeta", Description: "last"},
		{Name: "alpha", Description: "first"},
	}
	b := New(Config{
		SkillCollector: func(context.Context) ([]gateway.PlatformCommand, error) {
			return commands, nil
		},
	}, newMockSession(), nil)

	result, err := b.RefreshSkillGroup(context.Background())
	if err != nil {
		t.Fatalf("RefreshSkillGroup: %v", err)
	}
	if result.Count != 2 || result.Hidden != 0 {
		t.Fatalf("result = %+v, want count=2 hidden=0", result)
	}
	if got := skillNames(b.SkillGroupCommands()); !reflect.DeepEqual(got, []string{"alpha", "zeta"}) {
		t.Fatalf("SkillGroupCommands = %#v, want alphabetical alpha/zeta", got)
	}

	b.cfg.SkillCollector = func(context.Context) ([]gateway.PlatformCommand, error) {
		return nil, errors.New("collector unavailable")
	}
	result, err = b.RefreshSkillGroup(context.Background())
	if err == nil {
		t.Fatal("RefreshSkillGroup error = nil, want collector error")
	}
	if result.Count != 2 {
		t.Fatalf("error result count = %d, want previous count 2", result.Count)
	}
	if got := skillNames(b.SkillGroupCommands()); !reflect.DeepEqual(got, []string{"alpha", "zeta"}) {
		t.Fatalf("SkillGroupCommands after error = %#v, want previous entries preserved", got)
	}
}

func skillNames(commands []gateway.PlatformCommand) []string {
	out := make([]string, 0, len(commands))
	for _, cmd := range commands {
		out = append(out, cmd.Name)
	}
	return out
}
