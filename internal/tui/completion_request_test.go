package tui

import "testing"

func TestTUICompletionRequest(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want TUICompletionRequest
		ok   bool
	}{
		{
			name: "real slash command",
			in:   "/help",
			want: TUICompletionRequest{Method: TUICompletionSlash, Text: "/help", ReplaceFrom: 1},
			ok:   true,
		},
		{
			name: "leading absolute path",
			in:   "/home/d/Desktop/agenda/CrimsonRed/.hermes/plans/2026-05-04-HANDOFF-NEXT.md",
			want: TUICompletionRequest{
				Method:      TUICompletionPath,
				Word:        "/home/d/Desktop/agenda/CrimsonRed/.hermes/plans/2026-05-04-HANDOFF-NEXT.md",
				ReplaceFrom: 0,
			},
			ok: true,
		},
		{
			name: "trailing absolute path token",
			in:   "read /home/d/Desktop/file.md",
			want: TUICompletionRequest{Method: TUICompletionPath, Word: "/home/d/Desktop/file.md", ReplaceFrom: 5},
			ok:   true,
		},
		{
			name: "relative path token",
			in:   "open ./notes/today.md",
			want: TUICompletionRequest{Method: TUICompletionPath, Word: "./notes/today.md", ReplaceFrom: 5},
			ok:   true,
		},
		{name: "model picker owns model", in: "/model", ok: false},
		{name: "provider picker owns provider args", in: "/provider openai", ok: false},
		{name: "plain text", in: "hello there", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := CompletionRequestForInput(tt.in)
			if ok != tt.ok {
				t.Fatalf("CompletionRequestForInput(%q) ok = %v, want %v", tt.in, ok, tt.ok)
			}
			if !ok {
				return
			}
			if got != tt.want {
				t.Fatalf("CompletionRequestForInput(%q) = %+v, want %+v", tt.in, got, tt.want)
			}
		})
	}
}
