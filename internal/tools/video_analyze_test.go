package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeVideoAnalyzeProvider is a test-only provider that echoes back the
// video path and prompt as proof of routing, simulating a vision-capable
// model doing video analysis.
type fakeVideoAnalyzeProvider struct {
	available   bool
	visionCapab bool
	result      VideoAnalyzeResult
	err         error
}

func (p *fakeVideoAnalyzeProvider) Available(_ context.Context) bool { return p.available }
func (p *fakeVideoAnalyzeProvider) SupportsVision(_ context.Context) bool {
	return p.visionCapab
}
func (p *fakeVideoAnalyzeProvider) Analyze(_ context.Context, path, prompt string) (VideoAnalyzeResult, error) {
	return p.result, p.err
}

func TestVideoAnalyze_RoutesToVisionCapableProvider(t *testing.T) {
	provider := &fakeVideoAnalyzeProvider{
		available:   true,
		visionCapab: true,
		result: VideoAnalyzeResult{
			Success:  true,
			Evidence: VideoAnalyzeStatusOK,
			Analysis: "fake analysis: the video shows a cat",
			Provider: "fake-vision",
		},
	}
	tool := NewVideoAnalyzeTool(VideoAnalyzeConfig{
		ProviderFactory: func(_ context.Context) VideoAnalyzeProvider { return provider },
	})
	ctx := context.Background()
	args, _ := json.Marshal(map[string]any{
		"video_path": "https://example.com/video.mp4",
		"prompt":     "what is in this video?",
	})
	raw, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result VideoAnalyzeResult
	if jsonErr := json.Unmarshal(raw, &result); jsonErr != nil {
		t.Fatalf("invalid result JSON: %v", jsonErr)
	}
	if !result.Success {
		t.Fatalf("expected success, got evidence=%s error=%s", result.Evidence, result.Error)
	}
	if result.Analysis != "fake analysis: the video shows a cat" {
		t.Errorf("analysis = %q", result.Analysis)
	}
	if result.Provider != "fake-vision" {
		t.Errorf("provider = %q", result.Provider)
	}
}

func TestVideoAnalyze_NonVisionProviderUnsupportedError(t *testing.T) {
	provider := &fakeVideoAnalyzeProvider{
		available:   true,
		visionCapab: false, // vision NOT supported
	}
	tool := NewVideoAnalyzeTool(VideoAnalyzeConfig{
		ProviderFactory: func(_ context.Context) VideoAnalyzeProvider { return provider },
	})
	ctx := context.Background()
	args, _ := json.Marshal(map[string]any{
		"video_path": "https://example.com/video.mp4",
		"prompt":     "analyze this",
	})
	raw, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result VideoAnalyzeResult
	if jsonErr := json.Unmarshal(raw, &result); jsonErr != nil {
		t.Fatalf("invalid result JSON: %v", jsonErr)
	}
	if result.Success {
		t.Fatal("expected failure for non-vision provider")
	}
	if result.Evidence != VideoAnalyzeStatusUnsupportedVideo {
		t.Errorf("evidence = %q, want %q", result.Evidence, VideoAnalyzeStatusUnsupportedVideo)
	}
	if !strings.Contains(result.Error, "vision") {
		t.Errorf("error should mention vision capability: %s", result.Error)
	}
}

func TestVideoAnalyze_LocalPathSanitization(t *testing.T) {
	tool := NewVideoAnalyzeTool(VideoAnalyzeConfig{
		WorkspaceRoot: filepath.Join(t.TempDir(), "ws"),
	})
	ctx := context.Background()

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "directory traversal rejected",
			path: "../../../etc/passwd",
			want: string(VideoAnalyzeStatusWorkspaceViolation),
		},
		{
			name: "absolute path outside workspace rejected",
			path: "/etc/passwd",
			want: string(VideoAnalyzeStatusWorkspaceViolation),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]any{
				"video_path": tt.path,
				"prompt":     "analyze this",
			})
			raw, err := tool.Execute(ctx, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var result VideoAnalyzeResult
			if jsonErr := json.Unmarshal(raw, &result); jsonErr != nil {
				t.Fatalf("invalid result JSON: %v", jsonErr)
			}
			if result.Success {
				t.Fatal("expected failure for unsafe path")
			}
			if string(result.Evidence) != tt.want {
				t.Errorf("evidence = %q, want %q", result.Evidence, tt.want)
			}
		})
	}
}

