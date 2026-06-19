package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestGatewayCommandDispatchCompressCallsKernel(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventCompress, Text: "/compress billing history"}); err != nil {
		t.Fatalf("handleInbound: %v", err)
	}
	if len(fk.compressFocus) != 1 || fk.compressFocus[0] != "billing history" {
		t.Fatalf("compress focus = %#v, want billing history", fk.compressFocus)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "Session compressed") || !strings.Contains(sent[0].Text, "billing history") {
		t.Fatalf("compress reply = %#v", sent)
	}
}

func TestGatewayCommandDispatchCompressDisabled(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{compressErr: kernel.ErrCompressionUnavailable}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventCompress, Text: "/compress"}); err != nil {
		t.Fatalf("handleInbound: %v", err)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "compression is disabled") {
		t.Fatalf("compress disabled reply = %#v", sent)
	}
}

func TestGatewayCommandDispatchCompressRedactsErrors(t *testing.T) {
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{compressErr: errors.New("summary failed token=plain-secret")}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventCompress, Text: "/compress"}); err != nil {
		t.Fatalf("handleInbound: %v", err)
	}
	sent := ch.sentSnapshot()
	if len(sent) != 1 || !strings.Contains(sent[0].Text, "[redacted]") || strings.Contains(sent[0].Text, "plain-secret") {
		t.Fatalf("compress error reply = %#v", sent)
	}
}

func TestGatewayCommandDispatchHandlesDirectAndSlashBusyCommand(t *testing.T) {
	for _, tc := range []struct {
		name string
		ev   InboundEvent
	}{
		{name: "direct event", ev: InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventBusy, Text: "/busy queue"}},
		{name: "slash submit", ev: InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventSubmit, Text: "/busy queue"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ch := newFakeChannel("telegram")
			m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, &fakeKernel{}, slog.Default())
			if err := m.Register(ch); err != nil {
				t.Fatalf("Register: %v", err)
			}

			if err := m.handleInbound(context.Background(), tc.ev); err != nil {
				t.Fatalf("handleInbound: %v", err)
			}

			sent := ch.sentSnapshot()
			if len(sent) != 1 {
				t.Fatalf("sent messages = %d, want 1: %+v", len(sent), sent)
			}
			if !strings.Contains(sent[0].Text, "Busy input mode set to **queue**") {
				t.Fatalf("busy reply = %q, want queue confirmation", sent[0].Text)
			}
		})
	}
}
