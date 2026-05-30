package video

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AnalyzeStatus string

const (
	AnalyzeStatusOK                 AnalyzeStatus = "video_analyze_ok"
	AnalyzeStatusUnsupportedVideo   AnalyzeStatus = "unsupported_video"
	AnalyzeStatusWorkspaceViolation AnalyzeStatus = "workspace_root_violation"
	AnalyzeStatusInvalidScheme      AnalyzeStatus = "invalid_scheme"
	AnalyzeStatusInvalidArgs        AnalyzeStatus = "invalid_arguments"
	AnalyzeStatusProviderError      AnalyzeStatus = "video_analyze_provider_error"
)

type AnalyzeResult struct {
	Success  bool          `json:"success"`
	Analysis string        `json:"analysis,omitempty"`
	Provider string        `json:"provider,omitempty"`
	Evidence AnalyzeStatus `json:"evidence"`
	Error    string        `json:"error,omitempty"`
	Metadata *AnalyzeMeta  `json:"metadata,omitempty"`
}

type AnalyzeMeta struct {
	VideoPath  string `json:"video_path,omitempty"`
	ModelUsed  string `json:"model_used,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

type AnalyzeProvider interface {
	Available(ctx context.Context) bool
	SupportsVision(ctx context.Context) bool
	Analyze(ctx context.Context, videoPath, prompt string) (AnalyzeResult, error)
}

type AnalyzeConfig struct {
	WorkspaceRoot   string
	ProviderFactory func(ctx context.Context) AnalyzeProvider
}

type AnalyzeTool struct {
	cfg AnalyzeConfig
}

func NewAnalyzeTool(cfg AnalyzeConfig) *AnalyzeTool {
	return &AnalyzeTool{cfg: cfg}
}

func (*AnalyzeTool) Name() string { return "video_analyze" }

func (*AnalyzeTool) Description() string {
	return "Analyzes a video file (local path or HTTPS URL) using a vision-capable model. Returns a text analysis of the video content. Requires a configured vision provider."
}

func (*AnalyzeTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"video_path": {
				"type": "string",
				"description": "Path to a local video file within the workspace, or an https:// URL."
			},
			"prompt": {
				"type": "string",
				"description": "What to analyze or look for in the video."
			}
		},
		"required": ["video_path", "prompt"]
	}`)
}

func (*AnalyzeTool) Timeout() time.Duration { return 5 * time.Minute }

func (t *AnalyzeTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		VideoPath string `json:"video_path"`
		Prompt    string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return marshalResult(AnalyzeResult{
			Success:  false,
			Evidence: AnalyzeStatusInvalidArgs,
			Error:    "invalid video_analyze args: " + err.Error(),
		})
	}
	in.VideoPath = strings.TrimSpace(in.VideoPath)
	in.Prompt = strings.TrimSpace(in.Prompt)
	if in.VideoPath == "" || in.Prompt == "" {
		return marshalResult(AnalyzeResult{
			Success:  false,
			Evidence: AnalyzeStatusInvalidArgs,
			Error:    "video_path and prompt are required",
		})
	}

	sanitized, evidence := t.validateAndSanitizePath(in.VideoPath)
	if evidence != "" {
		return marshalResult(AnalyzeResult{
			Success:  false,
			Evidence: evidence,
			Error:    fmt.Sprintf("video path rejected: %s", in.VideoPath),
		})
	}

	provider := t.resolveProvider(ctx)
	if provider == nil {
		return marshalResult(AnalyzeResult{
			Success:  false,
			Evidence: AnalyzeStatusUnsupportedVideo,
			Error:    "no video analysis provider configured",
		})
	}
	if !provider.SupportsVision(ctx) {
		return marshalResult(AnalyzeResult{
			Success:  false,
			Evidence: AnalyzeStatusUnsupportedVideo,
			Error:    "configured provider does not support vision",
		})
	}

	result, err := provider.Analyze(ctx, sanitized, in.Prompt)
	if err != nil {
		return marshalResult(AnalyzeResult{
			Success:  false,
			Evidence: AnalyzeStatusProviderError,
			Error:    redactVideoAnalyzeError(err.Error(), in.Prompt),
		})
	}
	return json.Marshal(result)
}

func (t *AnalyzeTool) validateAndSanitizePath(raw string) (string, AnalyzeStatus) {
	if looksLikeURL(raw) {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", AnalyzeStatusInvalidScheme
		}
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https":
			return raw, ""
		default:
			return "", AnalyzeStatusInvalidScheme
		}
	}

	clean := filepath.Clean(raw)
	if filepath.IsAbs(clean) {
		return "", AnalyzeStatusWorkspaceViolation
	}
	if strings.Contains(clean, "..") {
		return "", AnalyzeStatusWorkspaceViolation
	}

	if t.cfg.WorkspaceRoot == "" {
		return clean, ""
	}

	cleanRoot := filepath.Clean(t.cfg.WorkspaceRoot)
	resolved := filepath.Join(cleanRoot, clean)
	if !strings.HasPrefix(filepath.Clean(resolved), cleanRoot+string(os.PathSeparator)) && resolved != cleanRoot {
		return "", AnalyzeStatusWorkspaceViolation
	}
	return resolved, ""
}

func (t *AnalyzeTool) resolveProvider(ctx context.Context) AnalyzeProvider {
	if t.cfg.ProviderFactory == nil {
		return nil
	}
	provider := t.cfg.ProviderFactory(ctx)
	if provider == nil {
		return nil
	}
	if !provider.Available(ctx) {
		return nil
	}
	return provider
}

func looksLikeURL(s string) bool {
	parsed, err := url.Parse(s)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "ftp", "file", "data", "ws", "wss":
		return true
	}
	return false
}

func marshalResult(result AnalyzeResult) (json.RawMessage, error) {
	return json.Marshal(result)
}

func redactVideoAnalyzeError(text, prompt string) string {
	redacted := strings.TrimSpace(text)
	if len(redacted) > 240 {
		redacted = redacted[:240] + "..."
	}
	if redacted == "" {
		return "redacted video analysis error"
	}
	if prompt = strings.TrimSpace(prompt); prompt != "" {
		redacted = strings.ReplaceAll(redacted, prompt, "[redacted_prompt]")
	}
	return redacted
}
