package llm

import (
	"reflect"
	"testing"
)

func TestOllamaCloudSuffixNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "colon cloud", in: "kimi-k2.6:cloud", want: "kimi-k2.6"},
		{name: "dash cloud", in: "qwen3-coder:480b-cloud", want: "qwen3-coder:480b"},
		{name: "unsuffixed", in: "nemotron-3-nano:30b", want: "nemotron-3-nano:30b"},
		{name: "empty", in: "", want: ""},
		{name: "middle cloud is preserved", in: "cloud-router:latest", want: "cloud-router:latest"},
		{name: "nonterminal dash cloud is preserved", in: "research-cloud:latest", want: "research-cloud:latest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeOllamaCloudModelID(tt.in)
			if got != tt.want {
				t.Fatalf("NormalizeOllamaCloudModelID(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeProviderModelIDStripsOnlyOllamaCloudSuffixes(t *testing.T) {
	t.Parallel()

	if got := NormalizeProviderModelID("ollama-cloud", "kimi-k2.6:cloud"); got != "kimi-k2.6" {
		t.Fatalf("ollama-cloud normalized model = %q, want kimi-k2.6", got)
	}
	if got := NormalizeProviderModelID("anthropic", "kimi-k2.6:cloud"); got != "kimi-k2.6:cloud" {
		t.Fatalf("non-Ollama model = %q, want suffix preserved", got)
	}
	if got := NormalizeProviderModelID("ollama_cloud", "qwen3-coder:480b-cloud"); got != "qwen3-coder:480b" {
		t.Fatalf("ollama_cloud alias normalized model = %q, want qwen3-coder:480b", got)
	}
}

func TestMergeOllamaCloudModelEntriesDedupesModelsDevSuffixes(t *testing.T) {
	t.Parallel()

	live := []ModelRegistryEntry{
		{
			Provider:         "ollama-cloud",
			Model:            "kimi-k2.6",
			ProviderFamily:   "ollama-cloud-live",
			ModelFamily:      "kimi",
			RawContextWindow: 128000,
		},
		{
			Provider:         "ollama-cloud",
			Model:            "glm-5.1",
			ProviderFamily:   "ollama-cloud-live",
			ModelFamily:      "glm",
			RawContextWindow: 64000,
		},
	}
	modelsDev := []ModelRegistryEntry{
		{
			Provider:         "ollama-cloud",
			Model:            "kimi-k2.6:cloud",
			ProviderFamily:   "models-dev",
			ModelFamily:      "kimi-from-models-dev",
			RawContextWindow: 256000,
		},
		{
			Provider:         "ollama-cloud",
			Model:            "qwen3-coder:480b-cloud",
			ProviderFamily:   "models-dev",
			ModelFamily:      "qwen",
			RawContextWindow: 128000,
		},
	}

	got := MergeOllamaCloudModelEntries(live, modelsDev)
	var gotModels []string
	for _, entry := range got {
		gotModels = append(gotModels, entry.Model)
	}
	wantModels := []string{"kimi-k2.6", "glm-5.1", "qwen3-coder:480b"}
	if !reflect.DeepEqual(gotModels, wantModels) {
		t.Fatalf("merged models = %#v, want %#v", gotModels, wantModels)
	}
	if got[0].ProviderFamily != "ollama-cloud-live" || got[0].RawContextWindow != 128000 {
		t.Fatalf("first merged entry = %+v, want live metadata precedence", got[0])
	}
	for _, entry := range got {
		if entry.Model == "kimi-k2.6:cloud" || entry.Model == "qwen3-coder:480b-cloud" {
			t.Fatalf("merged entries include suffixed model ID: %+v", got)
		}
	}
}
