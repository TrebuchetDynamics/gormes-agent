package codex

import "testing"

func TestChatGPTBackendBaseURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "exact backend", raw: "https://chatgpt.com/backend-api/codex", want: true},
		{name: "nested backend path", raw: "https://chatgpt.com/backend-api/codex/responses", want: true},
		{name: "trailing dot host", raw: "https://chatgpt.com./backend-api/codex", want: true},
		{name: "wrong host", raw: "https://api.openai.com/backend-api/codex", want: false},
		{name: "wrong path", raw: "https://chatgpt.com/v1/responses", want: false},
		{name: "invalid", raw: "http://%zz", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChatGPTBackendBaseURL(tt.raw); got != tt.want {
				t.Fatalf("ChatGPTBackendBaseURL(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestChatGPTAccountID(t *testing.T) {
	token := "header.eyJodHRwczovL2FwaS5vcGVuYWkuY29tL2F1dGgiOnsiY2hhdGdwdF9hY2NvdW50X2lkIjoiIGFjY3QtY29kZXgtaGVhZGVycyAifX0.signature"
	if got := ChatGPTAccountID(token); got != "acct-codex-headers" {
		t.Fatalf("ChatGPTAccountID() = %q, want acct-codex-headers", got)
	}

	for _, bad := range []string{"", "not-jwt", "header..signature", "header.%%%%.signature", "header.e30.signature"} {
		if got := ChatGPTAccountID(bad); got != "" {
			t.Fatalf("ChatGPTAccountID(%q) = %q, want empty", bad, got)
		}
	}
}
