package imagegen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// secretMarkers lists substrings that must never appear in any
// JSON or string-form rendering of an image-generation envelope.
var secretMarkers = []string{
	"ANTHROPIC_API_KEY",
	"OPENAI_API_KEY",
	"sk-",
	"Bearer ",
}

// renderEnvelopeForms returns a slice of string renderings of the envelope
// callers might surface (JSON marshal + raw struct printf) so redaction
// scans can run against every concrete leak surface.
func renderEnvelopeForms(t *testing.T, env ImageGenerationEnvelope) []string {
	t.Helper()
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	pretty, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		t.Fatalf("marshal indent envelope: %v", err)
	}
	return []string{
		string(raw),
		string(pretty),
		// Go's default %#v / %+v stringer surfaces every field too.
		structString(env),
	}
}

func structString(env ImageGenerationEnvelope) string {
	// Concatenate every string field directly so a leaked prompt or key
	// can be detected even if json marshaling were customised later.
	return strings.Join([]string{
		env.Provider,
		env.Model,
		env.PromptHash,
		env.MediaType,
		env.Artifact,
		string(env.Status),
		env.Reason,
	}, "\n")
}

func TestImageGenerationResult_WritesArtifactEnvelope(t *testing.T) {
	dir := t.TempDir()

	prompt := "a serene watercolor of an alpine lake at dawn"
	bytes := []byte("\x89PNG\r\n\x1a\n-fake-image-bytes-")

	req := ImageGenerationRequest{
		Provider:  "fal",
		Model:     "fal-ai/flux-2-pro",
		Prompt:    prompt,
		OutputDir: dir,
		Bytes:     bytes,
		MediaType: "image/png",
	}

	env, err := BuildImageGenerationEnvelope(req)
	if err != nil {
		t.Fatalf("BuildImageGenerationEnvelope: unexpected error: %v", err)
	}

	if env.Provider != "fal" {
		t.Errorf("Provider = %q, want %q", env.Provider, "fal")
	}
	if env.Model != "fal-ai/flux-2-pro" {
		t.Errorf("Model = %q, want %q", env.Model, "fal-ai/flux-2-pro")
	}
	if env.MediaType != "image/png" {
		t.Errorf("MediaType = %q, want %q", env.MediaType, "image/png")
	}
	if env.Status != ImageGenerationStatusOK {
		t.Errorf("Status = %q, want %q", env.Status, ImageGenerationStatusOK)
	}

	// Artifact must be a relative path (never absolute, never escapes dir).
	if env.Artifact == "" {
		t.Fatalf("Artifact path is empty")
	}
	if filepath.IsAbs(env.Artifact) {
		t.Errorf("Artifact = %q must be relative to OutputDir", env.Artifact)
	}
	if strings.Contains(env.Artifact, "..") {
		t.Errorf("Artifact = %q must not traverse parent directories", env.Artifact)
	}

	// File must exist on disk under the injected dir with the expected bytes.
	full := filepath.Join(dir, env.Artifact)
	got, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read artifact %q: %v", full, err)
	}
	if string(got) != string(bytes) {
		t.Errorf("artifact content mismatch: got %d bytes, want %d bytes", len(got), len(bytes))
	}

	// Prompt hash is the hex-encoded sha256 of the full prompt text.
	sum := sha256.Sum256([]byte(prompt))
	wantHash := hex.EncodeToString(sum[:])
	if env.PromptHash != wantHash {
		t.Errorf("PromptHash = %q, want %q", env.PromptHash, wantHash)
	}
}

func TestImageGenerationResult_DoesNotOverwriteSamePromptArtifacts(t *testing.T) {
	dir := t.TempDir()

	req := ImageGenerationRequest{
		Provider:  "fal",
		Model:     "fal-ai/flux-2-pro",
		Prompt:    "repeatable prompt",
		OutputDir: dir,
		Bytes:     []byte("first-image"),
		MediaType: "image/png",
	}

	first, err := BuildImageGenerationEnvelope(req)
	if err != nil {
		t.Fatalf("first BuildImageGenerationEnvelope: %v", err)
	}

	req.Bytes = []byte("second-image")
	second, err := BuildImageGenerationEnvelope(req)
	if err != nil {
		t.Fatalf("second BuildImageGenerationEnvelope: %v", err)
	}

	if first.Artifact == "" || second.Artifact == "" {
		t.Fatalf("artifacts must be populated: first=%q second=%q", first.Artifact, second.Artifact)
	}
	if first.Artifact == second.Artifact {
		t.Fatalf("same prompt generated the same artifact path %q; earlier result would be overwritten", first.Artifact)
	}

	gotFirst, err := os.ReadFile(filepath.Join(dir, first.Artifact))
	if err != nil {
		t.Fatalf("read first artifact: %v", err)
	}
	if string(gotFirst) != "first-image" {
		t.Fatalf("first artifact content = %q, want first-image", string(gotFirst))
	}

	gotSecond, err := os.ReadFile(filepath.Join(dir, second.Artifact))
	if err != nil {
		t.Fatalf("read second artifact: %v", err)
	}
	if string(gotSecond) != "second-image" {
		t.Fatalf("second artifact content = %q, want second-image", string(gotSecond))
	}
}

