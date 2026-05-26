package gateway

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

func TestManagerReloadSkillsCommandRefreshesAdaptersAndDoesNotLeak(t *testing.T) {
	root := writeReloadSkillsCommandSkill(t, "ops-skill", "Operate safely.")
	ch := &refreshableGatewayChannel{fakeChannel: newFakeChannel("discord")}
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"discord": "C42"},
		SkillRuntime: skills.NewRuntime(root, 8*1024, 5, ""),
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "discord", ChatID: "C42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "start long turn"})
	waitFor(t, 300*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })

	ch.pushInbound(InboundEvent{Platform: "discord", ChatID: "C42", UserID: "u", MsgID: "m2", Kind: EventSubmit, Text: "/reload_skills"})

	waitFor(t, 300*time.Millisecond, func() bool {
		for _, sent := range ch.sentSnapshot() {
			if strings.Contains(sent.Text, "Skills Reloaded") && strings.Contains(sent.Text, "1 skill(s) available") && strings.Contains(sent.Text, "discord: refreshed") {
				return true
			}
		}
		return false
	})
	if ch.calls != 1 {
		t.Fatalf("refresh calls = %d, want 1", ch.calls)
	}
	if got := len(fk.submitsSnapshot()); got != 1 {
		t.Fatalf("kernel submits = %d, want only original active turn", got)
	}
}

func TestManagerReloadSkillsCommandReportsScanAndAdapterErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "active"), []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile active: %v", err)
	}
	ch := &refreshableGatewayChannel{fakeChannel: newFakeChannel("discord"), err: errChannelScopeRefresh}
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"discord": "C42"},
		SkillRuntime: skills.NewRuntime(root, 8*1024, 5, ""),
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "discord", ChatID: "C42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "/reload-skills"})

	waitFor(t, 300*time.Millisecond, func() bool {
		for _, sent := range ch.sentSnapshot() {
			if strings.Contains(sent.Text, "Skills reload degraded") && strings.Contains(sent.Text, "discord: refresh error") {
				return true
			}
		}
		return false
	})
	if got := len(fk.submitsSnapshot()); got != 0 {
		t.Fatalf("kernel submits = %d, want no model submission", got)
	}
}

func writeReloadSkillsCommandSkill(t *testing.T, name, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "active", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	raw := "---\nname: " + name + "\ndescription: Invoke " + name + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile SKILL.md: %v", err)
	}
	return root
}
