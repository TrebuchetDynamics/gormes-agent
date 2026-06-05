//go:build !slim

package imagegen

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrImageGenProviderUnavailable = errors.New("image_generation: provider unavailable")

type ManagedGatewayEvidence string

const (
	ManagedGatewayEvidenceOK             ManagedGatewayEvidence = "ok"
	ManagedGatewayEvidenceUnavailable    ManagedGatewayEvidence = "gateway_unavailable"
	ManagedGatewayEvidenceAuthRequired   ManagedGatewayEvidence = "auth_required"
	ManagedGatewayEvidenceSchemaRejected ManagedGatewayEvidence = "schema_rejected"
	ManagedGatewayEvidenceToolCallFailed ManagedGatewayEvidence = "tool_call_failed"
)

type ManagedGatewayContent struct {
	Kind     string
	Text     string
	MimeType string
	URI      string
}

type ManagedGatewayCallResult struct {
	Content           []ManagedGatewayContent
	StructuredContent json.RawMessage
	IsError           bool
}

type ManagedGatewayDiscoveryTool struct {
	Name string
}

type ManagedGatewayDiscovery struct {
	Tools    []ManagedGatewayDiscoveryTool
	Evidence ManagedGatewayEvidence
}

type ManagedGatewayBridge interface {
	Discover(context.Context) (ManagedGatewayDiscovery, error)
	CallTool(context.Context, string, map[string]any) (ManagedGatewayCallResult, ManagedGatewayEvidence, error)
}

type ManagedGatewayImageGenProviderOptions struct {
	Name     string
	ToolName string
	Bridge   ManagedGatewayBridge
}

type managedGatewayImageGenProvider struct {
	name     string
	toolName string
	bridge   ManagedGatewayBridge
}

func NewManagedGatewayImageGenProvider(opts ManagedGatewayImageGenProviderOptions) ImageGenProvider {
	name := normalizeImageProviderName(opts.Name)
	if name == "" {
		name = "managed-fal"
	}
	toolName := strings.TrimSpace(opts.ToolName)
	if toolName == "" {
		toolName = "image_generate"
	}
	return &managedGatewayImageGenProvider{name: name, toolName: toolName, bridge: opts.Bridge}
}

func (p *managedGatewayImageGenProvider) Name() string { return p.name }

func (p *managedGatewayImageGenProvider) Available(ctx context.Context) bool {
	_, err := p.resolveTool(ctx)
	return err == nil
}

func (p *managedGatewayImageGenProvider) Generate(ctx context.Context, req ImageProviderRequest) (ImageProviderResult, error) {
	toolName, err := p.resolveTool(ctx)
	if err != nil {
		return ImageProviderResult{}, err
	}
	args := managedGatewayImageArgs(req)
	res, evidence, err := p.bridge.CallTool(ctx, toolName, args)
	if err != nil {
		return ImageProviderResult{}, managedGatewayProviderError(evidence, err)
	}
	if res.IsError {
		return ImageProviderResult{}, managedGatewayProviderError(evidence, errors.New(renderManagedGatewayCallResult(res)))
	}
	parsed, err := parseManagedGatewayImageResult(res)
	if err != nil {
		return ImageProviderResult{}, managedGatewayProviderError(evidence, err)
	}
	parsed.Provider = p.name
	if parsed.Model == "" {
		parsed.Model = req.Model
	}
	return parsed, nil
}

func (p *managedGatewayImageGenProvider) resolveTool(ctx context.Context) (string, error) {
	if p == nil || p.bridge == nil {
		return "", ErrImageGenProviderUnavailable
	}
	disc, err := p.bridge.Discover(ctx)
	if err != nil {
		return "", managedGatewayProviderError(disc.Evidence, err)
	}
	if disc.Evidence != ManagedGatewayEvidenceOK && len(disc.Tools) == 0 {
		return "", managedGatewayProviderError(disc.Evidence, ErrImageGenProviderUnavailable)
	}
	for _, tool := range disc.Tools {
		if tool.Name == p.toolName {
			return tool.Name, nil
		}
	}
	return "", managedGatewayProviderError(disc.Evidence, fmt.Errorf("managed gateway image tool %q not advertised", p.toolName))
}

