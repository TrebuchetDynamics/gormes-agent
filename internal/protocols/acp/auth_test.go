package acp

import "testing"

func TestDetectProviderRequiresProviderAndCredential(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		apiKey   string
		want     string
		ok       bool
	}{
		{name: "provider and key", provider: "OpenRouter", apiKey: "sk-test", want: "openrouter", ok: true},
		{name: "provider without key", provider: "openrouter", apiKey: "  ", want: "", ok: false},
		{name: "key without provider", provider: "", apiKey: "sk-test", want: "", ok: false},
		{name: "blank provider", provider: "  ", apiKey: "sk-test", want: "", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DetectProvider(tt.provider, tt.apiKey)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("DetectProvider(%q, %q) = (%q, %v), want (%q, %v)", tt.provider, tt.apiKey, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestBuildAuthMethodsAlwaysAdvertisesTerminalSetup(t *testing.T) {
	methods := BuildAuthMethods("", false)
	if len(methods) != 1 {
		t.Fatalf("methods = %#v, want terminal setup only", methods)
	}
	if methods[0].ID != TerminalSetupAuthMethodID || methods[0].Type != "terminal" {
		t.Fatalf("terminal method = %#v", methods[0])
	}
	if len(methods[0].Args) != 1 || methods[0].Args[0] != "--setup" {
		t.Fatalf("terminal args = %#v, want --setup", methods[0].Args)
	}
}

func TestBuildAuthMethodsAddsProviderWhenCredentialAvailable(t *testing.T) {
	methods := BuildAuthMethods("OpenRouter", true)
	if len(methods) != 2 {
		t.Fatalf("methods = %#v, want provider plus terminal", methods)
	}
	if methods[0].ID != "openrouter" || methods[0].Type != "" {
		t.Fatalf("provider method = %#v", methods[0])
	}
	if methods[1].ID != TerminalSetupAuthMethodID {
		t.Fatalf("terminal method = %#v", methods[1])
	}
}

func TestBuildAuthMethodsOmitsProviderWithoutCredential(t *testing.T) {
	methods := BuildAuthMethods("openrouter", false)
	if len(methods) != 1 || methods[0].ID != TerminalSetupAuthMethodID {
		t.Fatalf("methods = %#v, want terminal setup only", methods)
	}
}