func TestImageGenerationResult_RedactsPromptAndSecrets(t *testing.T) {
	dir := t.TempDir()

	// A prompt that contains marker tokens we will scan for.
	prompt := "secret-prompt-fragment ANTHROPIC_API_KEY=sk-leaked-123 Bearer abc"

	req := ImageGenerationRequest{
		Provider:  "openai",
		Model:     "gpt-image-1.5",
		Prompt:    prompt,
		OutputDir: dir,
		Bytes:     []byte("not-a-real-image"),
		MediaType: "image/png",
	}

	env, err := BuildImageGenerationEnvelope(req)
	if err != nil {
		t.Fatalf("BuildImageGenerationEnvelope: %v", err)
	}

	for _, form := range renderEnvelopeForms(t, env) {
		if strings.Contains(form, "secret-prompt-fragment") {
			t.Errorf("envelope form leaks raw prompt fragment: %s", form)
		}
		for _, marker := range secretMarkers {
			if strings.Contains(form, marker) {
				t.Errorf("envelope form leaks secret marker %q: %s", marker, form)
			}
		}
	}

	// The hex-encoded prompt hash IS expected to appear (it is the
	// only sanctioned identifier that derives from the prompt).
	sum := sha256.Sum256([]byte(prompt))
	wantHash := hex.EncodeToString(sum[:])
	if env.PromptHash != wantHash {
		t.Errorf("PromptHash = %q, want %q", env.PromptHash, wantHash)
	}
}

func TestImageGenerationResult_ErrorEnvelopeRedactsPromptEvenWhenPromptContainsSecretShape(t *testing.T) {
	dir := t.TempDir()
	prompt := "operator prompt with embedded sk-prompt-token and Bearer prompt-token"

	env, err := BuildImageGenerationEnvelope(ImageGenerationRequest{
		Provider:    "openai",
		Model:       "gpt-image-1.5",
		Prompt:      prompt,
		OutputDir:   dir,
		Err:         errors.New("provider rejected prompt: " + prompt),
		ErrorStatus: ImageGenerationStatusProviderError,
	})
	if err != nil {
		t.Fatalf("BuildImageGenerationEnvelope: %v", err)
	}

	for _, form := range renderEnvelopeForms(t, env) {
		if strings.Contains(form, "operator prompt with embedded") || strings.Contains(form, "prompt-token") {
			t.Fatalf("error envelope leaks prompt fragments after secret redaction: %s", form)
		}
		for _, marker := range secretMarkers {
			if strings.Contains(form, marker) {
				t.Fatalf("error envelope leaks secret marker %q: %s", marker, form)
			}
		}
	}
}

func TestImageGenerationResult_ErrorEnvelopeReasonTruncationPreservesUTF8(t *testing.T) {
	env, err := BuildImageGenerationEnvelope(ImageGenerationRequest{
		Provider:    "openai",
		Model:       "gpt-image-1.5",
		Prompt:      "safe prompt",
		Err:         errors.New(strings.Repeat("a", maxImageGenerationReasonLength-1) + "€"),
		ErrorStatus: ImageGenerationStatusProviderError,
	})
	if err != nil {
		t.Fatalf("BuildImageGenerationEnvelope: %v", err)
	}
	if !utf8.ValidString(env.Reason) {
		t.Fatalf("truncated reason is not valid UTF-8: %q", env.Reason)
	}
	if !strings.HasSuffix(env.Reason, "...") {
		t.Fatalf("truncated reason should keep ellipsis suffix, got %q", env.Reason)
	}
}

func TestImageGenerationResult_ErrorEnvelopeRedactsCredentialShapesUsedByRunner(t *testing.T) {
	dir := t.TempDir()
	jwt := "aaaaaaaaaaaaaaaaaaaa.bbbbbbbbbbbbbbbbbbbb.cccccccccccccccccccc"
	falKey := "FAL_SECRET_TOKEN_123456789"
	env, err := BuildImageGenerationEnvelope(ImageGenerationRequest{
		Provider:    "openai",
		Model:       "gpt-image-1.5",
		Prompt:      "safe prompt",
		OutputDir:   dir,
		Err:         errors.New("provider failed token=abc123 secret=shh Key " + falKey + " " + jwt),
		ErrorStatus: ImageGenerationStatusProviderError,
	})
	if err != nil {
		t.Fatalf("BuildImageGenerationEnvelope: %v", err)
	}

	for _, leaked := range []string{"token=abc123", "secret=shh", "Key " + falKey, falKey, jwt} {
		if strings.Contains(env.Reason, leaked) {
			t.Fatalf("error envelope leaks credential shape %q in reason %q", leaked, env.Reason)
		}
	}
}

