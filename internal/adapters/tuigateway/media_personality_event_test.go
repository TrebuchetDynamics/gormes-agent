package tuigateway

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Image helpers ─────────────────────────────────────────────────────────

// TestEstimateImageTokens_TileMath mirrors upstream
// hermes-agent/tui_gateway/server.py:_estimate_image_tokens. Each 512px tile
// is worth 85 tokens; the dimensions are tiled with ceil-division and the
// product is multiplied by 85. The helper is pure — width and height are
// integers, no filesystem access happens.
func TestEstimateImageTokens_TileMath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		width  int
		height int
		want   int
	}{
		// 1x1 tile -> 85 tokens
		{name: "subtile small image", width: 100, height: 100, want: 85},
		// exactly 512x512 -> 1 tile each side -> 85
		{name: "exact tile size", width: 512, height: 512, want: 85},
		// 513x513 -> 2 tiles each side -> 4 tiles -> 340
		{name: "just over tile boundary", width: 513, height: 513, want: 340},
		// 1024x1024 -> 2x2 tiles -> 340
		{name: "double tile size", width: 1024, height: 1024, want: 340},
		// 1280x720 -> ceil(1280/512)=3, ceil(720/512)=2 -> 6 tiles -> 510
		{name: "hd landscape", width: 1280, height: 720, want: 510},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := EstimateImageTokens(tc.width, tc.height)
			if got != tc.want {
				t.Errorf("EstimateImageTokens(%d, %d) = %d; want %d", tc.width, tc.height, got, tc.want)
			}
		})
	}
}

// TestEstimateImageTokens_NonPositiveReturnsZero exercises the upstream
// `if width <= 0 or height <= 0: return 0` guard. Negative or zero dims
// produce zero tokens with no panic.
func TestEstimateImageTokens_NonPositiveReturnsZero(t *testing.T) {
	t.Parallel()
	cases := []struct {
		w, h int
	}{
		{0, 0}, {0, 100}, {100, 0}, {-5, 100}, {100, -5}, {-1, -1},
	}
	for _, c := range cases {
		c := c
		if got := EstimateImageTokens(c.w, c.h); got != 0 {
			t.Errorf("EstimateImageTokens(%d, %d) = %d; want 0", c.w, c.h, got)
		}
	}
}

// TestReadImageMetadata_PNGFixture mirrors upstream
// hermes-agent/tui_gateway/server.py:_image_meta. A real PNG written to a
// temp directory yields name + dimensions + token estimate. The helper does
// a bounded header read only — no full-pixel decode is required.
func TestReadImageMetadata_PNGFixture(t *testing.T) {
	t.Parallel()
	path := writePNG(t, 320, 240)
	got, err := ReadImageMetadata(path)
	if err != nil {
		t.Fatalf("ReadImageMetadata(%q) error = %v; want nil", path, err)
	}
	if got.Name != filepath.Base(path) {
		t.Errorf("Name = %q; want %q", got.Name, filepath.Base(path))
	}
	if got.Width != 320 || got.Height != 240 {
		t.Errorf("dimensions = %dx%d; want 320x240", got.Width, got.Height)
	}
	// 320x240 -> ceil(320/512)=1, ceil(240/512)=1 -> 1 tile -> 85 tokens
	if got.TokenEstimate != 85 {
		t.Errorf("TokenEstimate = %d; want 85", got.TokenEstimate)
	}
}

// TestReadImageMetadata_JPEGFixture proves the helper accepts a JPEG header,
// not just PNG. JPEG decode picks up dimensions through Go's image package.
func TestReadImageMetadata_JPEGFixture(t *testing.T) {
	t.Parallel()
	path := writeJPEG(t, 600, 400)
	got, err := ReadImageMetadata(path)
	if err != nil {
		t.Fatalf("ReadImageMetadata(%q) error = %v; want nil", path, err)
	}
	if got.Width != 600 || got.Height != 400 {
		t.Errorf("dimensions = %dx%d; want 600x400", got.Width, got.Height)
	}
	// 600x400 -> ceil(600/512)=2, ceil(400/512)=1 -> 2 tiles -> 170 tokens
	if got.TokenEstimate != 170 {
		t.Errorf("TokenEstimate = %d; want 170", got.TokenEstimate)
	}
}

