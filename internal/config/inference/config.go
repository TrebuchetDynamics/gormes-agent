package inference

import (
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type ValueSource string

const (
	ValueSourceUnset  ValueSource = "unset"
	ValueSourceFlag   ValueSource = "flag"
	ValueSourceEnv    ValueSource = "env"
	ValueSourceConfig ValueSource = "config"
)

type Request struct {
	ConfigModel    string
	ConfigProvider string
	ModelFlag      string
	ProviderFlag   string
	LookupEnv      func(string) (string, bool)
	CommandLabel   string
}

type Resolution struct {
	Model                      string
	ModelSource                ValueSource
	Provider                   string
	ProviderSource             ValueSource
	ProviderAutoDetectRequired bool
	ModelResolutionSource      string
}

func ResolveOneshot(req Request) (Resolution, error) {
	req.CommandLabel = "gormes chat -q"
	return Resolve(req)
}

func ResolveTUI(req Request) (Resolution, error) {
	req.CommandLabel = "gormes tui"
	return Resolve(req)
}

func Resolve(req Request) (Resolution, error) {
	lookupEnv := req.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	model, modelSource := firstValue(
		candidate{value: req.ModelFlag, source: ValueSourceFlag},
		candidate{value: lookupEnvValue(lookupEnv, "GORMES_INFERENCE_MODEL"), source: ValueSourceEnv},
		candidate{value: req.ConfigModel, source: ValueSourceConfig},
	)
	provider, providerSource := firstValue(
		candidate{value: req.ProviderFlag, source: ValueSourceFlag},
		candidate{value: lookupEnvValue(lookupEnv, "GORMES_INFERENCE_PROVIDER"), source: ValueSourceEnv},
		candidate{value: req.ConfigProvider, source: ValueSourceConfig},
	)

	resolution := Resolution{
		Model:          model,
		ModelSource:    modelSource,
		Provider:       provider,
		ProviderSource: providerSource,
	}
	explicitModel := modelSource == ValueSourceFlag || modelSource == ValueSourceEnv
	explicitProvider := providerSource == ValueSourceFlag || providerSource == ValueSourceEnv
	if explicitProvider && !explicitModel {
		return resolution, providerRequiresExplicitModelError(req.CommandLabel, providerSource)
	}
	resolution.ProviderAutoDetectRequired = explicitModel && providerSource == ValueSourceUnset
	ResolveProviderDefaultModel(&resolution)
	return resolution, nil
}

func ResolveProviderDefault(provider, model string) (providerOut, modelOut, source string, resolved bool) {
	if !ShouldResolveProviderDefaultModel(provider, model) {
		if strings.TrimSpace(provider) != "" && strings.TrimSpace(model) != "" {
			return provider, model, "explicit_operator_config", false
		}
		return provider, model, "", false
	}
	resolution := llm.ResolveProviderDefaultModel(provider, llm.ProviderDefaultModelOptions{})
	if strings.TrimSpace(resolution.Model) == "" {
		return provider, model, "", false
	}
	return resolution.Provider, resolution.Model, string(resolution.Source), true
}

func ResolveProviderDefaultModel(resolution *Resolution) {
	if resolution == nil || !ShouldResolveProviderDefaultModel(resolution.Provider, resolution.Model) {
		return
	}
	defaultModel := llm.ResolveProviderDefaultModel(resolution.Provider, llm.ProviderDefaultModelOptions{})
	if strings.TrimSpace(defaultModel.Model) == "" {
		return
	}
	resolution.Provider = defaultModel.Provider
	resolution.Model = defaultModel.Model
	resolution.ModelResolutionSource = string(defaultModel.Source)
}

func ShouldResolveProviderDefaultModel(provider, model string) bool {
	if strings.TrimSpace(provider) == "" {
		return false
	}
	model = strings.TrimSpace(model)
	return model == "" || strings.EqualFold(model, "hermes-agent")
}

type candidate struct {
	value  string
	source ValueSource
}

func firstValue(candidates ...candidate) (string, ValueSource) {
	for _, candidate := range candidates {
		value := strings.TrimSpace(candidate.value)
		if value != "" {
			return value, candidate.source
		}
	}
	return "", ValueSourceUnset
}

func lookupEnvValue(lookup func(string) (string, bool), name string) string {
	value, ok := lookup(name)
	if !ok {
		return ""
	}
	return value
}

func providerRequiresExplicitModelError(commandLabel string, source ValueSource) error {
	commandLabel = strings.TrimSpace(commandLabel)
	if commandLabel == "" {
		commandLabel = "gormes inference"
	}
	if source == ValueSourceEnv {
		return fmt.Errorf("%s: GORMES_INFERENCE_PROVIDER requires --model or GORMES_INFERENCE_MODEL. Set both inference env vars, pass both flags, or neither to use your configured defaults", commandLabel)
	}
	return fmt.Errorf("%s: --provider requires --model (or GORMES_INFERENCE_MODEL). Pass both explicitly, or neither to use your configured defaults.", commandLabel)
}
