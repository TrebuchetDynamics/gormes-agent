package gormescli

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

	appmodel "github.com/TrebuchetDynamics/gormes-agent/internal/app/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/modelcontract"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
)

const (
	ModelChoiceSuggestionLimitDefault   = appmodel.SuggestionLimitDefault
	ModelChoiceSuggestionLimitUnlimited = appmodel.SuggestionLimitUnlimited
)

type ModelChoicePromptOptions = appmodel.PromptOptions

type ModelPickerSuggestionSet = appmodel.SuggestionSet

type OpenRouterModelCatalogFetchFunc = appmodel.OpenRouterModelCatalogFetchFunc

type ModelCommandSeams = modelcontract.Seams

var openRouterModelCatalogFetcher OpenRouterModelCatalogFetchFunc = fetchOpenRouterModelCatalog
var openRouterModelsURL = "https://openrouter.ai/api/v1/models"
var openRouterModelsHTTPClient = &http.Client{Timeout: 8 * time.Second}

func SetOpenRouterModelCatalogFetcherForTest(fetcher OpenRouterModelCatalogFetchFunc) func() {
	old := openRouterModelCatalogFetcher
	openRouterModelCatalogFetcher = fetcher
	return func() { openRouterModelCatalogFetcher = old }
}

func NewModelCommand() *cobra.Command {
	return NewModelCommandWithSeams(DefaultModelCommandSeams())
}

