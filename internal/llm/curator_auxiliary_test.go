package llm

import "testing"

func TestCuratorAuxiliarySlot_ResolveMainFallback(t *testing.T) {
	main := ModelRoute{Provider: "openrouter", Model: "openai/gpt-5.5"}
	cases := []struct {
		name      string
		canonical CuratorAuxiliarySlot
		wantCode  CuratorAuxiliaryEvidenceCode
	}{
		{name: "missing", canonical: CuratorAuxiliarySlot{}, wantCode: CuratorAuxiliarySlotMissing},
		{name: "auto empty", canonical: CuratorAuxiliarySlot{Provider: "auto"}, wantCode: CuratorAuxiliaryAutoMain},
		{name: "provider only", canonical: CuratorAuxiliarySlot{Provider: "openrouter", APIKey: "must-not-leak"}, wantCode: CuratorAuxiliaryPartialFallback},
		{name: "model only", canonical: CuratorAuxiliarySlot{Provider: "auto", Model: "gpt-5.4-mini", BaseURL: "http://ignored/v1"}, wantCode: CuratorAuxiliaryPartialFallback},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveCuratorAuxiliary(CuratorAuxiliaryRequest{
				Main:      main,
				Canonical: tc.canonical,
			})
			if got.Route != main {
				t.Fatalf("Route = %#v, want main %#v", got.Route, main)
			}
			if got.ExplicitAPIKey != "" || got.ExplicitBaseURL != "" {
				t.Fatalf("fallback leaked aux credentials: api=%q base=%q", got.ExplicitAPIKey, got.ExplicitBaseURL)
			}
			if !got.HasEvidence(tc.wantCode) {
				t.Fatalf("Evidence = %#v, want %s", got.Evidence, tc.wantCode)
			}
			if tc.canonical.APIKey != "" && !got.HasEvidence(CuratorAuxiliarySecretStripped) {
				t.Fatalf("Evidence = %#v, want secret-stripped", got.Evidence)
			}
		})
	}
}

func TestCuratorAuxiliarySlot_OverrideWins(t *testing.T) {
	got := ResolveCuratorAuxiliary(CuratorAuxiliaryRequest{
		Main: ModelRoute{Provider: "openrouter", Model: "openai/gpt-5.5"},
		Canonical: CuratorAuxiliarySlot{
			Provider:  " custom ",
			Model:     " local-mini ",
			APIKey:    " sk-curator-only ",
			BaseURL:   " http://localhost:11434/v1 ",
			Timeout:   900,
			ExtraBody: map[string]any{"reasoning_effort": "low"},
		},
	})
	if got.Route != (ModelRoute{Provider: "custom", Model: "local-mini"}) {
		t.Fatalf("Route = %#v, want custom/local-mini", got.Route)
	}
	if got.ExplicitAPIKey != "sk-curator-only" || got.ExplicitBaseURL != "http://localhost:11434/v1" {
		t.Fatalf("explicit credentials = api:%q base:%q", got.ExplicitAPIKey, got.ExplicitBaseURL)
	}
	if got.Timeout != 900 || got.ExtraBody["reasoning_effort"] != "low" {
		t.Fatalf("timeout/extra_body = %d %#v", got.Timeout, got.ExtraBody)
	}
}

func TestCuratorAuxiliarySlot_LegacyFallbackAndNewSlotWins(t *testing.T) {
	main := ModelRoute{Provider: "openrouter", Model: "openai/gpt-5.5"}
	legacy := CuratorAuxiliarySlot{
		Provider: "openrouter",
		Model:    "legacy-model",
		APIKey:   "legacy-key",
		BaseURL:  "http://legacy/v1",
	}

	got := ResolveCuratorAuxiliary(CuratorAuxiliaryRequest{
		Main:   main,
		Legacy: legacy,
	})
	if got.Route != (ModelRoute{Provider: "openrouter", Model: "legacy-model"}) {
		t.Fatalf("legacy Route = %#v", got.Route)
	}
	if got.ExplicitAPIKey != "legacy-key" || got.ExplicitBaseURL != "http://legacy/v1" {
		t.Fatalf("legacy explicit credentials = api:%q base:%q", got.ExplicitAPIKey, got.ExplicitBaseURL)
	}
	if !got.HasEvidence(CuratorAuxiliaryLegacyConfig) {
		t.Fatalf("Evidence = %#v, want legacy evidence", got.Evidence)
	}

	got = ResolveCuratorAuxiliary(CuratorAuxiliaryRequest{
		Main: main,
		Canonical: CuratorAuxiliarySlot{
			Provider: "nous",
			Model:    "new-winner",
		},
		Legacy: legacy,
	})
	if got.Route != (ModelRoute{Provider: "nous", Model: "new-winner"}) {
		t.Fatalf("canonical Route = %#v, want new slot to win", got.Route)
	}
	if got.HasEvidence(CuratorAuxiliaryLegacyConfig) {
		t.Fatalf("Evidence = %#v, legacy must not be used when canonical is complete", got.Evidence)
	}
}
