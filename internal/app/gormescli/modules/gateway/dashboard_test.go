package gateway

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestDashboardCommandUsesInjectedRunOptions(t *testing.T) {
	var gotPort int
	var gotNoOpen bool
	cmd := NewDashboardCommandWithSeams(DashboardCommandSeams{
		Run: func(_ *cobra.Command, opts DashboardOptions) error {
			gotPort = opts.Port
			gotNoOpen = opts.NoOpen
			return nil
		},
	})
	cmd.SetArgs([]string{"--port", "4567", "--no-open"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if gotPort != 4567 || !gotNoOpen {
		t.Fatalf("dashboard options = port %d noOpen %v, want 4567 true", gotPort, gotNoOpen)
	}
}
