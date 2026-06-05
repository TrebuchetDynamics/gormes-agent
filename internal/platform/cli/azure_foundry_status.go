package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/providerstatus"

// AzureFoundryManualOptions describes the manual fallback choices the
// CLI status should keep visible when probe-driven auto-detection cannot
// classify an endpoint.
type AzureFoundryManualOptions = providerstatus.AzureFoundryManualOptions

// AzureFoundryStatusInput is the deterministic input for rendering the
// CLI status surface.
type AzureFoundryStatusInput = providerstatus.AzureFoundryStatusInput

// AzureFoundryStatus is the rendered output.
type AzureFoundryStatus = providerstatus.AzureFoundryStatus

// RenderAzureFoundryStatus turns a runtime + probe read model into a
// deterministic CLI status surface.
func RenderAzureFoundryStatus(in AzureFoundryStatusInput) AzureFoundryStatus {
	return providerstatus.RenderAzureFoundryStatus(in)
}
