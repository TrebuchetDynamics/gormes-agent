package providers

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// AuthAddOptions is the parsed flag payload for `gormes auth add`.
type AuthAddOptions struct {
	Provider                    string
	AuthType                    string
	Label                       string
	APIKey                      string
	InferenceURL                string
	PortalURL                   string
	ClientID                    string
	Scope                       string
	NoBrowser                   bool
	Timeout                     string
	Insecure                    bool
	CABundle                    string
	EmergencyImportFromCodexCLI string
}

// AuthSeams routes the provider command tree to the auth runtime while that
// runtime remains in cmd/gormes for this slice.
type AuthSeams struct {
	RunBare                    func(*cobra.Command) error
	RunAdd                     func(*cobra.Command, AuthAddOptions) error
	RunList                    func(*cobra.Command, string, bool) error
	RunRemove                  func(*cobra.Command, string, string) error
	RunReset                   func(*cobra.Command, string) error
	RunStatus                  func(*cobra.Command, string, bool) error
	RunLogout                  func(*cobra.Command, string) error
	EmitJSONSubcommandRequired func(*cobra.Command) error
	EmitJSONInputError         func(*cobra.Command, string, string) error
}

func (s AuthSeams) withDefaults() AuthSeams {
	if s.RunBare == nil {
		s.RunBare = func(*cobra.Command) error { return errors.New("gormes auth: bare auth seam is not configured") }
	}
	if s.RunAdd == nil {
		s.RunAdd = func(*cobra.Command, AuthAddOptions) error {
			return errors.New("gormes auth add: seam is not configured")
		}
	}
	if s.RunList == nil {
		s.RunList = func(*cobra.Command, string, bool) error {
			return errors.New("gormes auth list: seam is not configured")
		}
	}
	if s.RunRemove == nil {
		s.RunRemove = func(*cobra.Command, string, string) error {
			return errors.New("gormes auth remove: seam is not configured")
		}
	}
	if s.RunReset == nil {
		s.RunReset = func(*cobra.Command, string) error { return errors.New("gormes auth reset: seam is not configured") }
	}
	if s.RunStatus == nil {
		s.RunStatus = func(*cobra.Command, string, bool) error {
			return errors.New("gormes auth status: seam is not configured")
		}
	}
	if s.RunLogout == nil {
		s.RunLogout = func(*cobra.Command, string) error { return errors.New("gormes auth logout: seam is not configured") }
	}
	if s.EmitJSONSubcommandRequired == nil {
		s.EmitJSONSubcommandRequired = func(*cobra.Command) error {
			return errors.New("gormes auth: json subcommand-required seam is not configured")
		}
	}
	if s.EmitJSONInputError == nil {
		s.EmitJSONInputError = func(_ *cobra.Command, action, message string) error {
			return fmt.Errorf("%s: %s", action, message)
		}
	}
	return s
}

func NewAuthCommandWithSeams(seams AuthSeams, opts Options) *cobra.Command {
	seams = seams.withDefaults()
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "auth",
		Short:        "Manage Hermes-compatible provider credentials",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if asJSON {
				return seams.EmitJSONSubcommandRequired(cmd)
			}
			return seams.RunBare(cmd)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a machine-readable {build, action: 'subcommand_required', parent, available, error} document on stdout (the bare credential pool listing remains the default text output)")
	cmd.AddCommand(newAuthAddCommand(seams))
	cmd.AddCommand(newAuthListCommand(seams))
	cmd.AddCommand(newAuthRemoveCommand(seams))
	cmd.AddCommand(newAuthResetCommand(seams))
	cmd.AddCommand(newAuthStatusCommand(seams))
	cmd.AddCommand(newAuthLogoutCommand(seams))
	cmd.AddCommand(NewAuthSpotifyCommand(opts))
	return cmd
}

func NewAuthSpotifyCommand(opts Options) *cobra.Command {
	return gormescli.NewRowBackedCommand(gormescli.RowBackedCommandSpec{
		Use:   "spotify",
		Short: "Manage Spotify service-provider OAuth",
		Row:   "Hermes auth Spotify service-provider subcommand",
	}, gormescli.RowBackedCommandOptions{
		BuildProvenance: opts.BuildProvenance,
	})
}