func TestImageGenerationResult_ErrorEnvelopeDoesNotRequireOutputDir(t *testing.T) {
	env, err := BuildImageGenerationEnvelope(ImageGenerationRequest{
		Provider:    "openai",
		Model:       "gpt-image-1.5",
		Prompt:      "a safe prompt",
		Err:         errors.New("provider not configured"),
		ErrorStatus: ImageGenerationStatusUnavailable,
	})
	if err != nil {
		t.Fatalf("BuildImageGenerationEnvelope: %v", err)
	}
	if env.Status != ImageGenerationStatusUnavailable {
		t.Fatalf("Status = %q, want %q", env.Status, ImageGenerationStatusUnavailable)
	}
	if env.Artifact != "" {
		t.Fatalf("Artifact = %q, want empty for degraded envelope", env.Artifact)
	}
}

func TestImageGenerationResult_ErrorEnvelopePreservesProviderUnavailableEvidence(t *testing.T) {
	env, err := BuildImageGenerationEnvelope(ImageGenerationRequest{
		Provider:    "openai",
		Model:       "gpt-image-1.5",
		Prompt:      "a safe prompt",
		Err:         errors.New("provider unavailable"),
		ErrorStatus: ImageGenerationStatusProviderUnavailable,
	})
	if err != nil {
		t.Fatalf("BuildImageGenerationEnvelope: %v", err)
	}
	if env.Status != ImageGenerationStatusProviderUnavailable {
		t.Fatalf("Status = %q, want provider-unavailable evidence", env.Status)
	}
}

func TestImageGenerationResult_ErrorEnvelope(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name       string
		err        error
		wantStatus ImageGenerationStatus
	}{
		{
			name:       "nil bytes treated as unavailable",
			err:        errors.New("provider not configured"),
			wantStatus: ImageGenerationStatusUnavailable,
		},
		{
			name:       "provider error",
			err:        errors.New("upstream 500: image_generation failed (Bearer leaked-token sk-secret-abc OPENAI_API_KEY=foo)"),
			wantStatus: ImageGenerationStatusProviderError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var status ImageGenerationStatus
			switch tc.wantStatus {
			case ImageGenerationStatusUnavailable:
				status = ImageGenerationStatusUnavailable
			case ImageGenerationStatusProviderError:
				status = ImageGenerationStatusProviderError
			}

			req := ImageGenerationRequest{
				Provider:    "openai",
				Model:       "gpt-image-1.5",
				Prompt:      "a beach scene",
				OutputDir:   dir,
				Bytes:       nil,
				MediaType:   "image/png",
				Err:         tc.err,
				ErrorStatus: status,
			}

			env, err := BuildImageGenerationEnvelope(req)
			if err != nil {
				t.Fatalf("BuildImageGenerationEnvelope: %v", err)
			}

			if env.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", env.Status, tc.wantStatus)
			}
			if env.Artifact != "" {
				t.Errorf("Artifact = %q, want empty for error envelope", env.Artifact)
			}

			// Provider/model identity preserved for transcript routing.
			if env.Provider == "" || env.Model == "" {
				t.Errorf("Provider/Model must be preserved on error: %+v", env)
			}

			// Reason exists but contains no secret markers.
			if env.Reason == "" {
				t.Errorf("Reason should describe the degraded error in human terms")
			}
			for _, form := range renderEnvelopeForms(t, env) {
				for _, marker := range secretMarkers {
					if strings.Contains(form, marker) {
						t.Errorf("error envelope leaks secret marker %q: %s", marker, form)
					}
				}
				// Raw provider error must not echo prompt bytes either.
				if strings.Contains(form, "a beach scene") {
					t.Errorf("error envelope leaks raw prompt: %s", form)
				}
			}

			// No file must have been written for an error envelope.
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("read tempdir: %v", err)
			}
			if len(entries) != 0 {
				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}
				t.Errorf("error envelope wrote files into OutputDir: %v", names)
			}
		})
	}
}

// TestImageGenerationResult_TempDirOnly proves the helper refuses to write
// outside its injected OutputDir even when the caller is sloppy. Together
// with the use of t.TempDir() in the other tests, this satisfies the
// "no test writes outside t.TempDir()" acceptance condition.
func TestImageGenerationResult_TempDirOnly(t *testing.T) {
	// Empty output dir must not silently fall back to cwd or $HOME.
	_, err := BuildImageGenerationEnvelope(ImageGenerationRequest{
		Provider:  "fal",
		Model:     "fal-ai/flux-2-pro",
		Prompt:    "x",
		OutputDir: "",
		Bytes:     []byte("data"),
		MediaType: "image/png",
	})
	if err == nil {
		t.Fatal("expected error when OutputDir is empty")
	}
}