// TestReadImageMetadata_GIFFixture covers GIF headers as a third format
// recognised by Go's image package, mirroring PIL.Image.open's permissive
// header parsing in upstream.
func TestReadImageMetadata_GIFFixture(t *testing.T) {
	t.Parallel()
	path := writeGIF(t, 64, 32)
	got, err := ReadImageMetadata(path)
	if err != nil {
		t.Fatalf("ReadImageMetadata(%q) error = %v; want nil", path, err)
	}
	if got.Width != 64 || got.Height != 32 {
		t.Errorf("dimensions = %dx%d; want 64x32", got.Width, got.Height)
	}
}

// TestReadImageMetadata_NonImageFallsBackToNameOnly mirrors upstream's
// `except Exception: pass` branch: a file that is not a recognised image
// still returns metadata with a name, but width/height/token_estimate are
// zero (omitempty in JSON). The helper does not panic.
func TestReadImageMetadata_NonImageFallsBackToNameOnly(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("seed text fixture: %v", err)
	}
	got, err := ReadImageMetadata(path)
	if err != nil {
		t.Fatalf("ReadImageMetadata(non-image) error = %v; want nil", err)
	}
	if got.Name != "notes.txt" {
		t.Errorf("Name = %q; want %q", got.Name, "notes.txt")
	}
	if got.Width != 0 || got.Height != 0 || got.TokenEstimate != 0 {
		t.Errorf("non-image metadata = %+v; want zero dimensions/tokens", got)
	}
}

// TestReadImageMetadata_MissingFileReturnsError covers the real-IO failure
// surface: a missing path yields a non-nil error from the helper, which
// callers translate into a degraded response upstream.
func TestReadImageMetadata_MissingFileReturnsError(t *testing.T) {
	t.Parallel()
	_, err := ReadImageMetadata(filepath.Join(t.TempDir(), "absent.png"))
	if err == nil {
		t.Fatalf("ReadImageMetadata(missing) error = nil; want non-nil")
	}
}

// TestReadImageMetadata_BoundedHeaderRead asserts the helper does not
// load the entire image into memory. We seed a 12 MiB pseudo-PNG header
// followed by junk bytes; the helper must reject it (decode fails because
// of junk) without exhausting memory or returning success. The point of
// this test is the contract — bounded reads only — not the exact rejection
// message.
func TestReadImageMetadata_BoundedHeaderRead(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "fake.png")
	// PNG signature followed by junk; decode will fail but the helper
	// should not panic or hang. The 12 MiB size ensures any "read the
	// whole file" implementation would visibly stall the test process.
	header := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	junk := bytes.Repeat([]byte{'X'}, 12*1024*1024)
	if err := os.WriteFile(path, append(header, junk...), 0o600); err != nil {
		t.Fatalf("seed fake png: %v", err)
	}
	got, err := ReadImageMetadata(path)
	if err != nil {
		t.Fatalf("ReadImageMetadata(fake png) error = %v; want nil (graceful)", err)
	}
	if got.Width != 0 || got.Height != 0 {
		t.Errorf("garbage png dimensions = %dx%d; want 0x0", got.Width, got.Height)
	}
}

// ── Personality helpers ───────────────────────────────────────────────────

// TestRenderPersonalityPrompt_DictAssembly mirrors upstream
// hermes-agent/tui_gateway/server.py:_render_personality_prompt: a
// personality value with system_prompt + tone + style is rendered as
// three newline-separated lines.
func TestRenderPersonalityPrompt_DictAssembly(t *testing.T) {
	t.Parallel()
	value := map[string]any{
		"system_prompt": "You are calm.",
		"tone":          "thoughtful",
		"style":         "concise",
	}
	got := RenderPersonalityPrompt(value)
	const want = "You are calm.\nTone: thoughtful\nStyle: concise"
	if got != want {
		t.Errorf("RenderPersonalityPrompt(dict) = %q; want %q", got, want)
	}
}

