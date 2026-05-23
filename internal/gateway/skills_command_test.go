package gateway

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

func TestHandleSkillsCommandSearchAndBrowseHubReadModel(t *testing.T) {
	providers := []skills.HubRegistryProvider{skills.NewInMemoryRegistryProvider([]skills.HubSearchResult{
		{
			Name:        "planner-pro",
			Description: "Plan safely with API_KEY=sk-live-secret and local path /home/xel/.gormes/secret.txt before editing.",
			Source:      "hermes-index",
			InstallID:   "github/openai/planner-pro",
			Score:       0.95,
			TrustLevel:  "community",
			Tags:        []string{"planning", "safe"},
		},
		{
			Name:        "browser-helper",
			Description: "Browse pages without mutating installed skills.",
			Source:      "skills.sh",
			InstallID:   "skills/browser-helper",
			Score:       0.50,
			TrustLevel:  "trusted",
		},
	}, nil)}
	opts := SkillsCommandOptions{HubProviders: providers, PageSize: 1, MaxDescriptionRunes: 90}

	search := HandleSkillsCommandWithOptions(context.Background(), "/skills search planner", opts)
	for _, want := range []string{
		"Skill Hub Search",
		"query: planner",
		"1 result(s)",
		"planner-pro",
		"source=hermes-index",
		"trust=community",
		"planning,safe",
	} {
		if !strings.Contains(search, want) {
			t.Fatalf("/skills search output missing %q:\n%s", want, search)
		}
	}
	for _, forbidden := range []string{"sk-live-secret", "/home/xel/.gormes", "secret.txt", "SKILL.md"} {
		if strings.Contains(search, forbidden) {
			t.Fatalf("/skills search leaked forbidden content %q:\n%s", forbidden, search)
		}
	}

	browse := HandleSkillsCommandWithOptions(context.Background(), "/skills browse --page 2 --page-size 1", opts)
	for _, want := range []string{
		"Skill Hub Browse",
		"page 2/2",
		"2 total",
		"browser-helper",
		"source=skills.sh",
		"trust=trusted",
	} {
		if !strings.Contains(browse, want) {
			t.Fatalf("/skills browse output missing %q:\n%s", want, browse)
		}
	}
	if strings.Contains(browse, "planner-pro") {
		t.Fatalf("/skills browse page 2 should not include page 1 result:\n%s", browse)
	}
}

func TestHandleSkillsCommandTelegramUsesPlainTextReply(t *testing.T) {
	kind, body := ParseInboundText("/skills search planner")
	if kind != EventSkills || body != "/skills search planner" {
		t.Fatalf("ParseInboundText(/skills search planner) = (%v, %q), want EventSkills with body", kind, body)
	}
	ch := newFakePlainSkillsChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "u",
		MsgID:    "m-skills-search",
		Kind:     kind,
		Text:     body,
	}); err != nil {
		t.Fatalf("handleInbound(/skills search): %v", err)
	}

	plain := ch.plainSnapshot()
	if len(plain) != 1 {
		t.Fatalf("plain skills sends = %d, want 1", len(plain))
	}
	if !strings.Contains(strings.ToLower(plain[0].Text), "skill hub search") {
		t.Fatalf("plain skills reply missing search evidence:\n%s", plain[0].Text)
	}
	if sent := ch.sentSnapshot(); len(sent) != 0 {
		t.Fatalf("skills command used Markdown Send path, sent=%+v", sent)
	}
	if submits := fk.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("/skills command must not submit to kernel, got submits=%+v", submits)
	}
}

func TestHandleSkillsCommandMutatingActionsUnavailable(t *testing.T) {
	for _, action := range []string{"install", "edit", "disable", "review"} {
		t.Run(action, func(t *testing.T) {
			out := HandleSkillsCommandWithOptions(context.Background(), "/skills "+action+" planner-pro", SkillsCommandOptions{
				HubProviders: []skills.HubRegistryProvider{skills.NewInMemoryRegistryProvider([]skills.HubSearchResult{{Name: "planner-pro", InstallID: "planner-pro"}}, nil)},
			})
			for _, want := range []string{"skills_manage_unavailable", action, "row 6.F", "read-only", "no skill store was changed"} {
				if !strings.Contains(out, want) {
					t.Fatalf("/skills %s output missing %q:\n%s", action, want, out)
				}
			}
		})
	}
}

type fakePlainSkillsChannel struct {
	*fakeChannel
	plainMu sync.Mutex
	plain   []fakeSent
}

func newFakePlainSkillsChannel(name string) *fakePlainSkillsChannel {
	return &fakePlainSkillsChannel{fakeChannel: newFakeChannel(name)}
}

func (f *fakePlainSkillsChannel) SendPlain(_ context.Context, chatID, text string) (string, error) {
	f.plainMu.Lock()
	defer f.plainMu.Unlock()
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.plain = append(f.plain, fakeSent{ChatID: chatID, Text: text, MsgID: id})
	return id, nil
}

func (f *fakePlainSkillsChannel) SendPlainReply(_ context.Context, chatID, replyToMsgID, text string) (string, error) {
	f.plainMu.Lock()
	defer f.plainMu.Unlock()
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.plain = append(f.plain, fakeSent{ChatID: chatID, Text: text, MsgID: id + ":reply:" + replyToMsgID})
	return id, nil
}

func (f *fakePlainSkillsChannel) plainSnapshot() []fakeSent {
	f.plainMu.Lock()
	defer f.plainMu.Unlock()
	out := make([]fakeSent, len(f.plain))
	copy(out, f.plain)
	return out
}
