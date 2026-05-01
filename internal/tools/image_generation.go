//go:build !slim

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Default FAL model catalog values mirroring Hermes image_generation_tool.py
const (
	DefaultFLUXModel         = "fal-ai/flux-2/klein/9b"
	DefaultAspectRatio       = "landscape"
	DefaultOutputFormat      = "png"
	DefaultNumInferenceSteps = 4
)

// ValidAspectRatios lists the valid aspect ratio options.
var ValidAspectRatios = []string{"landscape", "square", "portrait"}

// FALModelMetadata describes a single FAL model entry.
type FALModelMetadata struct {
	Display   string
	Speed     string
	Price     string
	SizeStyle string // "image_size_preset" | "aspect_ratio" | "gpt_literal"
	Sizes     map[string]string
	Defaults  map[string]any
	Supports  map[string]bool
	Upscale   bool
}

// FALModels is the catalog of supported FAL models mirroring Hermes FAL_MODELS.
var FALModels = map[string]FALModelMetadata{
	"fal-ai/flux-2/klein/9b": {
		Display:   "FLUX 2 Klein 9B",
		Speed:     "<1s",
		Price:     "$0.006/MP",
		SizeStyle: "image_size_preset",
		Sizes: map[string]string{
			"landscape": "landscape_16_9",
			"square":    "square_hd",
			"portrait":  "portrait_16_9",
		},
		Defaults: map[string]any{
			"num_inference_steps":   4,
			"output_format":         "png",
			"enable_safety_checker": false,
		},
		Supports: map[string]bool{
			"prompt":                  true,
			"image_size":               true,
			"num_inference_steps":      true,
			"seed":                     true,
			"output_format":            true,
			"enable_safety_checker":    true,
		},
		Upscale: false,
	},
	"fal-ai/flux-2-pro": {
		Display:   "FLUX 2 Pro",
		Speed:     "~6s",
		Price:     "$0.03/MP",
		SizeStyle: "image_size_preset",
		Sizes: map[string]string{
			"landscape": "landscape_16_9",
			"square":    "square_hd",
			"portrait":  "portrait_16_9",
		},
		Defaults: map[string]any{
			"num_inference_steps":   50,
			"guidance_scale":       4.5,
			"num_images":           1,
			"output_format":         "png",
			"enable_safety_checker": false,
			"safety_tolerance":     "5",
			"sync_mode":            true,
		},
		Supports: map[string]bool{
			"prompt":                  true,
			"image_size":               true,
			"num_inference_steps":      true,
			"guidance_scale":           true,
			"num_images":               true,
			"output_format":            true,
			"enable_safety_checker":    true,
			"safety_tolerance":         true,
			"sync_mode":                true,
			"seed":                     true,
		},
		Upscale: true,
	},
	"fal-ai/z-image/turbo": {
		Display:   "Z-Image Turbo",
		Speed:     "~2s",
		Price:     "$0.005/MP",
		SizeStyle: "image_size_preset",
		Sizes: map[string]string{
			"landscape": "landscape_16_9",
			"square":    "square_hd",
			"portrait":  "portrait_16_9",
		},
		Defaults: map[string]any{
			"num_inference_steps":     8,
			"num_images":             1,
			"output_format":          "png",
			"enable_safety_checker":  false,
			"enable_prompt_expansion": false,
		},
		Supports: map[string]bool{
			"prompt":                  true,
			"image_size":               true,
			"num_inference_steps":      true,
			"num_images":               true,
			"seed":                     true,
			"output_format":            true,
			"enable_safety_checker":    true,
			"enable_prompt_expansion":  true,
		},
		Upscale: false,
	},
	"fal-ai/nano-banana-pro": {
		Display:   "Nano Banana Pro (Gemini 3 Pro Image)",
		Speed:     "~8s",
		Price:     "$0.15/image (1K)",
		SizeStyle: "aspect_ratio",
		Sizes: map[string]string{
			"landscape": "16:9",
			"square":    "1:1",
			"portrait":  "9:16",
		},
		Defaults: map[string]any{
			"num_images":       1,
			"output_format":    "png",
			"safety_tolerance": "5",
			"resolution":       "1K",
		},
		Supports: map[string]bool{
			"prompt":             true,
			"aspect_ratio":       true,
			"num_images":         true,
			"output_format":      true,
			"safety_tolerance":   true,
			"seed":               true,
			"sync_mode":          true,
			"resolution":         true,
			"enable_web_search":  true,
			"limit_generations":  true,
		},
		Upscale: false,
	},
	"fal-ai/gpt-image-1.5": {
		Display:   "GPT Image 1.5",
		Speed:     "~15s",
		Price:     "$0.034/image",
		SizeStyle: "gpt_literal",
		Sizes: map[string]string{
			"landscape": "1536x1024",
			"square":    "1024x1024",
			"portrait":  "1024x1536",
		},
		Defaults: map[string]any{
			"quality":       "medium",
			"num_images":    1,
			"output_format": "png",
		},
		Supports: map[string]bool{
			"prompt":        true,
			"image_size":    true,
			"quality":       true,
			"num_images":    true,
			"output_format": true,
			"background":    true,
			"sync_mode":     true,
		},
		Upscale: false,
	},
	"fal-ai/gpt-image-2": {
		Display:   "GPT Image 2",
		Speed:     "~20s",
		Price:     "$0.04–0.06/image",
		SizeStyle: "image_size_preset",
		Sizes: map[string]string{
			"landscape": "landscape_4_3",
			"square":    "square_hd",
			"portrait":  "portrait_4_3",
		},
		Defaults: map[string]any{
			"quality":       "medium",
			"num_images":    1,
			"output_format": "png",
		},
		Supports: map[string]bool{
			"prompt":        true,
			"image_size":    true,
			"quality":       true,
			"num_images":    true,
			"output_format": true,
			"sync_mode":     true,
		},
		Upscale: false,
	},
	"fal-ai/ideogram/v3": {
		Display:   "Ideogram V3",
		Speed:     "~5s",
		Price:     "$0.03-0.09/image",
		SizeStyle: "image_size_preset",
		Sizes: map[string]string{
			"landscape": "landscape_16_9",
			"square":    "square_hd",
			"portrait":  "portrait_16_9",
		},
		Defaults: map[string]any{
			"rendering_speed": "BALANCED",
			"expand_prompt":   true,
			"style":           "AUTO",
		},
		Supports: map[string]bool{
			"prompt":          true,
			"image_size":       true,
			"rendering_speed":  true,
			"expand_prompt":   true,
			"style":           true,
			"seed":            true,
		},
		Upscale: false,
	},
	"fal-ai/recraft/v4/pro/text-to-image": {
		Display:   "Recraft V4 Pro",
		Speed:     "~8s",
		Price:     "$0.25/image",
		SizeStyle: "image_size_preset",
		Sizes: map[string]string{
			"landscape": "landscape_16_9",
			"square":    "square_hd",
			"portrait":  "portrait_16_9",
		},
		Defaults: map[string]any{
			"enable_safety_checker": false,
		},
		Supports: map[string]bool{
			"prompt":                  true,
			"image_size":               true,
			"enable_safety_checker":    true,
			"colors":                   true,
			"background_color":         true,
		},
		Upscale: false,
	},
	"fal-ai/qwen-image": {
		Display:   "Qwen Image",
		Speed:     "~12s",
		Price:     "$0.02/MP",
		SizeStyle: "image_size_preset",
		Sizes: map[string]string{
			"landscape": "landscape_16_9",
			"square":    "square_hd",
			"portrait":  "portrait_16_9",
		},
		Defaults: map[string]any{
			"num_inference_steps": 30,
			"guidance_scale":      2.5,
			"num_images":           1,
			"output_format":        "png",
			"acceleration":        "regular",
		},
		Supports: map[string]bool{
			"prompt":                  true,
			"image_size":               true,
			"num_inference_steps":      true,
			"guidance_scale":           true,
			"num_images":               true,
			"output_format":            true,
			"acceleration":             true,
			"seed":                     true,
			"sync_mode":                true,
		},
		Upscale: false,
	},
}

