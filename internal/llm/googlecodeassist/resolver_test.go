package googlecodeassist

import (
	"errors"
	"fmt"
	"testing"
)

type fakeTokenProvider struct {
	token string
	err   error
}

func (f *fakeTokenProvider) Token() (string, error) {
	return f.token, f.err
}

func TestResolverHeadersUseTokenProvider(t *testing.T) {
	resolver := NewResolver("my-project", "free", &fakeTokenProvider{token: "fake-token"})
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

func TestResolverHeadersMissingProvider(t *testing.T) {
	resolver := NewResolver("my-project", "free", nil)
	headers, err := resolver.Headers()
	if err != nil {
		t.Fatalf("Headers error = %v", err)
	}
	if got := headers.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestResolverHeadersProviderError(t *testing.T) {
	resolver := NewResolver("my-project", "free", &fakeTokenProvider{err: errors.New("refresh failed")})
	_, err := resolver.Headers()
	if err == nil {
		t.Fatal("expected error for token provider failure")
	}
}

func TestResolverProjectContextPrecedence(t *testing.T) {
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
			r := NewResolver(tt.project, tt.tier, nil)
			if got := r.ProjectContext(); got != tt.wantCtx {
				t.Errorf("ProjectContext = %q, want %q", got, tt.wantCtx)
			}
			if got := r.RequiresExplicitProject(); got != tt.wantPaid {
				t.Errorf("RequiresExplicitProject = %v, want %v", got, tt.wantPaid)
			}
		})
	}
}
