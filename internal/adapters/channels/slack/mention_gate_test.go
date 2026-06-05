package slack

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestSlackMentionPolicy_DefaultsRequireMentionAndNoFreeChannels(t *testing.T) {
	policy := ResolveMentionPolicy(MentionPolicyConfig{})
	if !policy.RequireMention {
		t.Fatal("RequireMention = false, want safe default true")
	}
	if policy.StrictMention {
		t.Fatal("StrictMention = true, want default false")
	}
	if len(policy.FreeResponseChannels) != 0 {
		t.Fatalf("FreeResponseChannels = %#v, want empty", policy.FreeResponseChannels)
	}
}

func TestSlackMentionPolicy_ParsesConfigAndEnv(t *testing.T) {
	policy := ResolveMentionPolicy(MentionPolicyConfig{
		RequireMention:       "false",
		StrictMention:        "yes",
		FreeResponseChannels: []any{"C-free", 12345},
	})
	if policy.RequireMention {
		t.Fatal("RequireMention = true, want false from config string")
	}
	if !policy.StrictMention {
		t.Fatal("StrictMention = false, want true from config string")
	}
	if !policy.ChannelAllowsFreeResponse("C-free") || !policy.ChannelAllowsFreeResponse("12345") {
		t.Fatalf("FreeResponseChannels = %#v, want C-free and 12345", policy.FreeResponseChannels)
	}

	env := map[string]string{
		"GORMES_SLACK_REQUIRE_MENTION":        "no",
		"GORMES_SLACK_STRICT_MENTION":         "on",
		"GORMES_SLACK_FREE_RESPONSE_CHANNELS": "C-env, C-other",
	}
	policy = ResolveMentionPolicy(MentionPolicyConfig{LookupEnv: func(name string) string {
		return env[name]
	}})
	if policy.RequireMention || !policy.StrictMention || !policy.ChannelAllowsFreeResponse("C-env") || !policy.ChannelAllowsFreeResponse("C-other") {
		t.Fatalf("policy from env = %+v", policy)
	}
}

func TestSlackMentionPolicy_InvalidValuesFailClosedForRequireMention(t *testing.T) {
	policy := ResolveMentionPolicy(MentionPolicyConfig{
		RequireMention: "maybe",
		StrictMention:  "maybe",
	})
	if !policy.RequireMention {
		t.Fatal("invalid require_mention disabled gating, want fail-closed true")
	}
	if policy.StrictMention {
		t.Fatal("invalid strict_mention enabled strict mode, want legacy default false")
	}
	if !hasMentionPolicyEvidence(policy.Evidence, SlackMentionPolicyUnavailable) {
		t.Fatalf("Evidence = %#v, want %s", policy.Evidence, SlackMentionPolicyUnavailable)
	}
}

func TestMentionGate_ChannelMessagesRequireMentionByDefault(t *testing.T) {
	policy := ResolveMentionPolicy(MentionPolicyConfig{})
	decision := EvaluateMentionGate(policy, MentionGateInput{
		ChannelID: "C123",
		UserID:    "U1",
		BotUserID: "UBOT",
		Text:      "hello channel",
		Timestamp: "1711111111.000100",
	})
	if decision.Process {
		t.Fatal("Process = true, want unmentioned channel message ignored")
	}
}

func TestMentionGate_AllowsDMsFreeResponseAndMentions(t *testing.T) {
	policy := ResolveMentionPolicy(MentionPolicyConfig{FreeResponseChannels: "C-free"})
	tests := []MentionGateInput{
		{ChannelID: "D123", UserID: "U1", BotUserID: "UBOT", Text: "hello dm"},
		{ChannelID: "C-free", UserID: "U1", BotUserID: "UBOT", Text: "hello free"},
		{ChannelID: "C123", UserID: "U1", BotUserID: "UBOT", Text: "<@UBOT> hello"},
	}
	for _, input := range tests {
		decision := EvaluateMentionGate(policy, input)
		if !decision.Process {
			t.Fatalf("EvaluateMentionGate(%+v).Process = false, want true", input)
		}
	}
	decision := EvaluateMentionGate(policy, tests[2])
	if decision.Text != "hello" {
		t.Fatalf("stripped text = %q, want hello", decision.Text)
	}
}

func TestMentionGate_AllowsActiveThreadReplyWithoutFreshMention(t *testing.T) {
	policy := ResolveMentionPolicy(MentionPolicyConfig{})
	decision := EvaluateMentionGate(policy, MentionGateInput{
		ChannelID:     "C123",
		UserID:        "U1",
		BotUserID:     "UBOT",
		Text:          "follow up",
		Timestamp:     "1711111111.000200",
		ThreadTS:      "1711111111.000100",
		ActiveSession: true,
	})
	if !decision.Process {
		t.Fatal("Process = false, want active-session thread reply allowed")
	}
}

func TestStrictMention_DoesNotRememberMentionedThread(t *testing.T) {
	policy := ResolveMentionPolicy(MentionPolicyConfig{StrictMention: true})
	decision := EvaluateMentionGate(policy, MentionGateInput{
		ChannelID: "C123",
		UserID:    "U1",
		BotUserID: "UBOT",
		Text:      "<@UBOT> hello",
		Timestamp: "1711111111.000300",
		ThreadTS:  "1711111111.000100",
	})
	if !decision.Process {
		t.Fatal("Process = false, want direct mention allowed")
	}
	if decision.RememberThread {
		t.Fatal("RememberThread = true in strict mode")
	}
}

func TestMentionGate_ChannelAdapterDropsUnmentionedSubmit(t *testing.T) {
	ch := NewChannel(newMockClient(), nil, ChannelConfig{RequireMention: true})
	ch.selfUserID = "UBOT"
	_, ok := ch.toInboundEvent(Event{
		ChannelID: "C123",
		UserID:    "U1",
		Text:      "hello",
		Timestamp: "1711111111.000400",
	})
	if ok {
		t.Fatal("toInboundEvent ok = true, want unmentioned channel submit dropped")
	}
}

func TestMentionGate_ChannelAdapterStripsMentionAndRemembersThread(t *testing.T) {
	ch := NewChannel(newMockClient(), nil, ChannelConfig{RequireMention: true})
	ch.selfUserID = "UBOT"
	ev, ok := ch.toInboundEvent(Event{
		ChannelID: "C123",
		UserID:    "U1",
		Text:      "<@UBOT> hello",
		Timestamp: "1711111111.000401",
		ThreadTS:  "1711111111.000300",
	})
	if !ok {
		t.Fatal("toInboundEvent ok = false, want mentioned submit")
	}
	if ev.Text != "hello" {
		t.Fatalf("Text = %q, want hello", ev.Text)
	}
	if !ch.threadMentioned("1711111111.000300") {
		t.Fatal("mentioned thread was not remembered")
	}
}

func TestMentionGate_ChannelAdapterReparsesMentionedCommand(t *testing.T) {
	ch := NewChannel(newMockClient(), nil, ChannelConfig{RequireMention: true})
	ch.selfUserID = "UBOT"
	ev, ok := ch.toInboundEvent(Event{
		ChannelID: "C123",
		UserID:    "U1",
		Text:      "<@UBOT> /start",
		Timestamp: "1711111111.000402",
	})
	if !ok {
		t.Fatal("toInboundEvent ok = false, want mentioned command")
	}
	if ev.Kind != gateway.EventStart {
		t.Fatalf("Kind = %v, want EventStart", ev.Kind)
	}
}

func hasMentionPolicyEvidence(items []MentionPolicyEvidence, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}
