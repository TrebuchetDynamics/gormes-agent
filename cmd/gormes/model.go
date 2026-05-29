package main

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
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

type modelCommandSeams struct {
	IsTTY            func() bool
	LoadCurrent      func() (cli.ProviderModel, error)
	ListProviders    func() ([]cli.ProviderMenuEntry, error)
	ChooseProvider   func(entries []cli.ProviderMenuEntry, defaultIndex int) (int, error)
	ChooseModel      func(provider string, current string) (string, error)
	PersistSelection func(cli.Selection) error
}

const (
	modelChoiceSuggestionLimitDefault   = 5
	modelChoiceSuggestionLimitUnlimited = -1
)

type modelChoicePromptOptions struct {
	Context         context.Context
	SuggestionLimit int
}

type modelPickerSuggestionSet struct {
	Models         []string
	DegradedReason string
}

type openRouterModelCatalogFetchFunc func(context.Context) ([]string, error)

var openRouterModelCatalogFetcher openRouterModelCatalogFetchFunc = fetchOpenRouterModelCatalog
var openRouterModelsURL = "https://openrouter.ai/api/v1/models"
var openRouterModelsHTTPClient = &http.Client{Timeout: 8 * time.Second}

func newModelCommand() *cobra.Command {
	return newModelCommandWithSeams(defaultModelCommandSeams())
}

func newModelCommandWithSeams(seams modelCommandSeams) *cobra.Command {
	return providermodule.NewModelCommandWithSeams(providerModelCommandSeams(seams))
}

func defaultModelCommandSeams() modelCommandSeams {
	return modelCommandSeams{
		IsTTY: isStdinTTY,
		LoadCurrent: func() (cli.ProviderModel, error) {
			cfg, err := config.Load(nil)
			if err != nil {
				return cli.ProviderModel{}, err
			}
			return cli.ProviderModel{Provider: cfg.Hermes.Provider, Model: cfg.Hermes.Model}, nil
		},
		ListProviders: defaultModelProviderEntries,
		ChooseProvider: func(entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
			return promptProviderChoice(os.Stdin, os.Stdout, entries, defaultIndex)
		},
		ChooseModel: func(provider string, current string) (string, error) {
			return promptModelChoice(os.Stdin, os.Stdout, provider, current, defaultModelPickerSuggestions(provider))
		},
		PersistSelection: persistModelSelectionToConfig,
	}
}

func isStdinTTY() bool {
	return stdinIsTerminal(os.Stdin)
}

func stdinIsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func defaultModelProviderEntries() ([]cli.ProviderMenuEntry, error) {
	return cli.HermesModelProviderMenu(), nil
}

func promptProviderChoice(in *os.File, out io.Writer, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
	if in != nil && term.IsTerminal(int(in.Fd())) {
		choices := make([]tuiPickChoice, len(entries))
		for i, e := range entries {
			choices[i] = tuiPickChoice{ID: fmt.Sprintf("%d", i), Label: e.Label}
		}
		selected, err := runBubbleTeaPick(context.Background(), in, out, "Choose a provider", choices, fmt.Sprintf("%d", defaultIndex))
		if err != nil && !bubbleTeaPickShouldFallback(err) {
			return -1, err
		}
		if err == nil && selected == "" {
			return -1, cli.ErrModelPickerCancelled
		}
		if err != nil {
			return promptProviderChoiceText(in, out, entries, defaultIndex)
		}
		idx, _ := strconv.Atoi(selected)
		if idx >= 0 && idx < len(entries) {
			return idx, nil
		}
		return -1, fmt.Errorf("invalid provider choice")
	}
	return promptProviderChoiceText(in, out, entries, defaultIndex)
}

func promptProviderChoiceText(in *os.File, out io.Writer, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
	cli.ClearScreen(out)
	cli.PrintHeader(out, "Choose a provider")
	for i, entry := range entries {
		num := fmt.Sprintf("%2d)", i+1)
		var marker, line string
		if i == defaultIndex {
			marker = cli.Yellow(out, "*")
			line = cli.BrightCyan(out, num) + " " + cli.Bold(out, entry.Label)
		} else {
			marker = " "
			line = cli.Dim(out, num) + " " + entry.Label
		}
		fmt.Fprintf(out, "%s %s\n", marker, line)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s ", cli.Bold(out, fmt.Sprintf("Select provider [%d] (or q to cancel):", defaultIndex+1)))
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return -1, err
	}
	answer := strings.TrimSpace(line)
	if answer == "" {
		return defaultIndex, nil
	}
	if strings.EqualFold(answer, "q") || strings.EqualFold(answer, "cancel") {
		return -1, cli.ErrModelPickerCancelled
	}
	idx, err := strconv.Atoi(answer)
	if err != nil || idx < 1 || idx > len(entries) {
		return -1, fmt.Errorf("invalid provider choice")
	}
	return idx - 1, nil
}

func defaultModelCatalogSuggestions(provider string) []string {
	return llm.ProviderModelCatalogSuggestions(provider, nil)
}

func defaultModelPickerSuggestions(provider string) []string {
	return defaultModelPickerSuggestionSet(provider).Models
}

