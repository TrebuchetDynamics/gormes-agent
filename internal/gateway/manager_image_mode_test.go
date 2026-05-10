package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestManager_ImageModeAutoVisionModelKeepsNativeContentParts(t *testing.T) {
	ev := submitManagerImageModeTurn(t, ManagerConfig{
		LiveTurnActiveProvider: func() string { return "openai" },
		LiveTurnActiveModel:    func() string { return "gpt-4o-mini" },
	})

	if len(ev.ContentParts) == 0 {
		t.Fatal("ContentParts is empty; want native image content for a vision-capable model")
	}
	if !hasImageURLContentPart(ev.ContentParts) {
		t.Fatalf("ContentParts = %+v, want image_url content part", ev.ContentParts)
	}
	if strings.Contains(ev.Text, visionPreAnalysisUnavailableMarker) {
		t.Fatalf("Text = %q, native mode must not add degraded marker", ev.Text)
	}
}

func TestManager_ImageModeTextSuppressesImageURLAndMarksDegraded(t *testing.T) {
	ev := submitManagerImageModeTurn(t, ManagerConfig{
		ImageInputMode:         hermes.ImageInputModeText,
		LiveTurnActiveProvider: func() string { return "openai" },
		LiveTurnActiveModel:    func() string { return "gpt-4o-mini" },
	})

	if len(ev.ContentParts) != 0 {
		t.Fatalf("ContentParts = %+v, want no native image parts in text mode", ev.ContentParts)
	}
	if !strings.Contains(ev.Text, visionPreAnalysisUnavailableMarker) {
		t.Fatalf("Text = %q, want typed degraded marker", ev.Text)
	}
	if !strings.Contains(ev.Text, "Attachments:") {
		t.Fatalf("Text = %q, want attachment marker preserved", ev.Text)
	}
}

func TestManager_ImageModeAutoUnknownModelSuppressesImageURLAndMarksDegraded(t *testing.T) {
	ev := submitManagerImageModeTurn(t, ManagerConfig{
		LiveTurnActiveProvider: func() string { return "ollama" },
		LiveTurnActiveModel:    func() string { return "text-only-local" },
	})

	if len(ev.ContentParts) != 0 {
		t.Fatalf("ContentParts = %+v, want no image_url parts for unknown/text-only model metadata", ev.ContentParts)
	}
	if !strings.Contains(ev.Text, visionPreAnalysisUnavailableMarker) {
		t.Fatalf("Text = %q, want typed degraded marker", ev.Text)
	}
}

func TestManager_ImageModeAuxiliaryVisionForcesText(t *testing.T) {
	ev := submitManagerImageModeTurn(t, ManagerConfig{
		AuxiliaryVision: hermes.AuxiliaryVisionConfig{
			Provider: "openai",
			Model:    "gpt-4o-mini",
		},
		LiveTurnActiveProvider: func() string { return "openai" },
		LiveTurnActiveModel:    func() string { return "gpt-4o-mini" },
	})

	if len(ev.ContentParts) != 0 {
		t.Fatalf("ContentParts = %+v, want no native image parts when auxiliary.vision overrides auto mode", ev.ContentParts)
	}
	if !strings.Contains(ev.Text, visionPreAnalysisUnavailableMarker) {
		t.Fatalf("Text = %q, want typed degraded marker", ev.Text)
	}
}

func submitManagerImageModeTurn(t *testing.T, cfg ManagerConfig) kernel.PlatformEvent {
	t.Helper()
	dir := t.TempDir()
	jpgPath := writeFixtureJPEGForManager(t, dir, "screenshot.jpg", 100, 200, 150)

	platform := "telegram"
	tg := newFakeChannel(platform)
	fk := &fakeKernel{}
	smap := session.NewMemMap()
	if err := smap.Put(context.Background(), platform+":42", "sess-stored"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	cfg.AllowedChats = map[string]string{platform: "42"}
	cfg.SessionMap = smap

	m := NewManagerWithSubmitter(cfg, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: platform,
		ChatID:   "42",
		UserID:   "7",
		MsgID:    "m-image-mode",
		Kind:     EventSubmit,
		Text:     "what is in this picture?",
		Attachments: []Attachment{
			{Kind: "photo", URL: jpgPath, MediaType: "image/jpeg", FileName: "screenshot.jpg"},
		},
	})

	waitFor(t, time.Second, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	submits := fk.submitsSnapshot()
	if len(submits) != 1 {
		t.Fatalf("kernel got %d submits, want 1", len(submits))
	}
	return submits[0]
}

func hasImageURLContentPart(parts []hermes.MessageContentPart) bool {
	for _, part := range parts {
		if part.Type == "image_url" {
			return true
		}
	}
	return false
}
