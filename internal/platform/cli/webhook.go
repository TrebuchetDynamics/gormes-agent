package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/webhooks"

var (
	ErrWebhookURLEmpty             = webhooks.ErrWebhookURLEmpty
	ErrWebhookURLBadScheme         = webhooks.ErrWebhookURLBadScheme
	ErrWebhookURLUserInfoForbidden = webhooks.ErrWebhookURLUserInfoForbidden
	ErrWebhookURLParseFailed       = webhooks.ErrWebhookURLParseFailed
)

func NormalizeWebhookURL(raw string) (string, error) { return webhooks.NormalizeWebhookURL(raw) }
