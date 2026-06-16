package whatsapp

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestWhatsAppCommandUsesInjectedOptions(t *testing.T) {
	var got WhatsAppOptions
	cmd := NewWhatsAppCommandWithSeams(WhatsAppCommandSeams{
		Run: func(_ *cobra.Command, opts WhatsAppOptions) error {
			got = opts
			return nil
		},
	})
	cmd.SetArgs([]string{
		"--mode", "self-chat",
		"--allowed-users", "528112345678",
		"--allow-all-users",
		"--debug",
		"--plan",
		"--json",
		"--bridge-script", "/tmp/bridge.js",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("whatsapp: %v", err)
	}
	if got.Mode != "self-chat" || got.AllowedUsers != "528112345678" || got.BridgeScript != "/tmp/bridge.js" {
		t.Fatalf("string options = %+v, want mode/users/bridge-script", got)
	}
	if !got.AllowAll || !got.Debug || !got.PlanOnly || !got.JSONOut {
		t.Fatalf("bool options = %+v, want all enabled", got)
	}
}
