package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestModelOpenCodeSuggestionsUseCatalogFloor(t *testing.T) {
	got := defaultModelCatalogSuggestions("opencode-go")
	for _, want := range []string{"mimo-v2-pro", "kimi-k2.6"} {
		if !containsStringModelCatalog(got, want) {
			t.Fatalf("defaultModelCatalogSuggestions(opencode-go) missing %q: %#v", want, got)
		}
	}
}

func TestFetchOpenRouterModelCatalogReadsModelsAPI(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "or-test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/models" {
			t.Fatalf("path = %q, want /api/v1/models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer or-test-key" {
			t.Fatalf("Authorization = %q, want bearer OpenRouter key", got)
		}
		_, _ = w.Write([]byte(`{"data":[
			{"id":"live/first"},
			{"id":"live/second"},
			{"id":""}
		]}`))
	}))
	defer server.Close()

	oldURL := openRouterModelsURL
	oldClient := openRouterModelsHTTPClient
	openRouterModelsURL = server.URL + "/api/v1/models"
	openRouterModelsHTTPClient = server.Client()
	t.Cleanup(func() {
		openRouterModelsURL = oldURL
		openRouterModelsHTTPClient = oldClient
	})

	got, err := fetchOpenRouterModelCatalog(context.Background())
	if err != nil {
		t.Fatalf("fetchOpenRouterModelCatalog: %v", err)
	}
	if want := []string{"live/first", "live/second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("models = %#v, want %#v", got, want)
	}
}

func TestSetupModelPickerUsesLiveOpenRouterModelCatalog(t *testing.T) {
	withOpenRouterModelCatalogFetcherForTest(t, func(context.Context) ([]string, error) {
		return []string{"live/alpha", "moonshotai/kimi-k2.6", "live/omega"}, nil
	})

	var out strings.Builder
	in := strings.NewReader("3\n")
	got, err := promptModelChoiceWithOptions(in, &out, "openrouter", "moonshotai/kimi-k2.6", defaultModelPickerSuggestions("openrouter"), modelChoicePromptOptions{
		SuggestionLimit: modelChoiceSuggestionLimitUnlimited,
	})
	if err != nil {
		t.Fatalf("promptModelChoiceWithOptions: %v", err)
	}
	if got != "live/omega" {
		t.Fatalf("model = %q, want live/omega", got)
	}
	text := out.String()
	for _, want := range []string{
		"Select model for openrouter:",
		"1. live/alpha",
		"2. moonshotai/kimi-k2.6",
		"3. live/omega",
		"Choice [1-3] (2), custom model, or q to cancel:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "inclusionai/ring-2.6-1t:free") {
		t.Fatalf("prompt used curated fallback instead of live OpenRouter catalog:\n%s", text)
	}
}

func withOpenRouterModelCatalogFetcherForTest(t *testing.T, fetcher openRouterModelCatalogFetchFunc) {
	t.Helper()
	oldFetcher := openRouterModelCatalogFetcher
	openRouterModelCatalogFetcher = fetcher
	t.Cleanup(func() {
		openRouterModelCatalogFetcher = oldFetcher
	})
}

func openRouterModelCatalogOfflineForTest(context.Context) ([]string, error) {
	return nil, errors.New("openrouter model catalog unavailable")
}

func TestModelCatalogPromptShowsBoundedSelectableList(t *testing.T) {
	var out strings.Builder
	in := strings.NewReader("2\n")
	got, err := promptModelChoice(in, &out, "opencode-go", "", []string{
		"mimo-v2-pro",
		"kimi-k2.6",
		"glm-5.1",
		"glm-5",
		"mimo-v2-omni",
		"minimax-m2.7",
	})
	if err != nil {
		t.Fatalf("promptModelChoice: %v", err)
	}
	if got != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6", got)
	}
	text := out.String()
	for _, want := range []string{
		"Select model for opencode-go:",
		"1. mimo-v2-pro",
		"2. kimi-k2.6",
		"5. mimo-v2-omni",
		"Choice [1-5], custom model, or q to cancel:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Suggested models for opencode-go:") {
		t.Fatalf("prompt output still uses comma-separated suggestions:\n%s", text)
	}
	if strings.Contains(text, "minimax-m2.7") {
		t.Fatalf("prompt output was not bounded:\n%s", text)
	}
}

func TestModelCatalogPromptAcceptsCustomModel(t *testing.T) {
	var out strings.Builder
	in := strings.NewReader("custom-model\n")
	got, err := promptModelChoice(in, &out, "opencode-go", "", []string{"kimi-k2.6"})
	if err != nil {
		t.Fatalf("promptModelChoice: %v", err)
	}
	if got != "custom-model" {
		t.Fatalf("model = %q, want custom-model", got)
	}
}

func containsStringModelCatalog(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
