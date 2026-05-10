package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var tinyVisionPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x04, 0x00, 0x00, 0x00, 0xb5, 0x1c, 0x0c,
	0x02, 0x00, 0x00, 0x00, 0x0b, 0x49, 0x44, 0x41,
	0x54, 0x78, 0xda, 0x63, 0x64, 0x60, 0x00, 0x00,
	0x00, 0x06, 0x00, 0x02, 0x30, 0x81, 0xd0, 0x2f,
	0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44,
	0xae, 0x42, 0x60, 0x82,
}

func TestVisionAnalyzeNativeLocalFileReturnsMultimodalEnvelope(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "tiny.png")
	if err := os.WriteFile(img, tinyVisionPNG, 0o600); err != nil {
		t.Fatal(err)
	}

	tool := NewVisionAnalyzeTool()
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"image_url":`+quoteVisionJSON(img)+`,"question":"what text is present?"}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	var env struct {
		Multimodal  bool   `json:"_multimodal"`
		TextSummary string `json:"text_summary"`
		Content     []struct {
			Type     string `json:"type"`
			Text     string `json:"text"`
			ImageURL struct {
				URL string `json:"url"`
			} `json:"image_url"`
		} `json:"content"`
		Meta struct {
			SizeBytes int64 `json:"size_bytes"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal envelope %s: %v", raw, err)
	}
	if !env.Multimodal {
		t.Fatalf("_multimodal = false, want true: %s", raw)
	}
	if !strings.Contains(env.TextSummary, "Image attached natively") {
		t.Fatalf("text_summary = %q, want native attachment summary", env.TextSummary)
	}
	if len(env.Content) != 2 {
		t.Fatalf("content len = %d, want 2: %+v", len(env.Content), env.Content)
	}
	if env.Content[0].Type != "text" || !strings.Contains(env.Content[0].Text, "what text is present?") {
		t.Fatalf("text part = %+v, want question preserved", env.Content[0])
	}
	wantURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(tinyVisionPNG)
	if env.Content[1].Type != "image_url" || env.Content[1].ImageURL.URL != wantURL {
		t.Fatalf("image part = %+v, want data URL %q", env.Content[1], wantURL)
	}
	if env.Meta.SizeBytes != int64(len(tinyVisionPNG)) {
		t.Fatalf("size_bytes = %d, want %d", env.Meta.SizeBytes, len(tinyVisionPNG))
	}
}

func TestVisionAnalyzeNativeRejectsMissingAndUnsupportedSources(t *testing.T) {
	dir := t.TempDir()
	tool := NewVisionAnalyzeTool()

	tests := []struct {
		name string
		args string
		want string
	}{
		{name: "empty", args: `{}`, want: "vision_analyze_missing_image"},
		{name: "missing file", args: `{"image_url":` + quoteVisionJSON(filepath.Join(dir, "missing.png")) + `}`, want: "vision_analyze_invalid_source"},
		{name: "directory", args: `{"image_url":` + quoteVisionJSON(dir) + `}`, want: "vision_analyze_invalid_source"},
		{name: "unsupported scheme", args: `{"image_url":"ftp://example.test/image.png"}`, want: "vision_analyze_unsupported_scheme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := tool.Execute(context.Background(), json.RawMessage(tt.args))
			if err != nil {
				t.Fatalf("Execute returned Go error: %v", err)
			}
			var out struct {
				Success  bool   `json:"success"`
				Evidence string `json:"evidence"`
				Error    string `json:"error"`
			}
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("unmarshal error envelope %s: %v", raw, err)
			}
			if out.Success {
				t.Fatalf("success = true, want false: %s", raw)
			}
			if out.Evidence != tt.want {
				t.Fatalf("evidence = %q, want %q (error=%q)", out.Evidence, tt.want, out.Error)
			}
		})
	}
}

func quoteVisionJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