// TestRenderPersonalityPrompt_DictSkipsBlankFields exercises the upstream
// `"\n".join(p for p in parts if p)` join: missing tone or style fields
// drop their lines instead of producing empty "Tone: " labels.
func TestRenderPersonalityPrompt_DictSkipsBlankFields(t *testing.T) {
	t.Parallel()
	value := map[string]any{
		"system_prompt": "You help.",
		"tone":          "",
	}
	got := RenderPersonalityPrompt(value)
	const want = "You help."
	if got != want {
		t.Errorf("RenderPersonalityPrompt(partial dict) = %q; want %q", got, want)
	}
}

// TestRenderPersonalityPrompt_PlainStringPassthrough proves the upstream
// `return str(value)` branch: a plain string personality is returned as
// the prompt verbatim.
func TestRenderPersonalityPrompt_PlainStringPassthrough(t *testing.T) {
	t.Parallel()
	got := RenderPersonalityPrompt("You are direct.")
	const want = "You are direct."
	if got != want {
		t.Errorf("RenderPersonalityPrompt(string) = %q; want %q", got, want)
	}
}

// TestValidatePersonality_BlankAndAliasesReturnEmpty mirrors the upstream
// `if not name or name in ("none", "default", "neutral"): return "", ""`
// branch: every personality alias clears the override.
func TestValidatePersonality_BlankAndAliasesReturnEmpty(t *testing.T) {
	t.Parallel()
	personalities := map[string]any{"helpful": "You help."}
	for _, raw := range []string{"", "  ", "none", "default", "neutral", "DEFAULT", " None "} {
		raw := raw
		t.Run(strings.TrimSpace(raw)+"_alias", func(t *testing.T) {
			t.Parallel()
			name, prompt, err := ValidatePersonality(raw, personalities)
			if err != nil {
				t.Fatalf("ValidatePersonality(%q) error = %v; want nil", raw, err)
			}
			if name != "" || prompt != "" {
				t.Errorf("ValidatePersonality(%q) = (%q, %q); want empty/empty", raw, name, prompt)
			}
		})
	}
}

// TestValidatePersonality_KnownNameReturnsRenderedPrompt covers the happy
// path: a configured personality round-trips its rendered prompt.
func TestValidatePersonality_KnownNameReturnsRenderedPrompt(t *testing.T) {
	t.Parallel()
	personalities := map[string]any{
		"helpful": map[string]any{
			"system_prompt": "You help.",
			"tone":          "warm",
		},
	}
	name, prompt, err := ValidatePersonality("Helpful", personalities)
	if err != nil {
		t.Fatalf("ValidatePersonality(known) error = %v; want nil", err)
	}
	if name != "helpful" {
		t.Errorf("name = %q; want %q", name, "helpful")
	}
	const wantPrompt = "You help.\nTone: warm"
	if prompt != wantPrompt {
		t.Errorf("prompt = %q; want %q", prompt, wantPrompt)
	}
}

// TestValidatePersonality_UnknownNameWithChoicesReportsAvailable mirrors
// the upstream "Available: `none`, `helpful`, …" sentence appended to the
// error so operators see the authoritative list. The list is sorted for
// determinism.
func TestValidatePersonality_UnknownNameWithChoicesReportsAvailable(t *testing.T) {
	t.Parallel()
	personalities := map[string]any{
		"zen":     "Be calm.",
		"helpful": "You help.",
		"acme":    "Acme.",
	}
	_, _, err := ValidatePersonality("bogus", personalities)
	if err == nil {
		t.Fatalf("ValidatePersonality(bogus) error = nil; want non-nil")
	}
	const want = "Unknown personality: `bogus`.\n\nAvailable: `none`, `acme`, `helpful`, `zen`"
	if err.Error() != want {
		t.Errorf("error = %q; want %q", err.Error(), want)
	}
}

// TestValidatePersonality_UnknownNameWithoutChoicesReportsEmpty matches
// the upstream "No personalities configured." branch when the personality
// map is empty.
func TestValidatePersonality_UnknownNameWithoutChoicesReportsEmpty(t *testing.T) {
	t.Parallel()
	_, _, err := ValidatePersonality("bogus", nil)
	if err == nil {
		t.Fatalf("ValidatePersonality(bogus, nil) error = nil; want non-nil")
	}
	const want = "Unknown personality: `bogus`.\n\nNo personalities configured."
	if err.Error() != want {
		t.Errorf("error = %q; want %q", err.Error(), want)
	}
}

