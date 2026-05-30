//go:build !slim

package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/imagegen"

// Default FAL model catalog values mirroring Hermes image_generation_tool.py.
const (
	DefaultFLUXModel         = imagegen.DefaultFLUXModel
	DefaultAspectRatio       = imagegen.DefaultAspectRatio
	DefaultOutputFormat      = imagegen.DefaultOutputFormat
	DefaultNumInferenceSteps = imagegen.DefaultNumInferenceSteps
)

// ValidAspectRatios lists the valid aspect ratio options.
var ValidAspectRatios = imagegen.ValidAspectRatios

// FALModels is the catalog of supported FAL models mirroring Hermes FAL_MODELS.
var FALModels = imagegen.FALModels

type FALModelMetadata = imagegen.FALModelMetadata
type ImageGenConfig = imagegen.ImageGenConfig
type ImageGenRequest = imagegen.ImageGenRequest
type ImageGenResult = imagegen.ImageGenResult
type ImageProviderRequest = imagegen.ImageProviderRequest
type ImageProviderResult = imagegen.ImageProviderResult
type ImageGenerator = imagegen.ImageGenerator
type ImageGenRunner = imagegen.ImageGenRunner
type FALGenImageProvider = imagegen.FALGenImageProvider
type ImageGenTool = imagegen.ImageGenTool
type ImageGenProvider = imagegen.ImageGenProvider
type ImageGenActiveProvider = imagegen.ImageGenActiveProvider
type ImageGenProviderRegistry = imagegen.ImageGenProviderRegistry
type ImageGenPluginDiscovery = imagegen.ImageGenPluginDiscovery
type ImageGenPluginDiscoveryFunc = imagegen.ImageGenPluginDiscoveryFunc

func NewImageGenRunner(cfg ImageGenConfig, providers map[string]ImageGenerator) *ImageGenRunner {
	return imagegen.NewImageGenRunner(cfg, providers)
}

func NewImageGenRunnerWithRegistry(cfg ImageGenConfig, registry *ImageGenProviderRegistry) *ImageGenRunner {
	return imagegen.NewImageGenRunnerWithRegistry(cfg, registry)
}

func NewFALGenImageProvider(apiKey string) *FALGenImageProvider {
	return imagegen.NewFALGenImageProvider(apiKey)
}

func NewImageGenProviderRegistry() *ImageGenProviderRegistry {
	return imagegen.NewImageGenProviderRegistry()
}

func NewImageGenTool(runner *ImageGenRunner) *ImageGenTool {
	return imagegen.NewImageGenTool(runner)
}
