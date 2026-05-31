package budget

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// Stable evidence codes returned by FormatToolResult so callers (gateway,
// channel handlers) can record degraded-mode operator evidence without
// depending on free-form messages.
const (
	ToolResultEvidenceUnderBudget       = "tool_output_under_budget"
	ToolResultEvidenceTruncated         = "tool_output_truncated"
	ToolResultEvidencePersisted         = "tool_output_persisted"
	ToolResultEvidencePersistenceFailed = "tool_output_persistence_failed"
)

// Default fallback bounds applied when callers leave config zero-valued.
// Owned divergence from Hermes: Hermes defaults are 100_000 result chars,
// 200_000 turn chars, and 1_500 preview chars. Gormes keeps smaller fallback
// bytes here because gateway/channel safety wins when caller config is absent;
// callers that need Hermes-sized budgets must pass explicit config.
const (
	defaultToolTextBudgetBytes = 4 * 1024
	defaultToolPreviewBytes    = 512
	maxToolPointerHeader       = 256
)

// ToolResultBudgetConfig is the per-call shape the helper expects from a
// session/run-scoped caller. OutputDir must already be a directory the caller
// considers safe for artifact storage; the helper never writes outside it.
type ToolResultBudgetConfig struct {
	// OutputDir is the session/run scoped temp dir that contains every
	// artifact written by this helper. Required for persistence. When empty
	// or unwriteable the helper degrades to inline truncated output.
	OutputDir string
	// TextBudgetBytes is the size after which text output is truncated and
	// promoted into an artifact pointer. Zero means use a safe default.
	TextBudgetBytes int
	// PreviewBytes is the head retained inline as a preview. Zero means use
	// a safe default.
	PreviewBytes int
}

// ToolResultEvidence is the structured record returned alongside the model-
// facing pointer string. Tools surface it to the gateway and operator log so
// degraded paths and persisted artifacts are auditable.
type ToolResultEvidence struct {
	// Code is one of the ToolResultEvidence* constants.
	Code string
	// Artifact is the relative path under OutputDir that received the full
	// payload, or empty when no artifact was written.
	Artifact string
	// Preview is the head of the original output retained inline so the
	// model has something to react to without us shipping the whole payload.
	Preview string
	// Bytes is the total size of the original raw output.
	Bytes int
}

// FormatToolResult bounds raw tool output to a session-scoped artifact when it
// exceeds the configured budget. It returns the short pointer text safe to
// expose to the model/channel and structured evidence describing what
// happened. The helper is pure: it touches only OutputDir on disk and never
// performs network I/O.
func FormatToolResult(cfg ToolResultBudgetConfig, raw []byte, mediaType string) (string, ToolResultEvidence, error) {
	cfg = cfg.withDefaults()
	if isTextMedia(mediaType) {
		raw = []byte(redaction.StripANSI(string(raw)))
	}

	bytes := len(raw)
	preview := safePreview(raw, cfg.PreviewBytes)

	if !cfg.shouldPersist(raw, mediaType) {
		return preview, ToolResultEvidence{
			Code:    ToolResultEvidenceUnderBudget,
			Preview: preview,
			Bytes:   bytes,
		}, nil
	}

	rel, err := persistArtifact(cfg.OutputDir, raw, mediaType)
	if err != nil {
		// Degraded mode: do not error out. Return bounded inline preview so
		// the channel/provider still gets *something* without flooding, and
		// surface persistence_failed evidence for operator triage.
		degraded := truncatePointer(preview)
		return degraded, ToolResultEvidence{
			Code:    ToolResultEvidencePersistenceFailed,
			Preview: degraded,
			Bytes:   bytes,
		}, nil
	}

	pointer := buildPointer(rel, preview, mediaType, bytes)
	code := ToolResultEvidenceTruncated
	// JSON/non-text payloads under the text budget still get persisted but
	// they were not "truncated" in the textual sense; they were promoted to
	// an artifact pointer. Report persisted in that case.
	if !isTextMedia(mediaType) || bytes <= cfg.TextBudgetBytes {
		code = ToolResultEvidencePersisted
	}
	return pointer, ToolResultEvidence{
		Code:     code,
		Artifact: rel,
		Preview:  preview,
		Bytes:    bytes,
	}, nil
}

func (cfg ToolResultBudgetConfig) withDefaults() ToolResultBudgetConfig {
	if cfg.TextBudgetBytes <= 0 {
		cfg.TextBudgetBytes = defaultToolTextBudgetBytes
	}
	if cfg.PreviewBytes <= 0 {
		cfg.PreviewBytes = defaultToolPreviewBytes
	}
	if cfg.PreviewBytes > cfg.TextBudgetBytes {
		cfg.PreviewBytes = cfg.TextBudgetBytes
	}
	return cfg
}

