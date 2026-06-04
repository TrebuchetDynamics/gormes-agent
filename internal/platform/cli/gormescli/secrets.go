package gormescli

import (
	"github.com/spf13/cobra"

	secretsapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/secrets"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type SecretsOptions = secretsapp.Options
type SecretsBuildProvenance = secretsapp.BuildProvenance

func NewSecretsCommand(options SecretsOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "secrets",
		Short:        "Apply, audit, configure, and reload SecretRef-backed runtime secrets",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newSecretsApplyCommand(options))
	cmd.AddCommand(newSecretsAuditCommand(options))
	cmd.AddCommand(newSecretsConfigureCommand(options))
	cmd.AddCommand(newSecretsReloadCommand(options))
	return cmd
}

func newSecretsApplyCommand(options SecretsOptions) *cobra.Command {
	var planPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "apply --plan <file>",
		Short: "Resolve a generated SecretRef plan into the runtime snapshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return secretsapp.Apply(cmd.Context(), cmd.OutOrStdout(), planPath, jsonOut, options)
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "JSON plan containing SecretRef targets")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("plan")
	return cmd
}

func newSecretsAuditCommand(options SecretsOptions) *cobra.Command {
	var planPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "audit --plan <file>",
		Short: "Audit plaintext secrets, unresolved refs, and snapshot precedence drift",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return secretsapp.Audit(cmd.Context(), cmd.OutOrStdout(), planPath, jsonOut, options)
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "JSON plan containing SecretRef targets")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	return cmd
}

func newSecretsConfigureCommand(options SecretsOptions) *cobra.Command {
	var source string
	var provider string
	var id string
	var optional bool
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "configure <path>",
		Short: "Build and preflight a typed SecretRef mapping for one config path",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return secretsapp.Configure(cmd.Context(), cmd.OutOrStdout(), args[0], source, provider, id, optional, jsonOut, options)
		},
	}
	cmd.Flags().StringVar(&source, "source", "env", "SecretRef source: env or file")
	cmd.Flags().StringVar(&provider, "provider", config.DefaultSecretProviderAlias, "SecretRef provider alias")
	cmd.Flags().StringVar(&id, "id", "", "SecretRef id, such as an environment variable name")
	cmd.Flags().BoolVar(&optional, "optional", false, "allow preflight failure for optional refs")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newSecretsReloadCommand(options SecretsOptions) *cobra.Command {
	var planPath string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "reload --plan <file>",
		Short: "Atomically re-resolve SecretRefs and keep the last-good snapshot on failure",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return secretsapp.Reload(cmd.Context(), cmd.OutOrStdout(), planPath, jsonOut, options)
		},
	}
	cmd.Flags().StringVar(&planPath, "plan", "", "JSON plan containing SecretRef targets")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "print machine-readable JSON")
	_ = cmd.MarkFlagRequired("plan")
	return cmd
}
