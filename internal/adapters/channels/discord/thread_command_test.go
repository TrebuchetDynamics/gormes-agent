package discord

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestDiscordThreadCommandCreatesThreadAndDispatchesStarter(t *testing.T) {
	ms := newMockSession()
	ms.currentUserID = "app-1"
	ms.threadStartResult = &discordgo.Channel{ID: "thread-1", ParentID: "chan-1", Name: "Planning", Type: discordgo.ChannelTypeGuildPublicThread}
	b := New(Config{AllowedChannelID: "chan-1"}, ms, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	interaction := newCommandInteraction("thread", "chan-1", "user-1", map[string]any{
		"name":                  "Planning",
		"message":               "Kickoff",
		"auto_archive_duration": 1440,
	})
	b.handleInteraction(context.Background(), inbox, ms, interaction)

	if len(ms.threadStartCallsSnapshot()) != 1 {
		t.Fatalf("thread starts = %+v, want one direct thread create", ms.threadStartCallsSnapshot())
	}
	if len(ms.messageThreadStartCallsSnapshot()) != 0 {
		t.Fatalf("message thread fallback used unexpectedly: %+v", ms.messageThreadStartCallsSnapshot())
	}
	responses := ms.interactionResponsesSnapshot()
	if len(responses) < 2 || responses[0].Type != discordgo.InteractionResponseDeferredChannelMessageWithSource {
		t.Fatalf("responses = %+v, want defer before success follow-up", responses)
	}
	if got := responses[len(responses)-1].Data.Content; !strings.Contains(got, "<#thread-1>") {
		t.Fatalf("success response = %q, want thread mention", got)
	}

	select {
	case ev := <-inbox:
		if ev.Kind != gateway.EventSubmit || ev.Text != "Kickoff" || ev.ChatID != "chan-1" || ev.ThreadID != "thread-1" {
			t.Fatalf("thread starter event = %+v", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no starter event dispatched")
	}
}

func TestDiscordThreadCommandFallsBackThroughSeedMessage(t *testing.T) {
	ms := newMockSession()
	ms.currentUserID = "app-1"
	ms.threadStartErr = errors.New("missing create public threads permission")
	ms.messageThreadStartResult = &discordgo.Channel{ID: "thread-seed", ParentID: "chan-1", Name: "Planning", Type: discordgo.ChannelTypeGuildPublicThread}
	b := New(Config{AllowedChannelID: "chan-1"}, ms, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	interaction := newCommandInteraction("thread", "chan-1", "user-1", map[string]any{
		"name":                  "Planning",
		"message":               "Kickoff",
		"auto_archive_duration": 1440,
	})
	b.handleInteraction(context.Background(), inbox, ms, interaction)

	sent := ms.complexSnapshot()
	if len(sent) != 1 || sent[0].Data == nil || sent[0].Data.Content != "Kickoff" {
		t.Fatalf("seed message = %+v, want starter content sent before fallback thread", sent)
	}
	fallbacks := ms.messageThreadStartCallsSnapshot()
	if len(fallbacks) != 1 || fallbacks[0].MessageID != sent[0].MsgID {
		t.Fatalf("fallback thread calls = %+v, seed=%+v", fallbacks, sent)
	}

	select {
	case ev := <-inbox:
		if ev.ThreadID != "thread-seed" || ev.Text != "Kickoff" {
			t.Fatalf("fallback starter event = %+v", ev)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("no fallback starter event dispatched")
	}
}

func TestDiscordThreadCommandSanitizesAndTruncatesNames(t *testing.T) {
	ms := newMockSession()
	ms.currentUserID = "app-1"
	ms.threadStartResult = &discordgo.Channel{ID: "thread-1", ParentID: "chan-1", Type: discordgo.ChannelTypeGuildPublicThread}
	b := New(Config{AllowedChannelID: "chan-1"}, ms, nil)

	long := "<@123> <@&456> <#789> " + strings.Repeat("a", 100)
	b.handleInteraction(context.Background(), make(chan gateway.InboundEvent, 1), ms, newCommandInteraction("thread", "chan-1", "user-1", map[string]any{
		"name": long,
	}))

	calls := ms.threadStartCallsSnapshot()
	if len(calls) != 1 {
		t.Fatalf("thread calls = %+v, want one", calls)
	}
	name := calls[0].Data.Name
	if strings.Contains(name, "<@") || strings.Contains(name, "<#") {
		t.Fatalf("thread name leaked mention syntax: %q", name)
	}
	if len(name) > discordThreadNameLimit || !strings.HasSuffix(name, "...") {
		t.Fatalf("thread name = %q length=%d, want truncated <= %d with ellipsis", name, len(name), discordThreadNameLimit)
	}
}

func TestDiscordThreadCommandRejectsBeforeDeferWhenUnauthorized(t *testing.T) {
	ms := newMockSession()
	ms.currentUserID = "app-1"
	b := New(Config{AllowedChannelID: "allowed-chan"}, ms, nil)

	b.handleInteraction(context.Background(), make(chan gateway.InboundEvent, 1), ms, newCommandInteraction("thread", "denied-chan", "user-1", map[string]any{
		"name": "Planning",
	}))

	responses := ms.interactionResponsesSnapshot()
	if len(responses) != 1 {
		t.Fatalf("responses = %+v, want only authorization rejection", responses)
	}
	if responses[0].Type == discordgo.InteractionResponseDeferredChannelMessageWithSource {
		t.Fatalf("unauthorized /thread deferred before auth: %+v", responses)
	}
	if responses[0].Data == nil || !strings.Contains(responses[0].Data.Content, "not authorized") {
		t.Fatalf("unauthorized response = %+v, want ephemeral rejection", responses[0])
	}
	if len(ms.threadStartCallsSnapshot()) != 0 {
		t.Fatalf("unauthorized thread starts = %+v, want none", ms.threadStartCallsSnapshot())
	}
}
