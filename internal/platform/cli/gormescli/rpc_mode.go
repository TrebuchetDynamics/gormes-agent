package gormescli

import "github.com/spf13/cobra"

// InstallRootRPCModeFlags installs root-level flags used by stdio RPC embedders.
func InstallRootRPCModeFlags(root *cobra.Command) {
	if root == nil {
		return
	}
	if root.PersistentFlags().Lookup("mode") == nil {
		root.PersistentFlags().String("mode", "", "run mode for embedders; supported: rpc")
	}
	if root.PersistentFlags().Lookup("no-session") == nil {
		root.PersistentFlags().Bool("no-session", false, "disable session persistence for RPC mode")
	}
	if root.PersistentFlags().Lookup("prompt-template") == nil {
		root.PersistentFlags().StringArray("prompt-template", nil, "load a Markdown prompt template file or directory for the native TUI (repeatable)")
	}
	if root.PersistentFlags().Lookup("no-prompt-templates") == nil {
		root.PersistentFlags().Bool("no-prompt-templates", false, "disable native TUI prompt template discovery")
	}
}
