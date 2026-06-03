package gormescli

import (
	"context"
	"io"
	"net/http"

	appmodel "github.com/TrebuchetDynamics/gormes-agent/internal/app/model"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

const (
	ModelChoiceSuggestionLimitDefault   = appmodel.SuggestionLimitDefault
	ModelChoiceSuggestionLimitUnlimited = appmodel.SuggestionLimitUnlimited
)

type ModelChoicePromptOptions = appmodel.PromptOptions

type ModelPickerSuggestionSet = appmodel.SuggestionSet

type OpenRouterModelCatalogFetchFunc = appmodel.OpenRouterModelCatalogFetchFunc

func DefaultModelCatalogSuggestions(provider string) []string {
	return appmodel.DefaultModelCatalogSuggestions(provider)
}

func DefaultModelPickerSuggestionSet(provider string, fetcher OpenRouterModelCatalogFetchFunc) ModelPickerSuggestionSet {
	return appmodel.DefaultModelPickerSuggestionSet(provider, fetcher)
}

func FetchOpenRouterModelCatalog(ctx context.Context, url string, client *http.Client) ([]string, error) {
	return appmodel.FetchOpenRouterModelCatalog(ctx, url, client)
}

func PromptModelChoiceText(in io.Reader, out io.Writer, provider string, current string, models []string) (string, error) {
	return appmodel.PromptModelChoiceText(in, out, provider, current, models)
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
