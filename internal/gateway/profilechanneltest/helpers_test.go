package profilechanneltest

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestChannelCredentialBuildsEnvBackedChannelSecretRef(t *testing.T) {
	got := ChannelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN")
	if got.Kind != "channel" || got.Channel != "whatsapp" || got.OwnerProfile != "main" {
		t.Fatalf("credential identity = %+v", got)
	}
	if got.SecretRef == nil || got.SecretRef.Source != config.SecretRefSourceEnv || got.SecretRef.ID != "GORMES_MAIN_WHATSAPP_TOKEN" {
		t.Fatalf("credential secret ref = %+v", got.SecretRef)
	}
}

func TestTokenCredentialHash(t *testing.T) {
	if got, want := TokenCredentialHash("secret"), "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b"; got != want {
		t.Fatalf("TokenCredentialHash = %q, want %q", got, want)
	}
}
