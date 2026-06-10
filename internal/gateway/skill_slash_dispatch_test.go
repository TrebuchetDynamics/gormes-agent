package gateway

import (
	"context"
	"log/slog"
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
	assertContainsAll(t, submit.Text,
		`[IMPORTANT: The user has invoked the "review-skill" skill`,
		"Review the requested code carefully.",
		"The user has provided the following instruction alongside the skill invocation: inspect src",
		"[Runtime note: telegram]",
	)
	if strings.HasPrefix(strings.TrimSpace(submit.Text), "/review-skill") {
		t.Fatalf("raw slash leaked into kernel submit: %q", submit.Text)
	}
	for _, sent := range ch.sentSnapshot() {
		if strings.Contains(strings.ToLower(sent.Text), "unknown command") {
			t.Fatalf("dynamic skill was treated as unknown: %#v", sent)
		}
	}
}

func TestManager_DynamicSkillSlashSubmitRejectsMixedDoubleSlashEscape(t *testing.T) {
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

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "/／review-skill inspect src"})

	waitFor(t, 300*time.Millisecond, func() bool { return len(ch.sentSnapshot()) == 1 || len(fk.submitsSnapshot()) > 0 })
	if submits := fk.submitsSnapshot(); len(submits) != 0 {
		t.Fatalf("mixed double slash expanded/submitted to kernel: %#v", submits)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(strings.ToLower(sent[0].Text), "unknown command") {
		t.Fatalf("mixed double slash response = %#v, want unknown command guidance", sent)
	}
}

func TestManager_DynamicSkillSlashSubmitAcceptsFullwidthSlash(t *testing.T) {
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

	ch.pushInbound(InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "／review-skill inspect src"})

	waitFor(t, 300*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
	submit := fk.submitsSnapshot()[0]
	if !strings.Contains(submit.Text, `invoked the "review-skill" skill`) || !strings.Contains(submit.Text, "inspect src") {
		t.Fatalf("fullwidth slash skill did not expand before kernel:\n%s", submit.Text)
	}
	if strings.Contains(submit.Text, "／review-skill") {
		t.Fatalf("fullwidth raw slash invocation leaked into expanded submit:\n%s", submit.Text)
	}
}

func TestManager_DynamicSkillFullwidthSlashFromUnknownEventUsesPreservedRawText(t *testing.T) {
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

	ch.pushInbound(InboundEvent{Platform: "discord", ChatID: "C42", UserID: "u", MsgID: "m1", Kind: EventUnknown, Text: "／ops-skill now"})

	waitFor(t, 300*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 || len(ch.sentSnapshot()) > 0 })
	if sent := ch.sentSnapshot(); len(sent) > 0 {
		t.Fatalf("fullwidth unknown-event skill was treated as unknown: %#v", sent)
	}
	submits := fk.submitsSnapshot()
	if len(submits) != 1 || !strings.Contains(submits[0].Text, `invoked the "ops-skill" skill`) {
		t.Fatalf("fullwidth unknown-event skill did not expand before kernel: %#v", submits)
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
	return writeActiveSkill(t, name, "Invoke "+name, body)
}
