package discord

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestDiscordSlashCommandsExposeRegistryAndSkipDynamicConflicts(t *testing.T) {
	ms := newMockSession()
	ms.currentUserID = "app-1"
	b := New(Config{
		AllowedChannelID: "chan-1",
		PluginCommands: []gateway.PlatformCommand{
			{Name: "metricas", Description: "Metrics dashboard"},
			{Name: "status", Description: "must not shadow built-in status"},
		},
	}, ms, nil)
	b.setSkillCommandsForTest([]gateway.PlatformCommand{
		{Name: "deploy-prod", Description: "Deploy production"},
	})

	if err := b.registerSlashCommands(context.Background()); err != nil {
		t.Fatalf("registerSlashCommands: %v", err)
	}

	commands := ms.applicationCommandsSnapshot()
	seen := map[string]*discordgo.ApplicationCommand{}
	for _, cmd := range commands {
		seen[cmd.Name] = cmd
	}
	for _, want := range []string{"help", "new", "restart", "status", "thread", "skill", "metricas"} {
		if _, ok := seen[want]; !ok {
			t.Fatalf("registered commands missing %q in %#v", want, commandNames(commands))
		}
	}
	if _, ok := seen["deploy-prod"]; ok {
		t.Fatalf("skill command registered as top-level command; want flat /skill autocomplete: %#v", commandNames(commands))
	}
	if got := countCommand(commands, "status"); got != 1 {
		t.Fatalf("/status registered %d times, want conflict-safe single built-in", got)
	}

	skill := seen["skill"]
	if len(skill.Options) != 1 || skill.Options[0].Name != "name" || !skill.Options[0].Autocomplete {
		t.Fatalf("/skill options = %#v, want one autocomplete name option", skill.Options)
	}
	if len(commands) > 0 && len(discordCommandPayloadBytes(commands)) > discordCommandPayloadSoftLimit {
		t.Fatalf("command payload length = %d, want <= %d", len(discordCommandPayloadBytes(commands)), discordCommandPayloadSoftLimit)
	}
}

