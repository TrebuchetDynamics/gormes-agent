package dashboard

import (
	"fmt"

	"github.com/spf13/cobra"
)

type DashboardOptions struct {
	Port   int
	NoOpen bool
}

type DashboardCommandSeams struct {
	Run func(*cobra.Command, DashboardOptions) error
}

func NewDashboardCommandWithSeams(seams DashboardCommandSeams) *cobra.Command {
	if seams.Run == nil {
		seams.Run = func(*cobra.Command, DashboardOptions) error {
			return fmt.Errorf("dashboard run seam is not configured")
		}
	}
	opts := DashboardOptions{}
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Start the Gormes web dashboard",
		Long:  "Starts an HTTP server with an htmx-based web dashboard for managing sessions, config, skills, and logs.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return seams.Run(cmd, opts)
		},
	}
	cmd.Flags().IntVar(&opts.Port, "port", 43827, "Dashboard HTTP server port")
	cmd.Flags().BoolVar(&opts.NoOpen, "no-open", false, "do not open the dashboard in a browser")
	return cmd
}