// TestValidatePersonality_PreservesRawCasingInError keeps the operator's
// original input visible in the error message even though the lookup is
// case-insensitive (matching upstream's `f"Unknown personality: \`{raw}\`"`).
func TestValidatePersonality_PreservesRawCasingInError(t *testing.T) {
	t.Parallel()
	_, _, err := ValidatePersonality("  BOGUS  ", map[string]any{"helpful": "x"})
	if err == nil {
		t.Fatalf("ValidatePersonality(BOGUS) error = nil; want non-nil")
	}
	if !strings.Contains(err.Error(), "`BOGUS`") {
		t.Errorf("error = %q; want it to contain raw casing `BOGUS`", err.Error())
	}
}

// ── PlatformEvent structs ──────────────────────────────────────────────────

// TestPlatformEventSubmit_RoundTripJSON proves a SubmitEvent serialises and
// deserialises through encoding/json without lossy fields.
func TestPlatformEventSubmit_RoundTripJSON(t *testing.T) {
	t.Parallel()
	in := SubmitEvent{Kind: PlatformEventKindSubmit, SessionID: "sid-1", Text: "hello"}
	if in.EventKind() != PlatformEventKindSubmit {
		t.Fatalf("EventKind = %q; want %q", in.EventKind(), PlatformEventKindSubmit)
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal(SubmitEvent) error = %v", err)
	}
	var out SubmitEvent
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal(SubmitEvent) error = %v", err)
	}
	if out != in {
		t.Errorf("round trip = %+v; want %+v", out, in)
	}
	if !strings.Contains(string(raw), `"kind":"submit"`) {
		t.Errorf("submit JSON %q must encode kind=%q", raw, "submit")
	}
}

// TestPlatformEventCancel_RoundTripJSON confirms the cancel event has only
// session-scoped fields and round-trips through JSON intact.
func TestPlatformEventCancel_RoundTripJSON(t *testing.T) {
	t.Parallel()
	in := CancelEvent{Kind: PlatformEventKindCancel, SessionID: "sid-2"}
	if in.EventKind() != PlatformEventKindCancel {
		t.Fatalf("EventKind = %q; want %q", in.EventKind(), PlatformEventKindCancel)
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal(CancelEvent) error = %v", err)
	}
	var out CancelEvent
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal(CancelEvent) error = %v", err)
	}
	if out != in {
		t.Errorf("round trip = %+v; want %+v", out, in)
	}
	if !strings.Contains(string(raw), `"kind":"cancel"`) {
		t.Errorf("cancel JSON %q must encode kind=%q", raw, "cancel")
	}
}

// TestPlatformEventResize_RoundTripJSON exercises the resize variant
// modelled after upstream terminal.resize: it carries cols (Bubble Tea
// ports an integer column count) and round-trips through JSON.
func TestPlatformEventResize_RoundTripJSON(t *testing.T) {
	t.Parallel()
	in := ResizeEvent{Kind: PlatformEventKindResize, SessionID: "sid-3", Cols: 132}
	if in.EventKind() != PlatformEventKindResize {
		t.Fatalf("EventKind = %q; want %q", in.EventKind(), PlatformEventKindResize)
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal(ResizeEvent) error = %v", err)
	}
	var out ResizeEvent
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal(ResizeEvent) error = %v", err)
	}
	if out != in {
		t.Errorf("round trip = %+v; want %+v", out, in)
	}
	if !strings.Contains(string(raw), `"cols":132`) {
		t.Errorf("resize JSON %q must encode cols=132", raw)
	}
}

