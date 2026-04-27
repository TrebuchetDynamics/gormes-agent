package hermes

import (
	"errors"
	"testing"
	"time"
)

func TestProviderTimeoutConfig_LoadFailureReturnsUnset(t *testing.T) {
	loader := func() (map[string]any, error) {
		return nil, errors.New("config unavailable")
	}

	for _, tt := range []struct {
		name string
		got  ProviderTimeoutResolution
	}{
		{
			name: "request",
			got:  ResolveProviderRequestTimeout(loader, "nous", "hermes-3"),
		},
		{
			name: "stale",
			got:  ResolveProviderStaleTimeout(loader, "nous", "hermes-3"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertProviderTimeoutResolution(t, tt.got, 0, ProviderTimeoutConfigUnavailable)
		})
	}
}

func TestProviderTimeoutConfig_MissingProviderReturnsUnset(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		provider string
	}{
		{
			name:     "nil config",
			config:   nil,
			provider: "nous",
		},
		{
			name:     "missing providers",
			config:   map[string]any{},
			provider: "nous",
		},
		{
			name:     "nil providers",
			config:   map[string]any{"providers": nil},
			provider: "nous",
		},
		{
			name:     "providers not map",
			config:   map[string]any{"providers": []any{"nous"}},
			provider: "nous",
		},
		{
			name: "blank provider id",
			config: map[string]any{"providers": map[string]any{
				"nous": map[string]any{"request_timeout_seconds": 5},
			}},
			provider: "",
		},
		{
			name: "missing provider id",
			config: map[string]any{"providers": map[string]any{
				"nous": map[string]any{"request_timeout_seconds": 5},
			}},
			provider: "anthropic",
		},
		{
			name: "non-map provider block",
			config: map[string]any{"providers": map[string]any{
				"nous": "request_timeout_seconds = 5",
			}},
			provider: "nous",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := func() (map[string]any, error) {
				return tt.config, nil
			}
			assertProviderTimeoutResolution(t, ResolveProviderRequestTimeout(loader, tt.provider, "hermes-3"), 0, ProviderTimeoutUnset)
			assertProviderTimeoutResolution(t, ResolveProviderStaleTimeout(loader, tt.provider, "hermes-3"), 0, ProviderTimeoutUnset)
		})
	}
}

func TestProviderTimeoutConfig_ParsesRequestAndStaleTimeouts(t *testing.T) {
	loader := func() (map[string]any, error) {
		return map[string]any{"providers": map[string]any{
			"nous": map[string]any{
				"request_timeout_seconds": 2,
				"stale_timeout_seconds":   "1500ms",
				"models": map[string]any{
					"hermes-3": map[string]any{
						"timeout_seconds":       "2.5s",
						"stale_timeout_seconds": 3.25,
					},
				},
			},
		}}, nil
	}

	tests := []struct {
		name string
		got  ProviderTimeoutResolution
		want time.Duration
	}{
		{
			name: "provider request numeric seconds",
			got:  ResolveProviderRequestTimeout(loader, "nous", ""),
			want: 2 * time.Second,
		},
		{
			name: "provider stale duration string",
			got:  ResolveProviderStaleTimeout(loader, "nous", ""),
			want: 1500 * time.Millisecond,
		},
		{
			name: "model request duration string",
			got:  ResolveProviderRequestTimeout(loader, "nous", "hermes-3"),
			want: 2500 * time.Millisecond,
		},
		{
			name: "model stale numeric seconds",
			got:  ResolveProviderStaleTimeout(loader, "nous", "hermes-3"),
			want: 3250 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertProviderTimeoutResolution(t, tt.got, tt.want, ProviderTimeoutConfigured)
		})
	}
}

func TestProviderTimeoutConfig_InvalidValuesFailClosed(t *testing.T) {
	tests := []struct {
		name string
		got  func(ProviderTimeoutConfigLoader) ProviderTimeoutResolution
		cfg  map[string]any
	}{
		{
			name: "negative request timeout",
			got: func(loader ProviderTimeoutConfigLoader) ProviderTimeoutResolution {
				return ResolveProviderRequestTimeout(loader, "nous", "")
			},
			cfg: timeoutConfigForProvider(map[string]any{"request_timeout_seconds": -1}),
		},
		{
			name: "zero stale timeout",
			got: func(loader ProviderTimeoutConfigLoader) ProviderTimeoutResolution {
				return ResolveProviderStaleTimeout(loader, "nous", "")
			},
			cfg: timeoutConfigForProvider(map[string]any{"stale_timeout_seconds": 0}),
		},
		{
			name: "non-numeric request timeout",
			got: func(loader ProviderTimeoutConfigLoader) ProviderTimeoutResolution {
				return ResolveProviderRequestTimeout(loader, "nous", "")
			},
			cfg: timeoutConfigForProvider(map[string]any{"request_timeout_seconds": "later"}),
		},
		{
			name: "overflow stale timeout",
			got: func(loader ProviderTimeoutConfigLoader) ProviderTimeoutResolution {
				return ResolveProviderStaleTimeout(loader, "nous", "")
			},
			cfg: timeoutConfigForProvider(map[string]any{"stale_timeout_seconds": "999999999999999999999999"}),
		},
		{
			name: "invalid model request does not fall back to provider default",
			got: func(loader ProviderTimeoutConfigLoader) ProviderTimeoutResolution {
				return ResolveProviderRequestTimeout(loader, "nous", "broken-model")
			},
			cfg: timeoutConfigForProvider(map[string]any{
				"request_timeout_seconds": 20,
				"models": map[string]any{
					"broken-model": map[string]any{"timeout_seconds": -5},
				},
			}),
		},
		{
			name: "invalid model stale does not fall back to provider default",
			got: func(loader ProviderTimeoutConfigLoader) ProviderTimeoutResolution {
				return ResolveProviderStaleTimeout(loader, "nous", "broken-model")
			},
			cfg: timeoutConfigForProvider(map[string]any{
				"stale_timeout_seconds": 20,
				"models": map[string]any{
					"broken-model": map[string]any{"stale_timeout_seconds": "later"},
				},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := func() (map[string]any, error) {
				return tt.cfg, nil
			}
			assertProviderTimeoutResolution(t, tt.got(loader), 0, ProviderTimeoutConfigInvalid)
		})
	}
}

func timeoutConfigForProvider(provider map[string]any) map[string]any {
	return map[string]any{"providers": map[string]any{"nous": provider}}
}

func assertProviderTimeoutResolution(t *testing.T, got ProviderTimeoutResolution, wantTimeout time.Duration, wantEvidence ProviderTimeoutEvidence) {
	t.Helper()
	if got.Timeout != wantTimeout {
		t.Fatalf("Timeout = %v, want %v", got.Timeout, wantTimeout)
	}
	if got.Evidence != wantEvidence {
		t.Fatalf("Evidence = %q, want %q", got.Evidence, wantEvidence)
	}
}