func managedGatewayImageArgs(req ImageProviderRequest) map[string]any {
	args := map[string]any{
		"prompt":        req.Prompt,
		"model":         req.Model,
		"aspect_ratio":  req.AspectRatio,
		"image_size":    req.Size,
		"size_style":    req.SizeStyle,
		"num_images":    req.NumImages,
		"output_format": req.OutputFormat,
	}
	if req.Seed != nil {
		args["seed"] = *req.Seed
	}
	if req.NumInferenceSteps != nil {
		args["num_inference_steps"] = *req.NumInferenceSteps
	}
	if req.GuidanceScale != nil {
		args["guidance_scale"] = *req.GuidanceScale
	}
	return args
}

func managedGatewayProviderError(evidence ManagedGatewayEvidence, err error) error {
	if err == nil {
		return nil
	}
	if evidence == "" {
		evidence = ManagedGatewayEvidenceToolCallFailed
	}
	return fmt.Errorf("managed image gateway %s: %w", evidence, err)
}

func parseManagedGatewayImageResult(res ManagedGatewayCallResult) (ImageProviderResult, error) {
	if len(res.StructuredContent) > 0 {
		if parsed, ok, err := parseManagedGatewayImageJSON(res.StructuredContent); err != nil || ok {
			return parsed, err
		}
	}
	for _, part := range res.Content {
		if part.Kind == "image" && part.URI != "" {
			return ImageProviderResult{ImageURL: part.URI, MediaType: nonEmpty(part.MimeType, "image/png")}, nil
		}
		if part.Kind == "resource" && part.URI != "" {
			return ImageProviderResult{ImageURL: part.URI, MediaType: part.MimeType}, nil
		}
		if strings.TrimSpace(part.Text) != "" {
			if parsed, ok, err := parseManagedGatewayImageJSON([]byte(part.Text)); err != nil || ok {
				return parsed, err
			}
		}
	}
	return ImageProviderResult{}, errors.New("managed image gateway returned no image payload")
}

func parseManagedGatewayImageJSON(raw []byte) (ImageProviderResult, bool, error) {
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ImageProviderResult{}, false, nil
	}
	mediaType := stringField(obj, "media_type", "mime_type", "mimeType")
	if mediaType == "" {
		mediaType = "image/png"
	}
	model := stringField(obj, "model")
	if b64 := stringField(obj, "image_base64", "b64_json", "base64", "bytes_base64"); b64 != "" {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return ImageProviderResult{}, true, fmt.Errorf("managed image gateway invalid base64 image: %w", err)
		}
		return ImageProviderResult{ImageBytes: data, MediaType: mediaType, Model: model}, true, nil
	}
	if url := stringField(obj, "image_url", "url", "uri"); url != "" {
		return ImageProviderResult{ImageURL: url, MediaType: mediaType, Model: model}, true, nil
	}
	if images, ok := obj["images"].([]any); ok && len(images) > 0 {
		if first, ok := images[0].(map[string]any); ok {
			if url := stringField(first, "url", "image_url", "uri"); url != "" {
				return ImageProviderResult{ImageURL: url, MediaType: nonEmpty(stringField(first, "media_type", "mime_type", "mimeType"), mediaType), Model: model}, true, nil
			}
			if b64 := stringField(first, "image_base64", "b64_json", "base64"); b64 != "" {
				data, err := base64.StdEncoding.DecodeString(b64)
				if err != nil {
					return ImageProviderResult{}, true, fmt.Errorf("managed image gateway invalid base64 image: %w", err)
				}
				return ImageProviderResult{ImageBytes: data, MediaType: nonEmpty(stringField(first, "media_type", "mime_type", "mimeType"), mediaType), Model: model}, true, nil
			}
		}
	}
	return ImageProviderResult{}, false, nil
}

func renderManagedGatewayCallResult(res ManagedGatewayCallResult) string {
	for _, part := range res.Content {
		if strings.TrimSpace(part.Text) != "" {
			return strings.TrimSpace(part.Text)
		}
	}
	return "managed image gateway tool call failed"
}

func stringField(obj map[string]any, names ...string) string {
	for _, name := range names {
		if v, ok := obj[name].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
