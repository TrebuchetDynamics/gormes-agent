package routing

import "testing"

func TestResolveFastModeRequestOverrides(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		want   RequestOverrides
		wantOK bool
	}{
		{name: "openai_gpt_prefix", model: "gpt-5.4", want: RequestOverrides{ServiceTier: "priority"}, wantOK: true},
		{name: "openai_vendor_prefix", model: "openai/gpt-4.1", want: RequestOverrides{ServiceTier: "priority"}, wantOK: true},
		{name: "openai_o_prefix", model: "o3", want: RequestOverrides{ServiceTier: "priority"}, wantOK: true},
		{name: "codex_excluded", model: "gpt-5.3-codex", wantOK: false},
		{name: "claude_opus_46_dash", model: "claude-opus-4-6", want: RequestOverrides{Speed: "fast"}, wantOK: true},
		{name: "claude_opus_46_dot_vendor", model: "anthropic/claude-opus-4.6", want: RequestOverrides{Speed: "fast"}, wantOK: true},
		{name: "claude_sonnet_unsupported", model: "claude-sonnet-4-6", wantOK: false},
		{name: "claude_opus_47_unsupported", model: "claude-opus-4-7", wantOK: false},
		{name: "gemini_unsupported", model: "gemini-3-pro-preview", wantOK: false},
		{name: "empty_unsupported", model: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolveFastModeRequestOverrides(tt.model)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v; overrides = %+v", ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Fatalf("overrides = %+v, want %+v", got, tt.want)
			}
		})
	}
}
