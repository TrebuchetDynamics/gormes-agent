package gormescli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewDeprecatedLoginCommand preserves Hermes' hidden top-level login shim for old scripts.
func NewDeprecatedLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "login",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.ErrOrStderr(), "gormes login is deprecated; use `gormes auth add <provider>`, `gormes model`, or `gormes setup`.")
			return err
		},
	}
	cmd.Flags().String("provider", "", "provider to authenticate")
	cmd.Flags().Bool("no-browser", false, "do not open a browser")
	cmd.Flags().String("timeout", "", "login timeout in seconds")
	cmd.Flags().String("ca-bundle", "", "custom CA bundle path")
	cmd.Flags().Bool("insecure", false, "skip TLS verification")
	cmd.Flags().String("portal-url", "", "OAuth portal URL")
	cmd.Flags().String("inference-url", "", "inference URL")
	cmd.Flags().String("client-id", "", "OAuth client ID")
	cmd.Flags().String("scope", "", "OAuth scope")
	return cmd
}
