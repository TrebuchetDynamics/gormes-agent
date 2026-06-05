//go:build !slim

package tools

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/imagegen"
)

// Provider facades for image generation live in image_generation.go; provider
// implementation lives in internal/tools/imagegen/provider.go.

// ManagedGatewayImageGenProviderOptions configures an image-generation provider
// backed by a managed MCP gateway.
type ManagedGatewayImageGenProviderOptions struct {
	Name     string
	ToolName string
	Bridge   *ManagedGatewayBridge
}

// NewManagedGatewayImageGenProvider wraps a ManagedGatewayBridge as an
// ImageGenProvider while keeping the imagegen package independent of the
// top-level tools package.
func NewManagedGatewayImageGenProvider(opts ManagedGatewayImageGenProviderOptions) imagegen.ImageGenProvider {
	return imagegen.NewManagedGatewayImageGenProvider(imagegen.ManagedGatewayImageGenProviderOptions{
		Name:     opts.Name,
		ToolName: opts.ToolName,
		Bridge:   managedGatewayImageGenBridgeAdapter{bridge: opts.Bridge},
	})
}

type managedGatewayImageGenBridgeAdapter struct {
	bridge *ManagedGatewayBridge
}

func (a managedGatewayImageGenBridgeAdapter) Discover(ctx context.Context) (imagegen.ManagedGatewayDiscovery, error) {
	if a.bridge == nil {
		return imagegen.ManagedGatewayDiscovery{Evidence: imagegen.ManagedGatewayEvidenceUnavailable}, nil
	}
	disc, err := a.bridge.Discover(ctx)
	out := imagegen.ManagedGatewayDiscovery{Evidence: imagegen.ManagedGatewayEvidence(disc.Evidence)}
	for _, tool := range disc.Tools {
		out.Tools = append(out.Tools, imagegen.ManagedGatewayDiscoveryTool{Name: tool.Name})
	}
	return out, err
}

func (a managedGatewayImageGenBridgeAdapter) CallTool(ctx context.Context, name string, arguments map[string]any) (imagegen.ManagedGatewayCallResult, imagegen.ManagedGatewayEvidence, error) {
	if a.bridge == nil {
		return imagegen.ManagedGatewayCallResult{}, imagegen.ManagedGatewayEvidenceUnavailable, imagegen.ErrImageGenProviderUnavailable
	}
	res, evidence, err := a.bridge.CallTool(ctx, name, arguments)
	out := imagegen.ManagedGatewayCallResult{
		StructuredContent: res.StructuredContent,
		IsError:           res.IsError,
	}
	for _, part := range res.Content {
		out.Content = append(out.Content, imagegen.ManagedGatewayContent{
			Kind:     part.Kind,
			Text:     part.Text,
			MimeType: part.MimeType,
			URI:      part.URI,
		})
	}
	return out, imagegen.ManagedGatewayEvidence(evidence), err
}
