package config

import inferenceconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/inference"

type InferenceValueSource = inferenceconfig.ValueSource

const (
	InferenceValueSourceUnset  = inferenceconfig.ValueSourceUnset
	InferenceValueSourceFlag   = inferenceconfig.ValueSourceFlag
	InferenceValueSourceEnv    = inferenceconfig.ValueSourceEnv
	InferenceValueSourceConfig = inferenceconfig.ValueSourceConfig
)

type InferenceRequest struct {
	Config       Config
	ModelFlag    string
	ProviderFlag string
	LookupEnv    func(string) (string, bool)
	CommandLabel string
}

type OneshotInferenceRequest = InferenceRequest
type TUIInferenceRequest = InferenceRequest

type InferenceResolution = inferenceconfig.Resolution

type OneshotInferenceResolution = InferenceResolution
type TUIInferenceResolution = InferenceResolution

// ResolveOneshotInference applies the scripted-chat inference precedence:
// flag > GORMES_INFERENCE_* env > config defaults. A provider override without
// a flag/env model is rejected so a stale configured model is not silently
// paired with a different provider.
func ResolveOneshotInference(req OneshotInferenceRequest) (OneshotInferenceResolution, error) {
	inner := inferenceRequest(req)
	return inferenceconfig.ResolveOneshot(inner)
}

func ResolveTUIInference(req TUIInferenceRequest) (TUIInferenceResolution, error) {
	inner := inferenceRequest(req)
	return inferenceconfig.ResolveTUI(inner)
}

func ResolveInference(req InferenceRequest) (InferenceResolution, error) {
	return inferenceconfig.Resolve(inferenceRequest(req))
}

func inferenceRequest(req InferenceRequest) inferenceconfig.Request {
	return inferenceconfig.Request{
		ConfigModel:    req.Config.Hermes.Model,
		ConfigProvider: req.Config.Hermes.Provider,
		ModelFlag:      req.ModelFlag,
		ProviderFlag:   req.ProviderFlag,
		LookupEnv:      req.LookupEnv,
		CommandLabel:   req.CommandLabel,
	}
}

func resolveProviderDefaultModel(cfg *Config) {
	provider, model, source, _ := inferenceconfig.ResolveProviderDefault(cfg.Hermes.Provider, cfg.Hermes.Model)
	if source == "" {
		return
	}
	cfg.Hermes.Provider = provider
	cfg.Hermes.Model = model
	cfg.Hermes.ModelResolutionSource = source
}
