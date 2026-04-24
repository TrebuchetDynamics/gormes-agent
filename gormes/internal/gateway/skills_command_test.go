package gateway

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/skills"
)

type fakeSkillsBrowser struct {
	mu      sync.Mutex
	calls   []int
	view    skills.BrowseView
	text    string
	err     error
	called  bool
	perPage int
}

func (f *fakeSkillsBrowser) Browse(_ context.Context, page, perPage int) (skills.BrowseView, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.called = true
	f.calls = append(f.calls, page)
	f.perPage = perPage
	if f.err != nil {
		return skills.BrowseView{}, "", f.err
	}
	return f.view, f.text, nil
}

func (f *fakeSkillsBrowser) snapshot() ([]int, bool, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]int, len(f.calls))
	copy(out, f.calls)
	return out, f.called, f.perPage
}

func TestCommandRegistry_ContainsSkills(t *testing.T) {
	cmd, ok := ResolveCommand("/skills")
	if !ok {
		t.Fatal("ResolveCommand(/skills) = false, want true")
	}
	if cmd.Kind != EventSkills {
		t.Fatalf("cmd.Kind = %v, want EventSkills", cmd.Kind)
	}
}

func TestParseInboundText_SkillsWithPageArgument(t *testing.T) {
	kind, body := ParseInboundText("/skills")
	if kind != EventSkills {
		t.Fatalf("kind = %v, want EventSkills", kind)
	}
	if body != "" {
		t.Fatalf("body = %q, want empty", body)
	}

	kind, body = ParseInboundText("/skills 2")
	if kind != EventSkills {
		t.Fatalf("kind with arg = %v, want EventSkills", kind)
	}
	if body != "2" {
		t.Fatalf("body = %q, want \"2\"", body)
	}
}

func TestManager_Inbound_SkillsCommandSendsFormattedBrowse(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	browser := &fakeSkillsBrowser{
		text: "SKILLS_BROWSER_SENTINEL\nPage 1/1",
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:   map[string]string{"telegram": "42"},
		SkillsBrowser:  browser,
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m",
		Kind: EventSkills,
		Text: "",
	})

	waitFor(t, 500*time.Millisecond, func() bool {
		_, called, _ := browser.snapshot()
		return called
	})
	pages, _, perPage := browser.snapshot()
	if len(pages) != 1 || pages[0] != 1 {
		t.Fatalf("browser pages = %v, want [1]", pages)
	}
	if perPage <= 0 {
		t.Fatalf("perPage = %d, want positive default", perPage)
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		for _, sent := range tg.sentSnapshot() {
			if strings.Contains(sent.Text, "SKILLS_BROWSER_SENTINEL") {
				return true
			}
		}
		return false
	})
}

func TestManager_Inbound_SkillsCommandParsesPageArg(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	browser := &fakeSkillsBrowser{text: "OK"}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:  map[string]string{"telegram": "42"},
		SkillsBrowser: browser,
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42",
		Kind: EventSkills,
		Text: "3",
	})

	waitFor(t, 500*time.Millisecond, func() bool {
		pages, _, _ := browser.snapshot()
		return len(pages) == 1 && pages[0] == 3
	})
}

func TestManager_Inbound_SkillsCommand_NoBrowserSendsUsage(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42",
		Kind: EventSkills,
	})

	waitFor(t, 500*time.Millisecond, func() bool {
		for _, sent := range tg.sentSnapshot() {
			if strings.Contains(strings.ToLower(sent.Text), "skills browser is not configured") {
				return true
			}
		}
		return false
	})
}
