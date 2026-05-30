package gateway

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestChannelSetupFacadeConvertsPairingStatus(t *testing.T) {
	plan := BuildChannelSetupPlanWithOptions(config.Config{
		Profiles: map[string]config.ProfileCfg{
			"main": {
				Enabled: true,
				Channels: map[string]config.ProfileChannelCfg{
					"whatsapp": {Enabled: true, Credential: "main-whatsapp", AllowedUsers: []string{"6586915095"}},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"main-whatsapp": channelCredential("whatsapp", "main", "GORMES_MAIN_WHATSAPP_TOKEN"),
		},
	}, ChannelSetupPlanOptions{
		Pairing: PairingStatus{Platforms: []PairingPlatformStatus{{Platform: "whatsapp", State: PairingPlatformStatePaired}}},
	})
	var whatsapp ChannelSetupEntry
	for _, entry := range plan.Channels {
		if entry.ID == "whatsapp" {
			whatsapp = entry
			break
		}
	}
	if whatsapp.Status != ChannelSetupStatusPaired {
		t.Fatalf("whatsapp status = %q, want paired through root facade", whatsapp.Status)
	}
}
