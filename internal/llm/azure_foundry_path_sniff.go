package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry"

// ClassifyAzurePath inspects rawURL and returns the Azure transport suggested by its path.
func ClassifyAzurePath(rawURL string) AzureTransport {
	return azurefoundry.ClassifyAzurePath(rawURL)
}
