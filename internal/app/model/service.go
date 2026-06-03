package model

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/modelchoice"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

const (
	SuggestionLimitDefault   = 5
	SuggestionLimitUnlimited = -1
)

type PromptOptions struct {
	Context         context.Context
	SuggestionLimit int
}

type SuggestionSet struct {
	Models         []string
	DegradedReason string
}

type OpenRouterModelCatalogFetchFunc func(context.Context) ([]string, error)

func DefaultModelCatalogSuggestions(provider string) []string {
	return llm.ProviderModelCatalogSuggestions(provider, nil)
}

func DefaultModelPickerSuggestions(provider string, fetcher OpenRouterModelCatalogFetchFunc) []string {
	return DefaultModelPickerSuggestionSet(provider, fetcher).Models
}

func DefaultModelPickerSuggestionSet(provider string, fetcher OpenRouterModelCatalogFetchFunc) SuggestionSet {
	provider = strings.TrimSpace(provider)
	if strings.EqualFold(provider, "openrouter") && fetcher != nil {
		if models, err := fetcher(context.Background()); err == nil && len(models) > 0 {
			return SuggestionSet{Models: models}
		}
	}
	foundProvider := false
	for _, entry := range llm.ListPickerProviders() {
		if !strings.EqualFold(entry.Slug, provider) {
			continue
		}
		foundProvider = true
		if len(entry.Models) > 0 {
			return SuggestionSet{Models: append([]string(nil), entry.Models...)}
		}
		break
	}
	fallback := DefaultModelCatalogSuggestions(provider)
	if len(fallback) > 0 {
		reason := "provider not in picker catalog"
		if foundProvider {
			reason = "picker catalog had no models"
		}
		return SuggestionSet{Models: fallback, DegradedReason: reason}
	}
	if foundProvider {
		return SuggestionSet{DegradedReason: "picker catalog had no models"}
	}
	return SuggestionSet{DegradedReason: "provider not in picker catalog"}
}

func FetchOpenRouterModelCatalog(ctx context.Context, url string, client *http.Client) ([]string, error) {
	if strings.TrimSpace(url) == "" {
		url = "https://openrouter.ai/api/v1/models"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if apiKey := firstNonEmpty(strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")), strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	if client == nil {
		client = &http.Client{Timeout: 8 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openrouter_models_http_status_%d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	entries, err := llm.ParseOpenRouterModelRegistry(raw, "openrouter-models-api")
	if err != nil {
		return nil, err
	}
	models := make([]string, 0, len(entries))
	for _, entry := range entries {
		model := strings.TrimSpace(entry.Model)
		if model != "" {
			models = append(models, model)
		}
	}
	return models, nil
}

func PromptModelChoiceText(in io.Reader, out io.Writer, provider string, current string, models []string) (string, error) {
	current = strings.TrimSpace(current)
	if len(models) > 0 {
		defaultIndex := IndexModelChoice(models, current)
		fmt.Fprintf(out, "Select model for %s:\n", provider)
		for i, model := range models {
			marker := " "
			if i == defaultIndex {
				marker = "→"
			}
			fmt.Fprintf(out, "  %s %d. %s\n", marker, i+1, model)
		}
		fmt.Fprintln(out)
		switch {
		case defaultIndex >= 0:
			fmt.Fprintf(out, "Choice [1-%d] (%d), custom model, or q to cancel: ", len(models), defaultIndex+1)
		case current != "":
			fmt.Fprintf(out, "Choice [1-%d], custom model [%s], or q to cancel: ", len(models), current)
		default:
			fmt.Fprintf(out, "Choice [1-%d], custom model, or q to cancel: ", len(models))
		}
	} else if current != "" {
		fmt.Fprintf(out, "Model for %s [%s] (or q to cancel): ", provider, current)
	} else {
		fmt.Fprintf(out, "Model for %s (or q to cancel): ", provider)
	}

	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return "", err
	}
	answer := strings.TrimSpace(line)
	if strings.EqualFold(answer, "q") || strings.EqualFold(answer, "cancel") {
		return "", cli.ErrModelPickerCancelled
	}
	if answer == "" {
		answer = current
	}
	if answer == "" {
		return "", cli.ErrSelectorNoMatch
	}
	if len(models) > 0 {
		if idx, parseErr := strconv.Atoi(answer); parseErr == nil {
			if idx < 1 || idx > len(models) {
				return "", fmt.Errorf("invalid model choice")
			}
			return models[idx-1], nil
		}
	}
	return answer, nil
}

func DefaultModelChoiceID(models []string, current string) string {
	return modelchoice.DefaultChoiceID(models, current)
}

func IndexModelChoice(models []string, current string) int {
	return modelchoice.IndexChoice(models, current)
}

func ModelCatalogSuggestionsForPrompt(suggestions []string, max int) []string {
	if max == SuggestionLimitUnlimited {
		max = modelchoice.UnlimitedSuggestions
	}
	return modelchoice.SuggestionsForPrompt(suggestions, max)
}

func BoundedModelCatalogSuggestions(suggestions []string, max int) []string {
	return modelchoice.BoundedSuggestions(suggestions, max)
}

func PersistModelSelectionToConfig(selection cli.Selection) error {
	provider := strings.TrimSpace(selection.Provider)
	model := strings.TrimSpace(selection.Model)
	if provider == "" || model == "" {
		return cli.ErrSelectorNoMatch
	}
	path := config.ConfigPath()
	document := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := toml.Unmarshal(data, &document); err != nil {
			return fmt.Errorf("decode config: %w", err)
		}
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read config: %w", err)
	}
	hermesSection, _ := document["hermes"].(map[string]any)
	if hermesSection == nil {
		hermesSection = map[string]any{}
	}
	hermesSection["model"] = model
	hermesSection["provider"] = provider
	document["hermes"] = hermesSection
	out, err := toml.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, out, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
