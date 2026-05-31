package providers

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/provideridentity"
)

const TopLevelLogoutAllowedProviders = "nous|openai-codex|spotify"

// LogoutSeams isolates top-level logout from the larger auth command
// implementation while auth itself remains in cmd/gormes.
type LogoutSeams struct {
	NormalizeAuthProvider   func(string) string
	ConfiguredProvider      func() (string, error)
	RunAuthLogout           func(*cobra.Command, string) error
	ResetProviderIfMatching func(string) error
}

func DefaultLogoutSeams() LogoutSeams {
	return LogoutSeams{
		NormalizeAuthProvider: normalizeLogoutProvider,
		ConfiguredProvider: func() (string, error) {
			cfg, err := config.Load(nil)
			if err != nil {
				return "", err
			}
			return normalizeLogoutProvider(cfg.Hermes.Provider), nil
		},
		RunAuthLogout: func(cmd *cobra.Command, provider string) error {
			return fmt.Errorf("gormes logout: auth logout seam is not configured for provider=%s", provider)
		},
		ResetProviderIfMatching: func(provider string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			if normalizeLogoutProvider(cfg.Hermes.Provider) != provider {
				return nil
			}
			return config.WriteTOMLValue(config.ConfigPath(), "hermes.provider", "auto")
		},
	}
}

func (s LogoutSeams) withDefaults() LogoutSeams {
	defaults := DefaultLogoutSeams()
	if s.NormalizeAuthProvider == nil {
		s.NormalizeAuthProvider = defaults.NormalizeAuthProvider
	}
	if s.ConfiguredProvider == nil {
		s.ConfiguredProvider = defaults.ConfiguredProvider
	}
	if s.RunAuthLogout == nil {
		s.RunAuthLogout = defaults.RunAuthLogout
	}
	if s.ResetProviderIfMatching == nil {
		s.ResetProviderIfMatching = defaults.ResetProviderIfMatching
	}
	return s
}

func NewLogoutCommand(opts Options) *cobra.Command {
	return NewLogoutCommandWithSeams(LogoutSeams{}, opts)
}

func NewLogoutCommandWithSeams(seams LogoutSeams, opts Options) *cobra.Command {
	seams = seams.withDefaults()
	var provider string
	cmd := &cobra.Command{
		Use:          "logout [--provider <provider>]",
		Short:        "Clear stored authentication for a Hermes-compatible provider",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return RunTopLevelLogoutCommand(cmd, provider, seams, opts)
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "provider to log out from: nous, openai-codex, or spotify")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, action, provider, redacted}")
	return cmd
}

func RunTopLevelLogoutCommand(cmd *cobra.Command, providerInput string, seams LogoutSeams, opts Options) error {
	seams = seams.withDefaults()
	provider := seams.NormalizeAuthProvider(providerInput)
	if provider == "" {
		var err error
		provider, err = TopLevelLogoutDefaultProvider(seams)
		if err != nil {
			return err
		}
		if provider == "" {
			return writeTopLevelLogoutDefaultAbsent(cmd, opts)
		}
	}
	if !TopLevelLogoutProviderAllowed(provider) {
		return gormescli.NewExitCodeError(2, fmt.Errorf("gormes logout: auth_logout_provider_unsupported provider=%s allowed=%s", provider, TopLevelLogoutAllowedProviders))
	}
	if err := seams.RunAuthLogout(cmd, provider); err != nil {
		return err
	}
	return seams.ResetProviderIfMatching(provider)
}

func TopLevelLogoutProviderAllowed(provider string) bool {
	switch strings.TrimSpace(provider) {
	case "nous", "openai-codex", "spotify":
		return true
	default:
		return false
	}
}

func TopLevelLogoutDefaultProvider(seams LogoutSeams) (string, error) {
	seams = seams.withDefaults()
	provider, err := seams.ConfiguredProvider()
	if err != nil {
		return "", err
	}
	switch provider {
	case "nous", "openai-codex":
		return provider, nil
	default:
		return "", nil
	}
}

func writeTopLevelLogoutDefaultAbsent(cmd *cobra.Command, opts Options) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		return WriteAuthLifecycleJSON(cmd.OutOrStdout(), AuthLifecycleReportJSON{
			Build:    opts.buildProvenance(),
			Action:   "absent",
			Provider: "auto",
			Redacted: true,
		})
	}
	fmt.Fprintln(cmd.OutOrStdout(), "auth_state_absent provider=auto redacted=true")
	return nil
}

func normalizeLogoutProvider(provider string) string {
	return provideridentity.NormalizeAuthProvider(provider)
}