// ImageGenConfig controls the image generation helper.
type ImageGenConfig struct {
	Disabled     bool
	Provider     string
	DefaultModel string
	DefaultSize  string
	FALAPIKey    string
	Timeout      time.Duration
	Now          func() time.Time
}

// ImageGenRequest is the public helper/tool input for image generation.
type ImageGenRequest struct {
	Prompt             string
	AspectRatio        string
	Model              string
	OutputDir          string
	NumImages          int
	Seed               *int
	NumInferenceSteps  *int
	GuidanceScale      *float64
	OutputFormat       *string
}

// ImageGenResult is the redacted helper/tool result envelope.
type ImageGenResult struct {
	Success  bool                  `json:"success"`
	ImageURL string                `json:"image_url,omitempty"`
	FilePath string                `json:"file_path,omitempty"`
	Provider string                `json:"provider,omitempty"`
	Model    string                `json:"model,omitempty"`
	Evidence ImageGenerationStatus `json:"evidence"`
	Error    string                `json:"error,omitempty"`
}

// ImageProviderRequest is the normalized provider call input.
type ImageProviderRequest struct {
	Prompt             string
	AspectRatio        string
	SizeStyle          string
	Size               string
	Model              string
	NumImages          int
	Seed               *int
	OutputFormat       string
	NumInferenceSteps  *int
	GuidanceScale      *float64
}

