package gateway

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

func TestManagerCommandsCommandIncludesSkillsAndDoesNotLeakDuringActiveTurn(t *testing.T) {
	root := writeCommandsCatalogSkill(t, "ops-skill", "Operate safely.")
	ch := newFakeChannel("slack")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"slack": "C42"},
		SkillRuntime: skills.NewRuntime(root, 8*1024, 5, ""),
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "slack", ChatID: "C42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "start long turn"})
	waitFor(t, 300*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })

	ch.pushInbound(InboundEvent{Platform: "slack", ChatID: "C42", UserID: "u", MsgID: "m2", Kind: EventSubmit, Text: "/commands 99"})

	waitFor(t, 300*time.Millisecond, func() bool {
		for _, sent := range ch.sentSnapshot() {
			if strings.Contains(sent.Text, "Available commands") && strings.Contains(sent.Text, "`/ops-skill`") {
				return true
			}
		}
		return false
	})
	if got := len(fk.submitsSnapshot()); got != 1 {
		t.Fatalf("kernel submits = %d, want only original active turn", got)
	}
}

func TestManagerCommandsCommandHandlesParsedGatewayEvent(t *testing.T) {
	root := writeCommandsCatalogSkill(t, "ops-skill", "Operate safely.")
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SkillRuntime: skills.NewRuntime(root, 8*1024, 5, ""),
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m1", Kind: EventCommands, Text: "/commands 99"})

	waitFor(t, 300*time.Millisecond, func() bool {
		for _, sent := range ch.sentSnapshot() {
			if strings.Contains(sent.Text, "Available commands") && strings.Contains(sent.Text, "`/ops-skill`") {
				return true
			}
		}
		return false
	})
	if got := len(fk.submitsSnapshot()); got != 0 {
		t.Fatalf("kernel submits = %d, want no model submission for parsed /commands", got)
	}
}

func TestManagerCommandsCommandDegradesWhenSkillScanFails(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "active"), []byte("not a dir"), 0o644); err != nil {
		t.Fatalf("WriteFile active: %v", err)
	}
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SkillRuntime: skills.NewRuntime(root, 8*1024, 5, ""),
	}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "/commands"})

	waitFor(t, 300*time.Millisecond, func() bool {
		for _, sent := range ch.sentSnapshot() {
			if strings.Contains(sent.Text, "Available commands") && strings.Contains(sent.Text, "`/help`") {
				return true
			}
		}
		return false
	})
	if got := len(fk.submitsSnapshot()); got != 0 {
		t.Fatalf("kernel submits = %d, want no model submission for degraded /commands", got)
	}
}

func writeCommandsCatalogSkill(t *testing.T, name, body string) string {
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
