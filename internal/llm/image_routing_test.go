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

func TestExtractImageRefs_LocalURLDedupAndSkipsCode(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "shot.png")
	if err := os.WriteFile(pngPath, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	missing := filepath.Join(dir, "missing.png")

	paths, urls := ExtractImageRefs("Real screenshot: " + pngPath + " and again " + pngPath + ".\n" +
		"Missing local file should not attach: " + missing + "\n" +
		"Design URL: https://example.com/mock/v3.png). Duplicate https://example.com/mock/v3.png\n" +
		"Inline code: `https://example.com/example.png`\n" +
		"```\n" +
		"code-only path " + pngPath + " and https://example.com/code.png\n" +
		"```\n")

	if len(paths) != 1 || paths[0] != pngPath {
		t.Fatalf("paths = %v, want [%q]", paths, pngPath)
	}
	wantURLs := []string{"https://example.com/mock/v3.png"}
	if len(urls) != len(wantURLs) || urls[0] != wantURLs[0] {
		t.Fatalf("urls = %v, want %v", urls, wantURLs)
	}
}

func TestExtractImageRefs_HomeRelativePathExpands(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	homeImage := filepath.Join(dir, "home.png")
	if err := os.WriteFile(homeImage, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	paths, urls := ExtractImageRefs("please inspect ~/home.png")

	if len(paths) != 1 || paths[0] != homeImage {
		t.Fatalf("paths = %v, want expanded home path %q", paths, homeImage)
	}
	if len(urls) != 0 {
		t.Fatalf("urls = %v, want none", urls)
	}
}

func TestBuildNativeImageContentParts_URLImageParts(t *testing.T) {
	url := "https://example.com/target.jpg"

	parts, skipped := BuildNativeImageContentParts("work kanban task 7", nil, []string{url})

	if len(skipped) != 0 {
		t.Fatalf("skipped = %v, want none", skipped)
	}
	if len(parts) != 2 {
		t.Fatalf("parts len = %d, want text plus URL image", len(parts))
	}
	if parts[0].Type != "text" || !strings.Contains(parts[0].Text, "[Image attached: "+url+"]") {
		t.Fatalf("part[0] = %+v, want URL hint", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL != url {
		t.Fatalf("part[1] = %+v, want remote image_url part", parts[1])
	}
}

func TestBuildNativeImageContentParts_SniffsMIMEFromBytes(t *testing.T) {
	dir := t.TempDir()
	mislabeled := filepath.Join(dir, "mislabeled.jpg")
	if err := os.WriteFile(mislabeled, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	parts, skipped := BuildNativeImageContentParts("inspect", []string{mislabeled})

	if len(skipped) != 0 {
		t.Fatalf("skipped = %v, want none", skipped)
	}
	if len(parts) != 2 {
		t.Fatalf("parts len = %d, want text plus image", len(parts))
	}
	if !strings.HasPrefix(parts[1].ImageURL, "data:image/png;base64,") {
		t.Fatalf("part[1].ImageURL = %q, want sniffed PNG data URL", parts[1].ImageURL)
	}
}