func TestDiscordSlashPluginCommandDispatchesOptionalArgs(t *testing.T) {
	ms := newMockSession()
	ms.currentUserID = "app-1"
	b := New(Config{
		AllowedChannelID: "chan-1",
		PluginCommands:   []gateway.PlatformCommand{{Name: "metricas", Description: "Metrics dashboard"}},
	}, ms, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	b.handleInteraction(context.Background(), inbox, ms, newCommandInteraction("metricas", "chan-1", "user-1", map[string]any{"args": "dias:7 formato:json"}))

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit || ev.Text != "/metricas dias:7 formato:json" {
			t.Fatalf("plugin event = %+v, want slash text with optional args", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no plugin event dispatched")
	}
}

func TestDiscordSkillSlashDispatchesSelectedSkillAndGuidesUnknown(t *testing.T) {
	ms := newMockSession()
	ms.currentUserID = "app-1"
	b := New(Config{AllowedChannelID: "chan-1"}, ms, nil)
	b.setSkillCommandsForTest([]gateway.PlatformCommand{{Name: "deploy-prod", Description: "Deploy production"}})

	inbox := make(chan gateway.InboundEvent, 1)
	b.handleInteraction(context.Background(), inbox, ms, newCommandInteraction("skill", "chan-1", "user-1", map[string]any{"name": "deploy-prod"}))

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit || ev.Text != "/deploy-prod" || ev.Platform != "discord" || ev.ChatID != "chan-1" {
			t.Fatalf("skill slash event = %+v, want selected skill dispatched as its slash command text", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no skill event dispatched")
	}

	b.handleInteraction(context.Background(), inbox, ms, newCommandInteraction("skill", "chan-1", "user-1", map[string]any{"name": "missing-skill"}))
	responses := ms.interactionResponsesSnapshot()
	if len(responses) == 0 {
		t.Fatal("unknown skill did not send ephemeral guidance")
	}
	last := responses[len(responses)-1]
	if last.Type != discordgo.InteractionResponseChannelMessageWithSource || last.Data == nil || !strings.Contains(last.Data.Content, "unknown skill") {
		t.Fatalf("unknown-skill response = %+v, want ephemeral guidance", last)
	}
	if last.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatalf("unknown-skill response flags = %v, want ephemeral", last.Data.Flags)
	}
}

func TestDiscordSlashSourcePreservesThreadContext(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "parent-1"}, ms, nil)
	b.rememberThread(&discordgo.Channel{
		ID:       "thread-1",
		ParentID: "parent-1",
		Name:     "Planning",
		GuildID:  "guild-1",
		Type:     discordgo.ChannelTypeGuildPublicThread,
	})

	inbox := make(chan gateway.InboundEvent, 1)
	b.handleInteraction(context.Background(), inbox, ms, newCommandInteraction("status", "thread-1", "user-1", nil))

	select {
	case ev := <-inbox:
		if ev.ChatID != "parent-1" || ev.ThreadID != "thread-1" || ev.ParentChatID != "parent-1" || ev.GuildID != "guild-1" {
			t.Fatalf("thread slash source = %+v, want parent chat plus thread metadata", ev)
		}
		if ev.Kind != gateway.EventStatus {
			t.Fatalf("Kind = %v, want status", ev.Kind)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no slash source event")
	}
}

func TestDiscordSystemMessagesAreDropped(t *testing.T) {
	for _, msgType := range []discordgo.MessageType{
		discordgo.MessageTypeChannelNameChange,
		discordgo.MessageTypeChannelPinnedMessage,
		discordgo.MessageTypeGuildMemberJoin,
		discordgo.MessageTypeUserPremiumGuildSubscription,
		discordgo.MessageTypeRecipientAdd,
	} {
		t.Run(fmt.Sprintf("type-%d", msgType), func(t *testing.T) {
			ms := newMockSession()
			b := New(Config{AllowedChannelID: "chan-1"}, ms, nil)
			inbox := make(chan gateway.InboundEvent, 1)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = b.Run(ctx, inbox) }()
			ms.waitOpen(t)

			ms.deliver(&discordgo.MessageCreate{Message: &discordgo.Message{
				ID:        "m-system",
				ChannelID: "chan-1",
				Content:   "system noise",
				Type:      msgType,
				Author:    &discordgo.User{ID: "user-1"},
			}})

			select {
			case ev := <-inbox:
				t.Fatalf("system message produced event %+v", ev)
			case <-time.After(50 * time.Millisecond):
			}
		})
	}
}

func newCommandInteraction(name, channelID, userID string, opts map[string]any) *discordgo.InteractionCreate {
	options := make([]*discordgo.ApplicationCommandInteractionDataOption, 0, len(opts))
	for key, value := range opts {
		switch v := value.(type) {
		case string:
			options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: key, Type: discordgo.ApplicationCommandOptionString, Value: v})
		case int:
			options = append(options, &discordgo.ApplicationCommandInteractionDataOption{Name: key, Type: discordgo.ApplicationCommandOptionInteger, Value: float64(v)})
		}
	}
	return &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
		ID:        "interaction-" + name,
		Token:     "token-" + name,
		Type:      discordgo.InteractionApplicationCommand,
		GuildID:   "guild-1",
		ChannelID: channelID,
		Data: discordgo.ApplicationCommandInteractionData{
			Name:    name,
			Options: options,
		},
		User: &discordgo.User{ID: userID, Username: "operator"},
	}}
}

func commandNames(commands []*discordgo.ApplicationCommand) []string {
	names := make([]string, 0, len(commands))
	for _, cmd := range commands {
		names = append(names, cmd.Name)
	}
	return names
}

func countCommand(commands []*discordgo.ApplicationCommand, name string) int {
	count := 0
	for _, cmd := range commands {
		if cmd.Name == name {
			count++
		}
	}
	return count
}