// ImageProviderResult is the provider-specific response.
type ImageProviderResult struct {
	ImageURL  string
	Width     int
	Height    int
	Provider  string
	MediaType string
	Upscaled  bool
}

// ImageGenerator is implemented by real or fake image generation backends.
type ImageGenerator interface {
	Available(ctx context.Context) bool
	Generate(ctx context.Context, req ImageProviderRequest) (ImageProviderResult, error)
}

// imageGenEvidence is internal evidence for runner outcomes.
type imageGenEvidence string

const (
	imageGenEvidenceOK                  imageGenEvidence = "image_gen_ok"
	imageGenEvidenceDisabled           imageGenEvidence = "image_gen_disabled"
	imageGenEvidenceInvalidArguments   imageGenEvidence = "image_gen_invalid_arguments"
	imageGenEvidenceProviderUnavailable imageGenEvidence = "image_gen_provider_unavailable"
	imageGenEvidenceAPIError           imageGenEvidence = "image_gen_api_error"
)

// ImageGenRunner validates inputs and dispatches to injected providers.
type ImageGenRunner struct {
	cfg       ImageGenConfig
	providers map[string]ImageGenerator
}

// NewImageGenRunner creates a runner with cloned providers.
func NewImageGenRunner(cfg ImageGenConfig, providers map[string]ImageGenerator) *ImageGenRunner {
	cloned := make(map[string]ImageGenerator, len(providers))
	for name, provider := range providers {
		key := normalizeImageProviderName(name)
		if key != "" && provider != nil {
			cloned[key] = provider
		}
	}
	return &ImageGenRunner{cfg: cfg, providers: cloned}
}