func TestVideoAnalyze_URLSchemeAllowlist(t *testing.T) {
	provider := &fakeVideoAnalyzeProvider{
		available:   true,
		visionCapab: true,
		result: VideoAnalyzeResult{
			Success:  true,
			Evidence: VideoAnalyzeStatusOK,
			Analysis: "ok",
		},
	}
	tool := NewVideoAnalyzeTool(VideoAnalyzeConfig{
		ProviderFactory: func(_ context.Context) VideoAnalyzeProvider { return provider },
	})
	ctx := context.Background()

	tests := []struct {
		name    string
		video   string
		success bool
	}{
		{"http allowed", "http://example.com/video.mp4", true},
		{"https allowed", "https://example.com/video.mp4", true},
		{"file rejected", "file:///etc/passwd", false},
		{"ftp rejected", "ftp://example.com/video.mp4", false},
		{"data rejected", "data:text/html,hello", false},
		{"relative local path allowed", "video.mp4", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]any{
				"video_path": tt.video,
				"prompt":     "analyze",
			})
			raw, err := tool.Execute(ctx, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var result VideoAnalyzeResult
			if jsonErr := json.Unmarshal(raw, &result); jsonErr != nil {
				t.Fatalf("invalid result JSON: %v", jsonErr)
			}
			if tt.success && !result.Success {
				if result.Evidence == VideoAnalyzeStatusInvalidScheme {
					t.Errorf("scheme %q unexpectedly rejected", tt.video)
				} else {
					t.Errorf("unexpected failure: evidence=%s error=%s", result.Evidence, result.Error)
				}
			}
			if !tt.success && result.Success {
				t.Errorf("scheme %q should have been rejected", tt.video)
			}
		})
	}
}

func TestVideoAnalyze_LocalPathWithExistingFile(t *testing.T) {
	ws := t.TempDir()
	videoPath := filepath.Join(ws, "test.mp4")
	if err := os.WriteFile(videoPath, []byte("fake-video-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	provider := &fakeVideoAnalyzeProvider{
		available:   true,
		visionCapab: true,
		result: VideoAnalyzeResult{
			Success:  true,
			Evidence: VideoAnalyzeStatusOK,
			Analysis: "analyzed local file",
			Provider: "fake-vision",
		},
	}
	tool := NewVideoAnalyzeTool(VideoAnalyzeConfig{
		WorkspaceRoot:   ws,
		ProviderFactory: func(_ context.Context) VideoAnalyzeProvider { return provider },
	})
	ctx := context.Background()
	args, _ := json.Marshal(map[string]any{
		"video_path": "test.mp4", // relative path inside workspace
		"prompt":     "what is in this video?",
	})
	raw, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result VideoAnalyzeResult
	if jsonErr := json.Unmarshal(raw, &result); jsonErr != nil {
		t.Fatalf("invalid result JSON: %v", jsonErr)
	}
	if !result.Success {
		t.Fatalf("expected success for local file in workspace: evidence=%s error=%s", result.Evidence, result.Error)
	}
}

func TestVideoAnalyze_MissingRequiredArgs(t *testing.T) {
	tool := NewVideoAnalyzeTool(VideoAnalyzeConfig{})
	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing video_path", map[string]any{"prompt": "test"}},
		{"missing prompt", map[string]any{"video_path": "test.mp4"}},
		{"empty video_path", map[string]any{"video_path": "", "prompt": "test"}},
		{"empty prompt", map[string]any{"video_path": "test.mp4", "prompt": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(tt.args)
			raw, err := tool.Execute(ctx, args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var result VideoAnalyzeResult
			if jsonErr := json.Unmarshal(raw, &result); jsonErr != nil {
				t.Fatalf("invalid result JSON: %v", jsonErr)
			}
			if result.Success {
				t.Fatal("expected failure for missing args")
			}
			if result.Evidence != VideoAnalyzeStatusInvalidArgs {
				t.Errorf("evidence = %q, want %q", result.Evidence, VideoAnalyzeStatusInvalidArgs)
			}
		})
	}
}

func TestVideoAnalyze_NoProviderConfigured(t *testing.T) {
	tool := NewVideoAnalyzeTool(VideoAnalyzeConfig{
		// ProviderFactory is nil → no provider
		ProviderFactory: nil,
	})
	ctx := context.Background()
	args, _ := json.Marshal(map[string]any{
		"video_path": "https://example.com/video.mp4",
		"prompt":     "analyze",
	})
	raw, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result VideoAnalyzeResult
	if jsonErr := json.Unmarshal(raw, &result); jsonErr != nil {
		t.Fatalf("invalid result JSON: %v", jsonErr)
	}
	if result.Success {
		t.Fatal("expected failure when no provider configured")
	}
	if result.Evidence != VideoAnalyzeStatusUnsupportedVideo {
		t.Errorf("evidence = %q, want %q", result.Evidence, VideoAnalyzeStatusUnsupportedVideo)
	}
}

func TestVideoAnalyze_DescriptorContract(t *testing.T) {
	tool := NewVideoAnalyzeTool(VideoAnalyzeConfig{})
	if tool.Name() != "video_analyze" {
		t.Fatalf("tool name = %q, want video_analyze", tool.Name())
	}
	if tool.Description() == "" {
		t.Fatal("tool description must not be empty")
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema invalid JSON: %v\ndata=%s", err, tool.Schema())
	}
	for _, field := range []string{"video_path", "prompt"} {
		if _, ok := schema.Properties[field]; !ok {
			t.Fatalf("schema missing %q", field)
		}
	}
	if len(schema.Required) != 2 || schema.Required[0] != "video_path" || schema.Required[1] != "prompt" {
		t.Fatalf("required = %#v, want [video_path, prompt]", schema.Required)
	}
}
