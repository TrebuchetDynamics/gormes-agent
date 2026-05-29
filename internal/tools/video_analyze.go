package tools

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

type VideoAnalyzeStatus string

const (
	VideoAnalyzeStatusOK                 VideoAnalyzeStatus = "video_analyze_ok"
	VideoAnalyzeStatusUnsupportedVideo   VideoAnalyzeStatus = "unsupported_video"
	VideoAnalyzeStatusWorkspaceViolation VideoAnalyzeStatus = "workspace_root_violation"
	VideoAnalyzeStatusInvalidScheme      VideoAnalyzeStatus = "invalid_scheme"
	VideoAnalyzeStatusInvalidArgs        VideoAnalyzeStatus = "invalid_arguments"
	VideoAnalyzeStatusProviderError      VideoAnalyzeStatus = "video_analyze_provider_error"
)

type VideoAnalyzeResult struct {
	Success  bool               `json:"success"`
	Analysis string             `json:"analysis,omitempty"`
	Provider string             `json:"provider,omitempty"`
	Evidence VideoAnalyzeStatus `json:"evidence"`
	Error    string             `json:"error,omitempty"`
	Metadata *VideoAnalyzeMeta  `json:"metadata,omitempty"`
}

type VideoAnalyzeMeta struct {
	VideoPath  string `json:"video_path,omitempty"`
	ModelUsed  string `json:"model_used,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

type VideoAnalyzeProvider interface {
	Available(ctx context.Context) bool
	SupportsVision(ctx context.Context) bool
	Analyze(ctx context.Context, videoPath, prompt string) (VideoAnalyzeResult, error)
}

type VideoAnalyzeConfig struct {
	WorkspaceRoot   string
	ProviderFactory func(ctx context.Context) VideoAnalyzeProvider
}

type VideoAnalyzeTool struct {
	cfg VideoAnalyzeConfig
}

func NewVideoAnalyzeTool(cfg VideoAnalyzeConfig) *VideoAnalyzeTool {
	return &VideoAnalyzeTool{cfg: cfg}
}

func (*VideoAnalyzeTool) Name() string { return "video_analyze" }

func (*VideoAnalyzeTool) Description() string {
	return "Analyzes a video file (local path or HTTPS URL) using a vision-capable model. Returns a text analysis of the video content. Requires a configured vision provider."
}

func (*VideoAnalyzeTool) Schema() json.RawMessage {
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

func (*VideoAnalyzeTool) Timeout() time.Duration { return 5 * time.Minute }

func (t *VideoAnalyzeTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		VideoPath string `json:"video_path"`
		Prompt    string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return marshalResult(VideoAnalyzeResult{
			Success:  false,
			Evidence: VideoAnalyzeStatusInvalidArgs,
			Error:    "invalid video_analyze args: " + err.Error(),
		})
	}
	in.VideoPath = strings.TrimSpace(in.VideoPath)
	in.Prompt = strings.TrimSpace(in.Prompt)
	if in.VideoPath == "" || in.Prompt == "" {
		return marshalResult(VideoAnalyzeResult{
			Success:  false,
			Evidence: VideoAnalyzeStatusInvalidArgs,
			Error:    "video_path and prompt are required",
		})
	}

	sanitized, evidence := t.validateAndSanitizePath(in.VideoPath)
	if evidence != "" {
		return marshalResult(VideoAnalyzeResult{
			Success:  false,
			Evidence: evidence,
			Error:    fmt.Sprintf("video path rejected: %s", in.VideoPath),
		})
	}

	provider := t.resolveProvider(ctx)
	if provider == nil {
		return marshalResult(VideoAnalyzeResult{
			Success:  false,
			Evidence: VideoAnalyzeStatusUnsupportedVideo,
			Error:    "no video analysis provider configured",
		})
	}
	if !provider.SupportsVision(ctx) {
		return marshalResult(VideoAnalyzeResult{
			Success:  false,
			Evidence: VideoAnalyzeStatusUnsupportedVideo,
			Error:    "configured provider does not support vision",
		})
	}

	result, err := provider.Analyze(ctx, sanitized, in.Prompt)
	if err != nil {
		return marshalResult(VideoAnalyzeResult{
			Success:  false,
			Evidence: VideoAnalyzeStatusProviderError,
			Error:    redactVideoAnalyzeError(err.Error(), in.Prompt),
		})
	}
	return json.Marshal(result)
}

func (t *VideoAnalyzeTool) validateAndSanitizePath(raw string) (string, VideoAnalyzeStatus) {
	if looksLikeURL(raw) {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", VideoAnalyzeStatusInvalidScheme
		}
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https":
			return raw, ""
		default:
			return "", VideoAnalyzeStatusInvalidScheme
		}
	}

	clean := filepath.Clean(raw)
	if filepath.IsAbs(clean) {
		return "", VideoAnalyzeStatusWorkspaceViolation
	}
	if strings.Contains(clean, "..") {
		return "", VideoAnalyzeStatusWorkspaceViolation
	}

	if t.cfg.WorkspaceRoot == "" {
		return clean, ""
	}

	cleanRoot := filepath.Clean(t.cfg.WorkspaceRoot)
	resolved := filepath.Join(cleanRoot, clean)
	if !strings.HasPrefix(filepath.Clean(resolved), cleanRoot+string(os.PathSeparator)) && resolved != cleanRoot {
		return "", VideoAnalyzeStatusWorkspaceViolation
	}
	return resolved, ""
}

func (t *VideoAnalyzeTool) resolveProvider(ctx context.Context) VideoAnalyzeProvider {
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

func marshalResult(result VideoAnalyzeResult) (json.RawMessage, error) {
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