// Generate produces an image from the given request.
func (r *ImageGenRunner) Generate(ctx context.Context, req ImageGenRequest) ImageGenResult {
	if r == nil {
		return imageGenFailure("", imageGenEvidenceProviderUnavailable, "no image generation runner configured")
	}
	cfg := r.cfg
	if cfg.Disabled {
		return imageGenFailure("", imageGenEvidenceDisabled, "image generation is disabled")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return imageGenFailure("", imageGenEvidenceInvalidArguments, "prompt is required")
	}

	aspectRatio := normalizeAspectRatio(req.AspectRatio)
	model := req.Model
	if model == "" {
		model = cfg.DefaultModel
	}
	if model == "" {
		model = DefaultFLUXModel
	}

	providerName, provider, evidence := r.selectProvider(ctx, req)
	if evidence != "" {
		return imageGenFailure(providerName, evidence, "no image generation provider available")
	}

	meta, ok := FALModels[model]
	if !ok {
		return imageGenFailure(providerName, imageGenEvidenceAPIError, fmt.Sprintf("unknown model: %s", model))
	}

	sizeStyle := meta.SizeStyle
	size, ok := meta.Sizes[aspectRatio]
	if !ok {
		size = meta.Sizes["landscape"]
	}

	numImages := req.NumImages
	if numImages <= 0 {
		numImages = 1
	}

	outputFormat := DefaultOutputFormat
	if v, ok := meta.Defaults["output_format"].(string); ok {
		outputFormat = v
	}

	providerResult, err := provider.Generate(ctx, ImageProviderRequest{
		Prompt:            req.Prompt,
		AspectRatio:       aspectRatio,
		SizeStyle:         sizeStyle,
		Size:              size,
		Model:             model,
		NumImages:         numImages,
		Seed:              req.Seed,
		OutputFormat:      outputFormat,
		NumInferenceSteps: req.NumInferenceSteps,
		GuidanceScale:     req.GuidanceScale,
	})
	if err != nil {
		return imageGenFailure(providerName, imageGenEvidenceAPIError, redactImageGenError(err.Error()))
	}

	return ImageGenResult{
		Success:  true,
		ImageURL: providerResult.ImageURL,
		Provider: providerResult.Provider,
		Model:    model,
		Evidence: ImageGenerationStatusOK,
	}
}

func (r *ImageGenRunner) selectProvider(ctx context.Context, req ImageGenRequest) (string, ImageGenerator, imageGenEvidence) {
	explicit := normalizeImageProviderName(r.cfg.Provider)
	if explicit != "" && explicit != "auto" {
		provider := r.providers[explicit]
		if provider == nil || !provider.Available(ctx) {
			return "", nil, imageGenEvidenceProviderUnavailable
		}
		return explicit, provider, ""
	}
	if provider := r.providers["fal"]; provider != nil && provider.Available(ctx) {
		return "fal", provider, ""
	}
	for name, provider := range r.providers {
		if provider != nil && provider.Available(ctx) {
			return name, provider, ""
		}
	}
	return "", nil, imageGenEvidenceProviderUnavailable
}

func normalizeImageProviderName(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func normalizeAspectRatio(aspect string) string {
	normalized := strings.ToLower(strings.TrimSpace(aspect))
	for _, valid := range ValidAspectRatios {
		if normalized == valid {
			return valid
		}
	}
	return DefaultAspectRatio
}

func imageGenFailure(provider string, evidence imageGenEvidence, message string) ImageGenResult {
	return ImageGenResult{
		Success:  false,
		Provider: strings.TrimSpace(provider),
		Evidence: ImageGenerationStatus(strings.TrimSpace(string(evidence))),
		Error:    redactImageGenError(message),
	}
}

// imageGenSecretPatterns matches credential shapes in error messages.
var imageGenSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)\b(sk|key|token|secret)[-_]?[A-Za-z0-9]*[=:]\s*["']?[^"'\s]+`),
	regexp.MustCompile(`\b[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}\b`),
}

func redactImageGenError(text string) string {
	redacted := strings.TrimSpace(text)
	for _, pattern := range imageGenSecretPatterns {
		redacted = pattern.ReplaceAllString(redacted, "[redacted]")
	}
	if len(redacted) > 240 {
		redacted = redacted[:240] + "..."
	}
	if redacted == "" {
		return "redacted image generation error"
	}
	return redacted
}

// falAPIKeyPresent returns true if FAL_API_KEY is set in the environment.
func falAPIKeyPresent() bool {
	return os.Getenv("FAL_API_KEY") != ""
}

