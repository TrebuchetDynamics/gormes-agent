package llm

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

type fakeTokenProvider struct {
	token string
	err   error
}

func (f *fakeTokenProvider) Token() (string, error) {
	return f.token, f.err
}

func TestGoogleCodeAssistHeadersUseTokenProvider(t *testing.T) {
	resolver := NewGoogleCodeAssistResolver("my-project", "free", &fakeTokenProvider{token: "fake-token"})
	headers, err := resolver.Headers()
	if err != nil {
		t.Fatalf("Headers error = %v", err)
	}
	if got := headers.Get("Authorization"); got != "Bearer fake-token" {
		t.Errorf("Authorization = %q, want Bearer fake-token", got)
	}
	if got := headers.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := headers.Get("User-Agent"); got != "gormes-agent/0.0.0" {
		t.Errorf("User-Agent = %q, want gormes-agent/0.0.0", got)
	}
}

func TestGoogleCodeAssistHeadersMissingProvider(t *testing.T) {
	resolver := NewGoogleCodeAssistResolver("my-project", "free", nil)
	headers, err := resolver.Headers()
	if err != nil {
		t.Fatalf("Headers error = %v", err)
	}
	if got := headers.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestGoogleCodeAssistHeadersProviderError(t *testing.T) {
	resolver := NewGoogleCodeAssistResolver("my-project", "free", &fakeTokenProvider{err: errors.New("refresh failed")})
	_, err := resolver.Headers()
	if err == nil {
		t.Fatal("expected error for token provider failure")
	}
}

func TestGoogleCodeAssistProjectContextPrecedence(t *testing.T) {
	tests := []struct {
		project  string
		tier     string
		wantCtx  string
		wantPaid bool
	}{
		{"my-project", "free", "my-project", false},
		{"", "free", "-", false},
		{"", "paid", "-", true},
		{"paid-project", "paid", "paid-project", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("project_%s_tier_%s", tt.project, tt.tier), func(t *testing.T) {
			r := NewGoogleCodeAssistResolver(tt.project, tt.tier, nil)
			if got := r.ProjectContext(); got != tt.wantCtx {
				t.Errorf("ProjectContext = %q, want %q", got, tt.wantCtx)
			}
			if got := r.RequiresExplicitProject(); got != tt.wantPaid {
				t.Errorf("RequiresExplicitProject = %v, want %v", got, tt.wantPaid)
			}
		})
	}
}

func TestGoogleCodeAssistErrorClassification(t *testing.T) {
	tests := []struct {
		status int
		want   ErrorClass
	}{
		{400, ClassFatal},
		{401, ClassFatal},
		{403, ClassFatal},
		{429, ClassRetryable},
		{500, ClassRetryable},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			class := classifyGoogleCodeAssistError(tt.status, `{"error":{"message":"test"}}`, http.Header{})
			if class.Class != tt.want {
				t.Errorf("class = %v, want %v", class.Class, tt.want)
			}
		})
	}
}

func TestGoogleCodeAssistProviderStatus(t *testing.T) {
	status := googleCodeAssistProviderStatus()
	if status.Provider != "google_code_assist" {
		t.Errorf("Provider = %q, want google_code_assist", status.Provider)
	}
	if status.Runtime != "gemini_cloudcode" {
		t.Errorf("Runtime = %q, want gemini_cloudcode", status.Runtime)
	}
}
