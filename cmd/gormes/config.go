package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// newConfigCommand builds the `gormes config` subtree. It exposes a small
// Hermes-aliased command surface — show, path, env-path, set — over the
// native Gormes XDG TOML/dotenv files. Writes route through
// internal/config helpers so secrets never land in config.toml.
func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "config",
		Short:        "Inspect or update the Gormes configuration files",
		SilenceUsage: true,
	}
	cmd.AddCommand(newConfigPathCommand())
	cmd.AddCommand(newConfigEnvPathCommand())
	cmd.AddCommand(newConfigShowCommand())
	cmd.AddCommand(newConfigSetCommand())
	return cmd
}

func newConfigPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the Gormes TOML config path",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.ConfigPath())
			return nil
		},
	}
}

func newConfigEnvPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "env-path",
		Short: "Print the Gormes dotenv (.env) secrets path",
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.EnvPath())
			return nil
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value (TOML for non-secret keys, .env for *_API_KEY/*_TOKEN/api_key)",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errors.New("usage: gormes config set <key> <value>")
			}
			if len(args) > 2 {
				return errors.New("gormes config set takes exactly two arguments: <key> <value>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := strings.TrimSpace(args[0])
			value := args[1]
			if key == "" {
				return errors.New("gormes config set: empty key")
			}
			if config.IsSecretKey(key) {
				envName := config.SecretEnvName(key)
				if err := config.WriteEnvValue(config.EnvPath(), envName, value); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "set %s in %s\n", envName, config.EnvPath())
				return nil
			}
			if err := config.WriteTOMLValue(config.ConfigPath(), key, value); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set %s in %s\n", key, config.ConfigPath())
			return nil
		},
	}
	return cmd
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show resolved Gormes configuration with secrets redacted",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Paths")
			fmt.Fprintf(out, "  config: %s\n", config.ConfigPath())
			fmt.Fprintf(out, "  env:    %s\n", config.EnvPath())
			fmt.Fprintln(out, "Hermes")
			fmt.Fprintf(out, "  endpoint: %s\n", cfg.Hermes.Endpoint)
			fmt.Fprintf(out, "  model:    %s\n", cfg.Hermes.Model)
			fmt.Fprintf(out, "  provider: %s\n", cfg.Hermes.Provider)
			fmt.Fprintln(out, "Secrets")
			fmt.Fprintf(out, "  api_key: %s\n", redactedSecretStatus(cfg.Hermes.APIKey))
			fmt.Fprintf(out, "  GORMES_API_KEY (env): %s\n", redactedSecretStatus(os.Getenv("GORMES_API_KEY")))
			return nil
		},
	}
}

func redactedSecretStatus(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(not set)"
	}
	return "set [REDACTED]"
}