func NewModelCommandWithSeams(seams ModelCommandSeams) *cobra.Command {
	chooseModel := modelcontract.NormalizedModelChooser(seams)
	cmd := &cobra.Command{
		Use:          "model",
		Short:        "Interactively select the active model/provider",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			picker := cli.NewModelPicker(cli.ModelPickerOptions{
				IsTTY:            seams.IsTTY,
				LoadCurrent:      seams.LoadCurrent,
				ListProviders:    seams.ListProviders,
				ChooseProvider:   seams.ChooseProvider,
				ChooseModel:      chooseModel,
				PersistSelection: seams.PersistSelection,
			})
			selection, err := picker.Pick(cmd.Context())
			if err != nil {
				return fmt.Errorf("gormes model: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "model selection saved: provider=%s model=%s\n", selection.Provider, selection.Model)
			fmt.Fprintf(cmd.OutOrStdout(), "Provider auth was not changed. If credentials are missing, run: gormes auth add %s\n", selection.Provider)
			return nil
		},
	}
	return cmd
}

func DefaultModelCommandSeams() ModelCommandSeams {
	return ModelCommandSeams{
		IsTTY: stdinIsTTY,
		LoadCurrent: func() (cli.ProviderModel, error) {
			cfg, err := config.Load(nil)
			if err != nil {
				return cli.ProviderModel{}, err
			}
			return cli.ProviderModel{Provider: cfg.Hermes.Provider, Model: cfg.Hermes.Model}, nil
		},
		ListProviders: DefaultModelProviderEntries,
		ChooseProvider: func(entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
			return PromptProviderChoice(os.Stdin, os.Stdout, entries, defaultIndex)
		},
		ChooseModel: func(provider string, current string) (string, error) {
			return PromptModelChoice(os.Stdin, os.Stdout, provider, current, DefaultModelPickerSuggestions(provider))
		},
		PersistSelection: PersistModelSelectionToConfig,
	}
}

func stdinIsTTY() bool {
	return StdinIsTerminal(os.Stdin)
}

func StdinIsTerminal(file *os.File) bool {
	if file == nil {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func DefaultModelProviderEntries() ([]cli.ProviderMenuEntry, error) {
	return cli.HermesModelProviderMenu(), nil
}

func PromptProviderChoice(in *os.File, out io.Writer, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
	if in != nil && term.IsTerminal(int(in.Fd())) {
		selected, err := RunTUIPick(context.Background(), in, out, "Choose a provider", ProviderPickerChoices(entries), fmt.Sprintf("%d", defaultIndex))
		if err != nil && !TUIPickShouldFallback(err) {
			return -1, err
		}
		if err == nil && selected == "" {
			return -1, cli.ErrModelPickerCancelled
		}
		if err != nil {
			return PromptProviderChoiceText(in, out, entries, defaultIndex)
		}
		idx, _ := strconv.Atoi(selected)
		if idx >= 0 && idx < len(entries) {
			return idx, nil
		}
		return -1, fmt.Errorf("invalid provider choice")
	}
	return PromptProviderChoiceText(in, out, entries, defaultIndex)
}

func ProviderPickerChoices(entries []cli.ProviderMenuEntry) []TUIPickChoice {
	choices := make([]TUIPickChoice, len(entries))
	for i, entry := range entries {
		choices[i] = TUIPickChoice{ID: strconv.Itoa(i), Label: entry.Label}
	}
	return choices
}

func PromptProviderChoiceText(in *os.File, out io.Writer, entries []cli.ProviderMenuEntry, defaultIndex int) (int, error) {
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

func DefaultModelCatalogSuggestions(provider string) []string {
	return appmodel.DefaultModelCatalogSuggestions(provider)
}

func DefaultModelPickerSuggestions(provider string) []string {
	return DefaultModelPickerSuggestionSet(provider).Models
}

func DefaultModelPickerSuggestionSet(provider string) ModelPickerSuggestionSet {
	return appmodel.DefaultModelPickerSuggestionSet(provider, openRouterModelCatalogFetcher)
}

func FetchOpenRouterModelCatalog(ctx context.Context, url string, client *http.Client) ([]string, error) {
	return appmodel.FetchOpenRouterModelCatalog(ctx, url, client)
}

func fetchOpenRouterModelCatalog(ctx context.Context) ([]string, error) {
	return FetchOpenRouterModelCatalog(ctx, openRouterModelsURL, openRouterModelsHTTPClient)
}

func PromptModelChoice(in io.Reader, out io.Writer, provider string, current string, suggestions []string) (string, error) {
	return PromptModelChoiceWithOptions(in, out, provider, current, suggestions, ModelChoicePromptOptions{
		SuggestionLimit: ModelChoiceSuggestionLimitDefault,
	})
}

func PromptModelChoiceWithOptions(in io.Reader, out io.Writer, provider string, current string, suggestions []string, opts ModelChoicePromptOptions) (string, error) {
	// For the Bubble Tea TUI path, show all available models (search UI
	// handles large lists naturally). Fall back to the capped suggestion
	// list only for the text-mode prompt.
	allModels := ModelCatalogSuggestionsForPrompt(suggestions, ModelChoiceSuggestionLimitUnlimited)
	if stdin, ok := in.(*os.File); ok && term.IsTerminal(int(stdin.Fd())) && len(allModels) > 0 {
		ctx := opts.Context
		if ctx == nil {
			ctx = context.Background()
		}
		// Use the searchable picker for lists larger than the old 5-model
		// default. Small lists still work fine with search.
		var stepOpts []setupwizard.StepOption
		if len(allModels) > ModelChoiceSuggestionLimitDefault {
			stepOpts = append(stepOpts, setupwizard.WithSearchChoices())
		}
		selected, err := RunTUIPickWithOptions(ctx, stdin, out, "Select model for "+provider, ModelPickerChoices(allModels), DefaultModelChoiceID(allModels, current), stepOpts...)
		if err == nil {
			if selected == "" {
				return "", cli.ErrModelPickerCancelled
			}
			return selected, nil
		}
		if !TUIPickShouldFallback(err) {
			return "", err
		}
	}
	models := ModelCatalogSuggestionsForPrompt(suggestions, opts.SuggestionLimit)
	return PromptModelChoiceText(in, out, provider, current, models)
}

func PromptModelChoiceText(in io.Reader, out io.Writer, provider string, current string, models []string) (string, error) {
	return appmodel.PromptModelChoiceText(in, out, provider, current, models)
}

func ModelPickerChoices(models []string) []TUIPickChoice {
	choices := make([]TUIPickChoice, 0, len(models))
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
		choices = append(choices, TUIPickChoice{ID: model, Label: model})
	}
	return choices
}

func DefaultModelChoiceID(models []string, current string) string {
	return appmodel.DefaultModelChoiceID(models, current)
}

func IndexModelChoice(models []string, current string) int {
	return appmodel.IndexModelChoice(models, current)
}

func ModelCatalogSuggestionsForPrompt(suggestions []string, max int) []string {
	return appmodel.ModelCatalogSuggestionsForPrompt(suggestions, max)
}

func BoundedModelCatalogSuggestions(suggestions []string, max int) []string {
	return appmodel.BoundedModelCatalogSuggestions(suggestions, max)
}

func PersistModelSelectionToConfig(selection cli.Selection) error {
	return appmodel.PersistModelSelectionToConfig(selection)
}

func RunSetupActiveProviderModelPicker(cmd *cobra.Command, current cli.ProviderModel) error {
	provider := strings.TrimSpace(current.Provider)
	if provider == "" {
		return cli.ErrSelectorNoMatch
	}
	suggestions := DefaultModelPickerSuggestionSet(provider)
	if suggestions.DegradedReason != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Model catalog degraded for %s: %s; accepting free-text model.\n", provider, suggestions.DegradedReason)
	}
	model, err := PromptModelChoiceWithOptions(cmd.InOrStdin(), cmd.OutOrStdout(), provider, current.Model, suggestions.Models, ModelChoicePromptOptions{
		Context:         cmd.Context(),
		SuggestionLimit: ModelChoiceSuggestionLimitUnlimited,
	})
	if err != nil {
		return err
	}
	model = llm.NormalizeProviderModelID(provider, model)
	if err := PersistModelSelectionToConfig(cli.Selection{Provider: provider, Model: model}); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "model selection saved: provider=%s model=%s\n", provider, model)
	return nil
}

func NormalizeProviderModelID(provider, model string) string {
	return llm.NormalizeProviderModelID(provider, model)
}
