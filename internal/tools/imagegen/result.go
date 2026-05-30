package imagegen

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ImageGenerationStatus is the typed envelope outcome surfaced to transcripts
// and downstream tooling. Only the listed values are valid; new modes must be
// added here so transcript exporters can switch on a closed set.
type ImageGenerationStatus string

const (
	// ImageGenerationStatusOK indicates an artifact was written and the
	// envelope's Artifact path is consumable.
	ImageGenerationStatusOK ImageGenerationStatus = "ok"

	// ImageGenerationStatusUnavailable indicates the provider/model is not
	// configured or otherwise unreachable in a structural way (no key,
	// disabled in config). Callers should not retry with the same inputs.
	ImageGenerationStatusUnavailable ImageGenerationStatus = "image_generation_unavailable"

	// ImageGenerationStatusProviderError indicates a transport- or
	// provider-level failure during generation. The Reason field carries
	// a redacted human-readable description.
	ImageGenerationStatusProviderError ImageGenerationStatus = "image_generation_provider_error"

	// ImageGenerationStatusProviderNotRegistered indicates image_gen.provider
	// names a plugin provider that is not currently registered.
	ImageGenerationStatusProviderNotRegistered ImageGenerationStatus = "provider_not_registered"
)

// ImageGenerationEnvelope is the stable, transcript-safe result of an image
// generation tool call. It deliberately omits raw prompt text and raw
// binary payloads — Artifact is a relative path under the caller's
// OutputDir and PromptHash is the only prompt-derived identifier.
type ImageGenerationEnvelope struct {
	Provider   string                `json:"provider"`
	Model      string                `json:"model"`
	PromptHash string                `json:"prompt_hash"`
	MediaType  string                `json:"media_type,omitempty"`
	Artifact   string                `json:"artifact,omitempty"`
	Status     ImageGenerationStatus `json:"status"`
	Reason     string                `json:"reason,omitempty"`
}

// ImageGenerationRequest captures the inputs needed to materialise an
// envelope. Network calls and decoding happen in the caller; this helper
// is pure given Bytes (success) or Err (failure).
type ImageGenerationRequest struct {
	Provider  string
	Model     string
	Prompt    string
	OutputDir string
	Bytes     []byte
	MediaType string

	// Err signals a failed generation. When non-nil, Bytes is ignored and
	// the envelope reports ErrorStatus. ErrorStatus must be one of the
	// degraded modes; if unset it defaults to provider error.
	Err         error
	ErrorStatus ImageGenerationStatus
}

// secretRedactionMarkers are substrings BuildImageGenerationEnvelope strips
// from any human-readable Reason field so provider error messages cannot
// leak operator credentials into transcripts.
var secretRedactionMarkers = []string{
	"ANTHROPIC_API_KEY",
	"OPENAI_API_KEY",
	"sk-",
	"Bearer ",
}

// reBearer matches Bearer-prefixed tokens for whole-token redaction.
var reBearer = regexp.MustCompile(`(?i)Bearer\s+\S+`)

// reSKToken matches OpenAI-style sk- tokens.
var reSKToken = regexp.MustCompile(`sk-[A-Za-z0-9_\-]+`)

// reKeyAssign matches KEY=value style leaks for known sensitive env vars.
var reKeyAssign = regexp.MustCompile(`(?i)(ANTHROPIC_API_KEY|OPENAI_API_KEY)\s*=\s*\S+`)

// extByMediaType returns a best-effort artifact extension for the given
// IANA media type. Falls back to .bin so the path is always usable.
func extByMediaType(mt string) string {
	switch strings.ToLower(strings.TrimSpace(mt)) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	}
	return ".bin"
}

// hashPrompt computes the hex-encoded sha256 of the prompt. Empty prompts
// still produce a stable hash so callers can identify reruns.
func hashPrompt(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

// redactReason scrubs known credential shapes from a free-form error
// message and elides the raw prompt if it appears verbatim.
func redactReason(reason, prompt string) string {
	if reason == "" {
		return reason
	}
	out := reason
	out = reBearer.ReplaceAllString(out, "[REDACTED_BEARER]")
	out = reSKToken.ReplaceAllString(out, "[REDACTED_SK_TOKEN]")
	out = reKeyAssign.ReplaceAllString(out, "[REDACTED_API_KEY]")
	for _, marker := range secretRedactionMarkers {
		// Defence-in-depth: drop any residual marker substring that
		// survived regex-based redaction.
		out = strings.ReplaceAll(out, marker, "[REDACTED]")
	}
	if prompt != "" && strings.Contains(out, prompt) {
		out = strings.ReplaceAll(out, prompt, "[REDACTED_PROMPT]")
	}
	return out
}

// BuildImageGenerationEnvelope is the pure, transcript-safe boundary for
// image-generation results. On success it writes Bytes to a freshly named
// file under OutputDir and returns an OK envelope; on Err it returns a
// degraded envelope with no file write. It never echoes raw prompt text or
// provider credentials.
func BuildImageGenerationEnvelope(req ImageGenerationRequest) (ImageGenerationEnvelope, error) {
	if req.Provider == "" {
		return ImageGenerationEnvelope{}, errors.New("image_generation: provider is required")
	}
	if req.Model == "" {
		return ImageGenerationEnvelope{}, errors.New("image_generation: model is required")
	}
	if req.OutputDir == "" {
		return ImageGenerationEnvelope{}, errors.New("image_generation: output dir is required")
	}

	promptHash := hashPrompt(req.Prompt)

	// Error path: never write a file, never echo bytes, redact reason.
	if req.Err != nil {
		status := req.ErrorStatus
		switch status {
		case ImageGenerationStatusUnavailable, ImageGenerationStatusProviderError, ImageGenerationStatusProviderNotRegistered:
			// ok
		default:
			status = ImageGenerationStatusProviderError
		}
		return ImageGenerationEnvelope{
			Provider:   req.Provider,
			Model:      req.Model,
			PromptHash: promptHash,
			Status:     status,
			Reason:     redactReason(req.Err.Error(), req.Prompt),
		}, nil
	}

	if len(req.Bytes) == 0 {
		return ImageGenerationEnvelope{}, errors.New("image_generation: bytes required for ok envelope")
	}
	if req.MediaType == "" {
		return ImageGenerationEnvelope{}, errors.New("image_generation: media type is required")
	}

	// Resolve output dir to an absolute path so subsequent path checks are
	// stable regardless of caller cwd.
	absDir, err := filepath.Abs(req.OutputDir)
	if err != nil {
		return ImageGenerationEnvelope{}, fmt.Errorf("image_generation: resolve output dir: %w", err)
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return ImageGenerationEnvelope{}, fmt.Errorf("image_generation: create output dir: %w", err)
	}

	name := fmt.Sprintf("image-%s%s", promptHash[:16], extByMediaType(req.MediaType))
	full := filepath.Join(absDir, name)

	// Belt-and-braces: refuse to write outside the resolved OutputDir
	// even if extByMediaType ever returned a path-bearing extension.
	rel, err := filepath.Rel(absDir, full)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return ImageGenerationEnvelope{}, fmt.Errorf("image_generation: refusing to write outside output dir")
	}

	if err := os.WriteFile(full, req.Bytes, 0o644); err != nil {
		return ImageGenerationEnvelope{}, fmt.Errorf("image_generation: write artifact: %w", err)
	}

	return ImageGenerationEnvelope{
		Provider:   req.Provider,
		Model:      req.Model,
		PromptHash: promptHash,
		MediaType:  req.MediaType,
		Artifact:   rel,
		Status:     ImageGenerationStatusOK,
	}, nil
}