// FALGenImageProvider is the FAL.ai image generation provider.
type FALGenImageProvider struct {
	apiKey  string
	timeout time.Duration
}

// NewFALGenImageProvider creates a FAL provider from config.
func NewFALGenImageProvider(apiKey string) *FALGenImageProvider {
	if apiKey == "" {
		apiKey = os.Getenv("FAL_API_KEY")
	}
	return &FALGenImageProvider{
		apiKey:  apiKey,
		timeout: 120 * time.Second,
	}
}

// Available reports whether the FAL provider is configured.
func (p *FALGenImageProvider) Available(ctx context.Context) bool {
	if p == nil {
		return false
	}
	return p.apiKey != ""
}

// Generate calls the FAL.ai API for image generation.
func (p *FALGenImageProvider) Generate(ctx context.Context, req ImageProviderRequest) (ImageProviderResult, error) {
	if p == nil || p.apiKey == "" {
		return ImageProviderResult{}, errors.New("FAL provider: api key not configured")
	}

	meta, ok := FALModels[req.Model]
	if !ok {
		return ImageProviderResult{}, fmt.Errorf("FAL provider: unknown model %s", req.Model)
	}

	payload := buildFALPayload(req, meta)
	_ = payload
	return ImageProviderResult{}, errors.New("FAL provider: live API not implemented (use mock provider for testing)")
}

// buildFALPayload constructs the FAL API payload per model metadata.
func buildFALPayload(req ImageProviderRequest, meta FALModelMetadata) map[string]any {
	payload := make(map[string]any)
	for k, v := range meta.Defaults {
		payload[k] = v
	}
	payload["prompt"] = strings.TrimSpace(req.Prompt)

	switch meta.SizeStyle {
	case "image_size_preset", "gpt_literal":
		payload["image_size"] = req.Size
	case "aspect_ratio":
		payload["aspect_ratio"] = req.Size
	}

	if req.Seed != nil {
		payload["seed"] = *req.Seed
	}

	filtered := make(map[string]any)
	for k, v := range payload {
		if meta.Supports[k] {
			filtered[k] = v
		}
	}
	return filtered
}

// ImageGenTool exposes the helper through the standard Go tool contract.
type ImageGenTool struct {
	runner *ImageGenRunner
}

// NewImageGenTool creates a new image generation tool.
func NewImageGenTool(runner *ImageGenRunner) *ImageGenTool {
	return &ImageGenTool{runner: runner}
}

// Name returns the tool name.
func (*ImageGenTool) Name() string { return "image_generate" }

// Description returns the tool description.
func (*ImageGenTool) Description() string {
	return "Generate high-quality images from text prompts using FAL.ai models. Returns image URL."
}

// Schema returns the JSON schema for the tool.
func (*ImageGenTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"prompt":{
				"type":"string",
				"description":"The text prompt describing the desired image. Be detailed and descriptive."
			},
			"aspect_ratio":{
				"type":"string",
				"enum":["landscape","square","portrait"],
				"description":"The aspect ratio of the generated image. 'landscape' is 16:9, 'portrait' is 9:16, 'square' is 1:1.",
				"default":"landscape"
			}
		},
		"required":["prompt"]
	}`)
}

// Timeout returns the tool timeout.
func (*ImageGenTool) Timeout() time.Duration { return 120 * time.Second }

// Execute runs the image generation tool.
func (t *ImageGenTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Prompt      string `json:"prompt"`
		AspectRatio string `json:"aspect_ratio"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		result := imageGenFailure("", imageGenEvidenceInvalidArguments, "invalid image generation args: "+err.Error())
		return json.Marshal(result)
	}

	outputDir := os.TempDir()
	result := t.runner.Generate(ctx, ImageGenRequest{
		Prompt:      in.Prompt,
		AspectRatio: in.AspectRatio,
		OutputDir:   outputDir,
	})
	return json.Marshal(result)
}

// Ensure compile-time tool conformance.
var _ Tool = (*ImageGenTool)(nil)
