package live

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/commandtest"
	"github.com/spf13/cobra"
)

func TestGatewayCommandUsesInjectedRunAndSubcommands(t *testing.T) {
	var ran bool
	cmd := NewGatewayCommandWithSeams(GatewayCommandSeams{
		Run: func(cmd *cobra.Command, _ []string) error {
			ran = true
			_, _ = cmd.OutOrStdout().Write([]byte("gateway-run\n"))
			return nil
		},
		StopCommand:      stubGatewayChild("stop"),
		RestartCommand:   stubGatewayChild("restart"),
		ReloadCommand:    stubGatewayChild("reload"),
		StatusCommand:    stubGatewayChild("status"),
		DiscoverCommand:  stubGatewayChild("discover"),
		ProbeCommand:     stubGatewayChild("probe"),
		UsageCostCommand: stubGatewayChild("usage-cost"),
	}, Options{})

	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	stdout, _, err := commandtest.Execute(t, cmd)
	if err != nil {
		t.Fatalf("gateway: %v", err)
	}
	if !ran || !strings.Contains(stdout, "gateway-run") {
		t.Fatalf("gateway run seam not called; ran=%v stdout=%q", ran, stdout)
	}
	for _, want := range []string{"stop", "restart", "reload", "status", "fleet", "discover", "probe", "usage-cost", "start", "install", "uninstall", "run", "setup", "migrate-legacy", "list"} {
		if _, _, err := cmd.Find([]string{want}); err != nil {
			t.Fatalf("gateway command missing child %q: %v", want, err)
		}
	}
}

func stubGatewayChild(name string) func() *cobra.Command {
	return func() *cobra.Command {
		return &cobra.Command{Use: name}
	}
}