// shouldPersist decides whether the raw payload must be promoted to an
// artifact. Text output stays inline when it fits the text budget; non-text
// output is always persisted because the model/channel cannot safely consume
// raw bytes inline.
func (cfg ToolResultBudgetConfig) shouldPersist(raw []byte, mediaType string) bool {
	if !isTextMedia(mediaType) {
		return true
	}
	return len(raw) > cfg.TextBudgetBytes
}

func isTextMedia(mediaType string) bool {
	base := baseMediaType(mediaType)
	if base == "" {
		return true
	}
	if strings.HasPrefix(base, "text/") {
		return true
	}
	return false
}

func baseMediaType(mediaType string) string {
	mt := strings.ToLower(strings.TrimSpace(mediaType))
	if base, _, ok := strings.Cut(mt, ";"); ok {
		mt = strings.TrimSpace(base)
	}
	return mt
}

// persistArtifact writes raw to a sanitized relative path under outputDir.
// The filename is derived from a content hash so callers cannot influence the
// path; this defends against `..` traversal regardless of upstream input.
func persistArtifact(outputDir string, raw []byte, mediaType string) (string, error) {
	if outputDir == "" {
		return "", errors.New("output dir required")
	}
	if err := os.MkdirAll(outputDir, 0o750); err != nil {
		return "", fmt.Errorf("create artifact dir: %w", err)
	}

	sum := sha256.Sum256(raw)
	name := hex.EncodeToString(sum[:16]) + extensionFor(mediaType)

	// Defense-in-depth: confirm the joined path stays inside outputDir even
	// though the filename is derived from a hash and cannot legitimately
	// escape.
	cleanDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output dir: %w", err)
	}
	candidate, err := filepath.Abs(filepath.Join(outputDir, name))
	if err != nil {
		return "", fmt.Errorf("resolve artifact path: %w", err)
	}
	if !strings.HasPrefix(candidate+string(filepath.Separator), cleanDir+string(filepath.Separator)) &&
		candidate != cleanDir {
		return "", errors.New("artifact path escapes output dir")
	}

	if err := os.WriteFile(candidate, raw, 0o640); err != nil {
		return "", fmt.Errorf("write artifact: %w", err)
	}
	return name, nil
}

// extensionFor maps a media type to a stable file extension. We deliberately
// keep this conservative; unknown media types land as .bin so we never invent
// an executable suffix.
func extensionFor(mediaType string) string {
	mt := baseMediaType(mediaType)
	switch {
	case mt == "" || strings.HasPrefix(mt, "text/"):
		return ".txt"
	case mt == "application/json" || strings.HasSuffix(mt, "+json"):
		return ".json"
	default:
		return ".bin"
	}
}

// safePreview returns the leading n bytes of raw, truncated to a UTF-8
// boundary so we never emit half a multi-byte rune to the model.
func safePreview(raw []byte, n int) string {
	if n <= 0 || len(raw) == 0 {
		return ""
	}
	if len(raw) <= n {
		return string(raw)
	}
	head := raw[:n]
	for len(head) > 0 && !utf8.Valid(head) {
		head = head[:len(head)-1]
	}
	return string(head)
}

// truncatePointer enforces an upper bound on the pointer string we hand back
// in degraded mode; this ensures the channel/provider still receives a
// bounded payload even when we could not persist the artifact.
func truncatePointer(s string) string {
	return truncateUTF8Bytes(s, defaultToolPreviewBytes)
}

func truncateUTF8Bytes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	head := []byte(s)[:max]
	for len(head) > 0 && !utf8.Valid(head) {
		head = head[:len(head)-1]
	}
	return string(head)
}

// buildPointer composes the short text the model sees: an artifact reference
// header followed by the preview. The header is short enough to dominate the
// pointer length budget even when the preview is empty.
func buildPointer(relArtifact, preview, mediaType string, totalBytes int) string {
	header := fmt.Sprintf("[tool_output_artifact path=%s media=%s bytes=%d]",
		relArtifact, normalizeMedia(mediaType), totalBytes)
	header = truncateUTF8Bytes(header, maxToolPointerHeader)
	if preview == "" {
		return header
	}
	return header + "\n" + preview
}

func normalizeMedia(mediaType string) string {
	mt := strings.ToLower(strings.TrimSpace(mediaType))
	if mt == "" {
		return "text/plain"
	}
	return mt
}
