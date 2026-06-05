package gateway

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

func TestManager_ChannelScopeAutoSkillsAndPrompt(t *testing.T) {
	root := t.TempDir()
	writeChannelScopeSkill(t, root, "research", "Research-only instructions.")
	writeChannelScopeSkill(t, root, "review", "Review-only instructions.")

	ch := newFakeChannel("discord")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"discord": "C123"},
		SkillRuntime: skills.NewRuntime(root, 8*1024, 5, ""),
	}, fk, nil)
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{
		Platform:      "discord",
		ChatID:        "C123",
		UserID:        "U1",
		MsgID:         "M1",
		Kind:          EventSubmit,
		Text:          "summarize this",
		AutoSkills:    []string{"research", "review"},
		ChannelPrompt: "Use the channel's research workflow.",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0]
	if !strings.Contains(got.SessionContext, "## Channel Prompt\nUse the channel's research workflow.") {
		t.Fatalf("SessionContext missing channel prompt:\n%s", got.SessionContext)
	}
	if !strings.Contains(got.SessionContext, "## Current Session Context") {
		t.Fatalf("SessionContext missing base session block:\n%s", got.SessionContext)
	}
	if strings.Index(got.SessionContext, "## Channel Prompt") > strings.Index(got.SessionContext, "## Current Session Context") {
		t.Fatalf("channel prompt should precede session metadata:\n%s", got.SessionContext)
	}
	if got.Skills == nil {
		t.Fatal("kernel submit Skills is nil, want channel-scoped provider")
	}
	block, names, err := got.Skills.BuildSkillBlock(ctx, "summarize this")
	if err != nil {
		t.Fatalf("BuildSkillBlock: %v", err)
	}
	if strings.Join(names, ",") != "research,review" {
		t.Fatalf("skill names = %#v, want research/review", names)
	}
	if !strings.Contains(block, "Research-only instructions.") || !strings.Contains(block, "Review-only instructions.") {
		t.Fatalf("skill block missing channel skills:\n%s", block)
	}
}

func TestReloadSkillsRefreshesSkillGroupAdapters(t *testing.T) {
	refreshable := &refreshableGatewayChannel{fakeChannel: newFakeChannel("discord")}
	plain := newFakeChannel("slack")
	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, nil)
	if err := m.Register(refreshable); err != nil {
		t.Fatalf("Register refreshable: %v", err)
	}
	if err := m.Register(plain); err != nil {
		t.Fatalf("Register plain: %v", err)
	}

	results := m.RefreshSkillGroups(context.Background())
	if len(results) != 1 {
		t.Fatalf("RefreshSkillGroups results = %#v, want one refreshable adapter", results)
	}
	if results[0].Channel != "discord" || results[0].Count != 2 || results[0].Hidden != 1 || results[0].Error != "" {
		t.Fatalf("refresh result = %+v, want discord count=2 hidden=1", results[0])
	}
	if refreshable.calls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refreshable.calls)
	}

	refreshable.err = errChannelScopeRefresh
	results = m.RefreshSkillGroups(context.Background())
	if len(results) != 1 || results[0].Error == "" {
		t.Fatalf("RefreshSkillGroups error results = %#v, want redacted error evidence", results)
	}
	if refreshable.calls != 2 {
		t.Fatalf("refresh calls after error = %d, want 2", refreshable.calls)
	}
}

type refreshableGatewayChannel struct {
	*fakeChannel
	calls int
	err   error
}

var errChannelScopeRefresh = errString("collector failed")

func (c *refreshableGatewayChannel) RefreshSkillGroup(context.Context) (SkillGroupRefreshResult, error) {
	c.calls++
	if c.err != nil {
		return SkillGroupRefreshResult{Channel: c.Name(), Count: 2, Hidden: 1}, c.err
	}
	return SkillGroupRefreshResult{Channel: c.Name(), Count: 2, Hidden: 1}, nil
}

type errString string

func (e errString) Error() string { return string(e) }

func writeChannelScopeSkill(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "active", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	raw := "---\nname: " + name + "\ndescription: test skill\n---\n\n# " + name + "\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}