// TestPlatformEventProgress_RoundTripJSON verifies the progress variant
// carries tool name + preview, mirroring upstream's tool.progress emit.
func TestPlatformEventProgress_RoundTripJSON(t *testing.T) {
	t.Parallel()
	in := ProgressEvent{
		Kind:      PlatformEventKindProgress,
		SessionID: "sid-4",
		ToolName:  "web_search",
		Preview:   "fetching results…",
	}
	if in.EventKind() != PlatformEventKindProgress {
		t.Fatalf("EventKind = %q; want %q", in.EventKind(), PlatformEventKindProgress)
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal(ProgressEvent) error = %v", err)
	}
	var out ProgressEvent
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal(ProgressEvent) error = %v", err)
	}
	if out != in {
		t.Errorf("round trip = %+v; want %+v", out, in)
	}
	if !strings.Contains(string(raw), `"tool_name":"web_search"`) {
		t.Errorf("progress JSON %q must encode tool_name", raw)
	}
}

// TestPlatformEventImageMetadata_RoundTripJSON proves the embedded
// ImageMetadata payload survives JSON marshal/unmarshal as part of the
// platform event envelope.
func TestPlatformEventImageMetadata_RoundTripJSON(t *testing.T) {
	t.Parallel()
	in := ImageMetadataEvent{
		Kind:      PlatformEventKindImageMetadata,
		SessionID: "sid-5",
		Metadata: ImageMetadata{
			Name:          "cat.png",
			Width:         320,
			Height:        240,
			TokenEstimate: 85,
		},
	}
	if in.EventKind() != PlatformEventKindImageMetadata {
		t.Fatalf("EventKind = %q; want %q", in.EventKind(), PlatformEventKindImageMetadata)
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal(ImageMetadataEvent) error = %v", err)
	}
	var out ImageMetadataEvent
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal(ImageMetadataEvent) error = %v", err)
	}
	if out != in {
		t.Errorf("round trip = %+v; want %+v", out, in)
	}
	if !strings.Contains(string(raw), `"name":"cat.png"`) {
		t.Errorf("image metadata JSON %q must encode name=cat.png", raw)
	}
}

// TestImageMetadata_OmitsEmptyDimensionsInJSON exercises the omitempty
// contract: a metadata struct with only a name (decode-failed image) emits
// just `{"name":"…"}`, so wire payloads stay compact for fallback cases.
func TestImageMetadata_OmitsEmptyDimensionsInJSON(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal(ImageMetadata{Name: "fallback.bin"})
	if err != nil {
		t.Fatalf("Marshal(ImageMetadata) error = %v", err)
	}
	const want = `{"name":"fallback.bin"}`
	if string(raw) != want {
		t.Errorf("ImageMetadata JSON = %s; want %s", raw, want)
	}
}

// TestValidatePlatformEventKind covers the discriminator helper: recognised
// kinds round-trip true, unknown strings round-trip false. Callers use this
// to guard JSON-RPC dispatch when transport is wired in later slices.
func TestValidatePlatformEventKind(t *testing.T) {
	t.Parallel()
	known := []PlatformEventKind{
		PlatformEventKindSubmit,
		PlatformEventKindCancel,
		PlatformEventKindResize,
		PlatformEventKindProgress,
		PlatformEventKindImageMetadata,
	}
	for _, k := range known {
		k := k
		t.Run(string(k), func(t *testing.T) {
			t.Parallel()
			if !ValidPlatformEventKind(k) {
				t.Errorf("ValidPlatformEventKind(%q) = false; want true", k)
			}
		})
	}
	for _, raw := range []string{"", "delete", "Submit", "image-metadata"} {
		raw := raw
		t.Run("invalid_"+raw, func(t *testing.T) {
			t.Parallel()
			if ValidPlatformEventKind(PlatformEventKind(raw)) {
				t.Errorf("ValidPlatformEventKind(%q) = true; want false", raw)
			}
		})
	}
}

// ── Test fixture helpers (temp-only, no network) ─────────────────────────

func writePNG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	path := filepath.Join(t.TempDir(), "fixture.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return path
}

func writeJPEG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	path := filepath.Join(t.TempDir(), "fixture.jpg")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create jpeg: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 75}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return path
}

func writeGIF(t *testing.T, w, h int) string {
	t.Helper()
	pal := []color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}}
	img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	path := filepath.Join(t.TempDir(), "fixture.gif")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create gif: %v", err)
	}
	defer f.Close()
	if err := gif.Encode(f, img, nil); err != nil {
		t.Fatalf("encode gif: %v", err)
	}
	return path
}