func newAuthAddCommand(seams AuthSeams) *cobra.Command {
	var authType string
	var label string
	var apiKey string
	var inferenceURL string
	var portalURL string
	var clientID string
	var scope string
	var noBrowser bool
	var timeout string
	var insecure bool
	var caBundle string
	var emergencyImportFromCodexCLI string

	cmd := &cobra.Command{
		Use:   "add <provider>",
		Short: "Add a provider credential to the Hermes-compatible credential pool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return seams.RunAdd(cmd, AuthAddOptions{
				Provider:                    args[0],
				AuthType:                    authType,
				Label:                       label,
				APIKey:                      apiKey,
				InferenceURL:                inferenceURL,
				PortalURL:                   portalURL,
				ClientID:                    clientID,
				Scope:                       scope,
				NoBrowser:                   noBrowser,
				Timeout:                     timeout,
				Insecure:                    insecure,
				CABundle:                    caBundle,
				EmergencyImportFromCodexCLI: emergencyImportFromCodexCLI,
			})
		},
	}
	cmd.Flags().StringVar(&authType, "type", "", "credential type: api-key, api_key, or oauth")
	cmd.Flags().StringVar(&label, "label", "", "credential label")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key to store; omitted values are not echoed")
	cmd.Flags().StringVar(&inferenceURL, "inference-url", "", "provider inference base URL override")
	cmd.Flags().StringVar(&portalURL, "portal-url", "", "OAuth portal URL")
	cmd.Flags().StringVar(&clientID, "client-id", "", "OAuth client ID")
	cmd.Flags().StringVar(&scope, "scope", "", "OAuth scope")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "do not open a browser for OAuth")
	cmd.Flags().StringVar(&timeout, "timeout", "", "OAuth timeout")
	cmd.Flags().BoolVar(&insecure, "insecure", false, "disable OAuth TLS verification")
	cmd.Flags().StringVar(&caBundle, "ca-bundle", "", "OAuth CA bundle")
	cmd.Flags().StringVar(&emergencyImportFromCodexCLI, "emergency-import-from-codex-cli", "", "explicitly import Codex CLI auth.json after accepting the refresh-token race envelope")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action: 'added', provider, id, label, redacted}` (api-key path only)")
	return cmd
}

func newAuthListCommand(seams AuthSeams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "list [provider]",
		Short:        "List provider credentials with secrets redacted",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := ""
			if len(args) > 0 {
				provider = args[0]
			}
			return seams.RunList(cmd, provider, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, provider, credentials: [...]}` JSON document with the same redacted fields the human row prints (suitable for fleet credential-health monitoring)")
	return cmd
}

func newAuthRemoveCommand(seams AuthSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "remove <provider> <target>",
		Short:        "Remove a provider credential by index, id, or label",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return seams.RunRemove(cmd, args[0], args[1])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action, provider, removed: {id, label}, redacted}`")
	return cmd
}

func newAuthResetCommand(seams AuthSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "reset <provider>",
		Short:        "Reset provider credential cooldown/exhaustion state",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return seams.RunReset(cmd, args[0])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action, provider, count, redacted}`")
	return cmd
}

func newAuthStatusCommand(seams AuthSeams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "status <provider>",
		Short:        "Show redacted provider auth status",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				const msg = "auth status: missing required <provider> argument"
				if asJSON {
					return seams.EmitJSONInputError(cmd, "missing_argument", msg)
				}
				return fmt.Errorf("%s", msg)
			}
			return seams.RunStatus(cmd, args[0], asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a machine-readable JSON document with the same redacted fields (suitable for credential-health monitoring)")
	return cmd
}

func newAuthLogoutCommand(seams AuthSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "logout <provider>",
		Short:        "Clear provider credentials",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return seams.RunLogout(cmd, args[0])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action: 'logged_out'|'absent', provider, redacted}`")
	return cmd
}
