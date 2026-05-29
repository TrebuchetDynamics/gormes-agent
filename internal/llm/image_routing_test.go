package llm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageInputRouting_AutoNativeWhenModelSupportsVision(t *testing.T) {
	cfg := ImageRoutingConfig{
		Mode:                  ImageInputModeAuto,
		ModelVisionCapability: ModelCapabilitySupported,
	}
	if got := DecideImageInputMode(cfg); got != ImageInputModeNative {
		t.Fatalf("DecideImageInputMode = %q, want %q", got, ImageInputModeNative)
	}
}

func TestImageInputRouting_AutoTextWhenAuxVisionConfigured(t *testing.T) {
	cfg := ImageRoutingConfig{
		Mode:                  ImageInputModeAuto,
		ModelVisionCapability: ModelCapabilitySupported,
		AuxiliaryVision: AuxiliaryVisionConfig{
			Provider: "openai",
			Model:    "gpt-4o-mini",
		},
	}
	if got := DecideImageInputMode(cfg); got != ImageInputModeText {
		t.Fatalf("DecideImageInputMode = %q, want %q (aux vision must override native)", got, ImageInputModeText)
	}
}

func TestImageInputRouting_AutoTextForUnknownOrNonVisionModel(t *testing.T) {
	for _, flag := range []ModelCapabilityFlag{ModelCapabilityUnknown, ModelCapabilityUnsupported, ""} {
		cfg := ImageRoutingConfig{Mode: ImageInputModeAuto, ModelVisionCapability: flag}
		if got := DecideImageInputMode(cfg); got != ImageInputModeText {
			t.Fatalf("vision=%q: DecideImageInputMode = %q, want %q", flag, got, ImageInputModeText)
		}
	}
}

func TestBuildNativeImageContentParts_TextAndImages(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "img.png")
	if err := os.WriteFile(pngPath, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	missing := filepath.Join(dir, "missing.jpg")

	parts, skipped := BuildNativeImageContentParts("hello", []string{pngPath, missing})

	if len(parts) != 2 {
		t.Fatalf("parts len = %d, want 2; parts=%+v", len(parts), parts)
	}
	wantText := "hello\n\n[Image attached at: " + pngPath + "]"
	if parts[0].Type != "text" || parts[0].Text != wantText {
		t.Fatalf("part[0] = %+v, want text:%q", parts[0], wantText)
	}
	if parts[1].Type != "image_url" {
		t.Fatalf("part[1].Type = %q, want image_url", parts[1].Type)
	}
	if !strings.HasPrefix(parts[1].ImageURL, "data:image/png;base64,") {
		t.Fatalf("part[1].ImageURL = %q, want data URL prefix", parts[1].ImageURL)
	}
	if len(skipped) != 1 || skipped[0] != missing {
		t.Fatalf("skipped = %v, want one missing path %q", skipped, missing)
	}
}

func TestBuildNativeImageContentParts_DefaultPrompt(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "img.png")
	if err := os.WriteFile(pngPath, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	parts, _ := BuildNativeImageContentParts("", []string{pngPath})

	if len(parts) != 2 {
		t.Fatalf("parts len = %d, want 2 (default prompt + image)", len(parts))
	}
	wantText := defaultImagePromptText + "\n\n[Image attached at: " + pngPath + "]"
	if parts[0].Type != "text" || parts[0].Text != wantText {
		t.Fatalf("part[0] = %+v, want text:%q", parts[0], wantText)
	}
	if parts[1].Type != "image_url" {
		t.Fatalf("part[1].Type = %q, want image_url", parts[1].Type)
	}
}

func TestBuildNativeImageContentParts_PathHintOnlyForReadableImages(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "img.png")
	if err := os.WriteFile(pngPath, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	missing := filepath.Join(dir, "missing.jpg")

	parts, skipped := BuildNativeImageContentParts("see attached", []string{pngPath, missing})

	if len(skipped) != 1 || skipped[0] != missing {
		t.Fatalf("skipped = %v, want one missing path %q", skipped, missing)
	}
	if len(parts) != 2 {
		t.Fatalf("parts len = %d, want text plus one image", len(parts))
	}
	if strings.Count(parts[0].Text, "[Image attached at:") != 1 {
		t.Fatalf("text part = %q, want one image path hint", parts[0].Text)
	}
	if !strings.Contains(parts[0].Text, pngPath) {
		t.Fatalf("text part = %q, want readable path %q", parts[0].Text, pngPath)
	}
	if strings.Contains(parts[0].Text, missing) {
		t.Fatalf("text part = %q, must not advertise missing path %q", parts[0].Text, missing)
	}
}
