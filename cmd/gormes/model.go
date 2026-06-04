package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
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
	modelChoiceSuggestionLimitDefault   = gormescli.ModelChoiceSuggestionLimitDefault
	modelChoiceSuggestionLimitUnlimited = gormescli.ModelChoiceSuggestionLimitUnlimited
)

type modelChoicePromptOptions = gormescli.ModelChoicePromptOptions

type modelPickerSuggestionSet = gormescli.ModelPickerSuggestionSet

type openRouterModelCatalogFetchFunc = gormescli.OpenRouterModelCatalogFetchFunc

var openRouterModelCatalogFetcher openRouterModelCatalogFetchFunc = fetchOpenRouterModelCatalog
var openRouterModelsURL = "https://openrouter.ai/api/v1/models"
var openRouterModelsHTTPClient = &http.Client{Timeout: 8 * time.Second}

func newModelCommand() *cobra.Command {
	return newModelCommandWithSeams(defaultModelCommandSeams())
}

func newModelCommandWithSeams(seams modelCommandSeams) *cobra.Command {
	return providermodule.NewModelCommandWithSeams(providerModelCommandSeams(seams))
}

func providerModelCommandSeams(seams modelCommandSeams) providermodule.ModelCommandSeams {
	return providermodule.ModelCommandSeams{
		IsTTY:            seams.IsTTY,
		LoadCurrent:      seams.LoadCurrent,
		ListProviders:    seams.ListProviders,
		ChooseProvider:   seams.ChooseProvider,
		ChooseModel:      seams.ChooseModel,
		PersistSelection: seams.PersistSelection,
	}
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
	return gormescli.DefaultModelCatalogSuggestions(provider)
}

func defaultModelPickerSuggestions(provider string) []string {
	return defaultModelPickerSuggestionSet(provider).Models
}

func defaultModelPickerSuggestionSet(provider string) modelPickerSuggestionSet {
	return gormescli.DefaultModelPickerSuggestionSet(provider, openRouterModelCatalogFetcher)
}

func fetchOpenRouterModelCatalog(ctx context.Context) ([]string, error) {
	return gormescli.FetchOpenRouterModelCatalog(ctx, openRouterModelsURL, openRouterModelsHTTPClient)
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
	return gormescli.PromptModelChoiceText(in, out, provider, current, models)
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
	return gormescli.DefaultModelChoiceID(models, current)
}

func indexModelChoice(models []string, current string) int {
	return gormescli.IndexModelChoice(models, current)
}

func modelCatalogSuggestionsForPrompt(suggestions []string, max int) []string {
	return gormescli.ModelCatalogSuggestionsForPrompt(suggestions, max)
}

func boundedModelCatalogSuggestions(suggestions []string, max int) []string {
	return gormescli.BoundedModelCatalogSuggestions(suggestions, max)
}

func persistModelSelectionToConfig(selection cli.Selection) error {
	return gormescli.PersistModelSelectionToConfig(selection)
}
