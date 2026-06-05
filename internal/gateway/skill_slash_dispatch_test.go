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
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestManager_DynamicSkillSlashSubmitExpandsBeforeKernel(t *testing.T) {
	root := writeGatewaySkillSlashTestSkill(t, "review-skill", "Review the requested code carefully.")
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

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "/review-skill inspect src"})

	waitFor(t, 300*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
	submit := fk.submitsSnapshot()[0]
	if submit.Kind != kernel.PlatformEventSubmit {
		t.Fatalf("submit kind = %v, want submit", submit.Kind)
	}
	for _, want := range []string{
		`[IMPORTANT: The user has invoked the "review-skill" skill`,
		"Review the requested code carefully.",
		"The user has provided the following instruction alongside the skill invocation: inspect src",
		"[Runtime note: telegram]",
	} {
		if !strings.Contains(submit.Text, want) {
			t.Fatalf("submit text missing %q:\n%s", want, submit.Text)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(submit.Text), "/review-skill") {
		t.Fatalf("raw slash leaked into kernel submit: %q", submit.Text)
	}
	for _, sent := range ch.sentSnapshot() {
		if strings.Contains(strings.ToLower(sent.Text), "unknown command") {
			t.Fatalf("dynamic skill was treated as unknown: %#v", sent)
		}
	}
}

func TestManager_DynamicSkillSlashFromUnknownEventUsesPreservedRawText(t *testing.T) {
	root := writeGatewaySkillSlashTestSkill(t, "ops-skill", "Operate safely.")
	ch := newFakeChannel("discord")
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

	ch.pushInbound(InboundEvent{Platform: "discord", ChatID: "C42", UserID: "u", MsgID: "m1", Kind: EventUnknown, Text: "/ops-skill now"})

	waitFor(t, 300*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
	submit := fk.submitsSnapshot()[0]
	if !strings.Contains(submit.Text, `invoked the "ops-skill" skill`) || !strings.Contains(submit.Text, "Operate safely.") {
		t.Fatalf("submit text did not expand skill:\n%s", submit.Text)
	}
	if strings.Contains(submit.Text, "/ops-skill now") {
		t.Fatalf("raw slash invocation leaked into expanded submit:\n%s", submit.Text)
	}
}

func writeGatewaySkillSlashTestSkill(t *testing.T, name, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "active", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	raw := "---\nname: " + name + "\ndescription: Invoke " + name + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return root
}
