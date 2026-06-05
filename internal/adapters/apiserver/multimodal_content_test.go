package apiserver

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestAPIServerNormalizeMultimodal_TextOnlyCollapses(t *testing.T) {
	loop := &fakeTurnLoop{result: TurnResult{Content: "ok"}}
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: loop})

	rec := postJSON(t, srv.Handler(), "/v1/chat/completions", map[string]any{
		"model": "gormes-agent",
		"messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": "hello"},
			map[string]any{"type": "input_text", "text": "there"},
		}}},
	}, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	call := loop.lastCall()
	if call.UserMessage != "hello\nthere" {
		t.Fatalf("UserMessage = %q, want collapsed text", call.UserMessage)
	}
	if len(call.UserContentParts) != 0 {
		t.Fatalf("UserContentParts = %+v, want none for text-only content", call.UserContentParts)
	}
}

func TestAPIServerNormalizeMultimodal_ImageURLPreserved(t *testing.T) {
	loop := &fakeTurnLoop{result: TurnResult{Content: "ok"}}
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: loop})

	rec := postJSON(t, srv.Handler(), "/v1/chat/completions", map[string]any{
		"model": "gormes-agent",
		"messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": "describe this"},
			map[string]any{"type": "image_url", "image_url": map[string]any{
				"url":    "https://example.com/cat.png",
				"detail": "high",
			}},
		}}},
	}, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	call := loop.lastCall()
	want := []llm.MessageContentPart{
		{Type: "text", Text: "describe this"},
		{Type: "image_url", ImageURL: "https://example.com/cat.png", Detail: "high"},
	}
	assertContentParts(t, call.UserContentParts, want)
	if call.UserMessage != "describe this" {
		t.Fatalf("UserMessage = %q, want text component", call.UserMessage)
	}
}

func TestAPIServerNormalizeMultimodal_InputImageCanonicalized(t *testing.T) {
	loop := &fakeTurnLoop{result: TurnResult{Content: "ok"}}
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: loop})

	rec := postJSON(t, srv.Handler(), "/v1/responses", map[string]any{
		"model": "gormes-agent",
		"input": []any{map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "input_text", "text": "Describe."},
				map[string]any{"type": "input_image", "image_url": "https://example.com/cat.png"},
			},
		}},
	}, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	call := loop.lastCall()
	want := []llm.MessageContentPart{
		{Type: "text", Text: "Describe."},
		{Type: "image_url", ImageURL: "https://example.com/cat.png"},
	}
	assertContentParts(t, call.UserContentParts, want)
}

func TestAPIServerNormalizeMultimodal_ImageOnlyVisiblePayload(t *testing.T) {
	loop := &fakeTurnLoop{result: TurnResult{Content: "ok"}}
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: loop})

	rec := postJSON(t, srv.Handler(), "/v1/chat/completions", map[string]any{
		"model": "gormes-agent",
		"messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "image_url", "image_url": map[string]any{
				"url": "data:image/png;base64,AAAA",
			}},
		}}},
	}, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	call := loop.lastCall()
	want := []llm.MessageContentPart{{Type: "image_url", ImageURL: "data:image/png;base64,AAAA"}}
	assertContentParts(t, call.UserContentParts, want)
	if call.UserMessage != "" {
		t.Fatalf("UserMessage = %q, want empty text for image-only input", call.UserMessage)
	}
}

func TestAPIServerNormalizeMultimodal_RejectsFilesAndBadSchemes(t *testing.T) {
	tests := []struct {
		name    string
		content []any
		code    string
	}{
		{
			name:    "file",
			content: []any{map[string]any{"type": "file", "file": map[string]any{"file_id": "f_1"}}},
			code:    "unsupported_content_type",
		},
		{
			name:    "input_file",
			content: []any{map[string]any{"type": "input_file", "file_id": "f_1"}},
			code:    "unsupported_content_type",
		},
		{
			name:    "non_image_data_url",
			content: []any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:text/plain;base64,SGVsbG8="}}},
			code:    "unsupported_content_type",
		},
		{
			name:    "ftp_url",
			content: []any{map[string]any{"type": "image_url", "image_url": map[string]any{"url": "ftp://example.com/x.png"}}},
			code:    "invalid_image_url",
		},
		{
			name:    "missing_url",
			content: []any{map[string]any{"type": "image_url", "image_url": map[string]any{}}},
			code:    "invalid_image_url",
		},
		{
			name:    "unknown_part",
			content: []any{map[string]any{"type": "audio", "audio": map[string]any{}}},
			code:    "unsupported_content_type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loop := &fakeTurnLoop{}
			srv := NewServer(Config{ModelName: "gormes-agent", Loop: loop})
			rec := postJSON(t, srv.Handler(), "/v1/chat/completions", map[string]any{
				"model": "gormes-agent",
				"messages": []any{map[string]any{
					"role":    "user",
					"content": tt.content,
				}},
			}, nil)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
			var got struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("error envelope JSON: %v", err)
			}
			if got.Error.Code != tt.code {
				t.Fatalf("error.code = %q, want %q", got.Error.Code, tt.code)
			}
			if loop.callCount() != 0 {
				t.Fatalf("turn loop calls = %d, want 0", loop.callCount())
			}
		})
	}
}

func assertContentParts(t *testing.T, got, want []llm.MessageContentPart) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(ContentParts) = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ContentParts[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}