func defaultModelPickerSuggestionSet(provider string) modelPickerSuggestionSet {
	provider = strings.TrimSpace(provider)
	if strings.EqualFold(provider, "openrouter") && openRouterModelCatalogFetcher != nil {
		if models, err := openRouterModelCatalogFetcher(context.Background()); err == nil && len(models) > 0 {
			return modelPickerSuggestionSet{Models: models}
		}
	}
	foundProvider := false
	for _, entry := range llm.ListPickerProviders() {
		if !strings.EqualFold(entry.Slug, provider) {
			continue
		}
		foundProvider = true
		if len(entry.Models) > 0 {
			return modelPickerSuggestionSet{Models: append([]string(nil), entry.Models...)}
		}
		break
	}
	fallback := defaultModelCatalogSuggestions(provider)
	if len(fallback) > 0 {
		reason := "provider not in picker catalog"
		if foundProvider {
			reason = "picker catalog had no models"
		}
		return modelPickerSuggestionSet{Models: fallback, DegradedReason: reason}
	}
	if foundProvider {
		return modelPickerSuggestionSet{DegradedReason: "picker catalog had no models"}
	}
	return modelPickerSuggestionSet{DegradedReason: "provider not in picker catalog"}
}

func fetchOpenRouterModelCatalog(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterModelsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if apiKey := firstNonEmpty(strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")), strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := openRouterModelsHTTPClient
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

func promptModelChoice(in io.Reader, out io.Writer, provider string, current string, suggestions []string) (string, error) {
	return promptModelChoiceWithOptions(in, out, provider, current, suggestions, modelChoicePromptOptions{
		SuggestionLimit: modelChoiceSuggestionLimitDefault,
	})
}

func promptModelChoiceWithOptions(in io.Reader, out io.Writer, provider string, current string, suggestions []string, opts modelChoicePromptOptions) (string, error) {
	// For the Bubble Tea TUI path, show all available models (search UI
	// handles large lists naturally). Fall back to the capped suggestion
	// list only for the text-mode prompt.
	allModels := modelCatalogSuggestionsForPrompt(suggestions, modelChoiceSuggestionLimitUnlimited)
	if stdin, ok := in.(*os.File); ok && term.IsTerminal(int(stdin.Fd())) && len(allModels) > 0 {
		ctx := opts.Context
		if ctx == nil {
			ctx = context.Background()
		}
		// Use the searchable picker for lists larger than the old 5-model
		// default. Small lists still work fine with search.
		var stepOpts []setupwizard.StepOption
		if len(allModels) > modelChoiceSuggestionLimitDefault {
			stepOpts = append(stepOpts, setupwizard.WithSearchChoices())
		}
		selected, err := runBubbleTeaPickWithOptions(ctx, stdin, out, "Select model for "+provider, modelPickerChoices(allModels), defaultModelChoiceID(allModels, current), stepOpts...)
		if err == nil {
			if selected == "" {
				return "", cli.ErrModelPickerCancelled
			}
			return selected, nil
		}
		if !bubbleTeaPickShouldFallback(err) {
			return "", err
		}
	}
	models := modelCatalogSuggestionsForPrompt(suggestions, opts.SuggestionLimit)
	return promptModelChoiceText(in, out, provider, current, models)
}

func promptModelChoiceText(in io.Reader, out io.Writer, provider string, current string, models []string) (string, error) {
	current = strings.TrimSpace(current)
	if len(models) > 0 {
		defaultIndex := indexModelChoice(models, current)
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

func modelPickerChoices(models []string) []tuiPickChoice {
	choices := make([]tuiPickChoice, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		key := strings.ToLower(model)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		choices = append(choices, tuiPickChoice{ID: model, Label: model})
	}
	return choices
}

func defaultModelChoiceID(models []string, current string) string {
	current = strings.TrimSpace(current)
	if current == "" {
		return ""
	}
	for _, model := range models {
		if strings.EqualFold(strings.TrimSpace(model), current) {
			return strings.TrimSpace(model)
		}
	}
	return ""
}

func indexModelChoice(models []string, current string) int {
	current = strings.TrimSpace(current)
	if current == "" {
		return -1
	}
	for i, model := range models {
		if strings.EqualFold(strings.TrimSpace(model), current) {
			return i
		}
	}
	return -1
}

func modelCatalogSuggestionsForPrompt(suggestions []string, max int) []string {
	if max == modelChoiceSuggestionLimitUnlimited {
		max = len(suggestions)
	}
	return boundedModelCatalogSuggestions(suggestions, max)
}

func boundedModelCatalogSuggestions(suggestions []string, max int) []string {
	if max <= 0 {
		return nil
	}
	out := make([]string, 0, min(len(suggestions), max))
	seen := map[string]struct{}{}
	for _, suggestion := range suggestions {
		suggestion = strings.TrimSpace(suggestion)
		key := strings.ToLower(suggestion)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, suggestion)
		if len(out) == max {
			return out
		}
	}
	return out
}

func persistModelSelectionToConfig(selection cli.Selection) error {
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
