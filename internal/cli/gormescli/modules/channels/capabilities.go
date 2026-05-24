package channels

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	channelcaps "github.com/TrebuchetDynamics/gormes-agent/internal/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// Options carries binary-owned build metadata into importable channels
// commands without making them depend on cmd/gormes.
type Options struct {
	BuildProvenance func() gormescli.BuildProvenance
}

func (o Options) buildProvenance() gormescli.BuildProvenance {
	if o.BuildProvenance == nil {
		return gormescli.BuildProvenance{}
	}
	return o.BuildProvenance()
}

// Seams isolates config loading and redacted config detail calculation for
// hermetic command tests.
type Seams struct {
	LoadConfig        func() (config.Config, error)
	ConfiguredDetails func(config.Config) map[string]string
}

func DefaultSeams() Seams {
	return Seams{
		LoadConfig:        func() (config.Config, error) { return config.Load(nil) },
		ConfiguredDetails: defaultConfiguredChannelDetails,
	}
}

func (s Seams) withDefaults() Seams {
	defaults := DefaultSeams()
	if s.LoadConfig == nil {
		s.LoadConfig = defaults.LoadConfig
	}
	if s.ConfiguredDetails == nil {
		s.ConfiguredDetails = defaults.ConfiguredDetails
	}
	return s
}

func NewCommand(opts Options) *cobra.Command {
	return NewCommandWithSeams(Seams{}, opts)
}

func NewCommandWithSeams(seams Seams, opts Options) *cobra.Command {
	seams = seams.withDefaults()
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "Inspect channel capability metadata",
	}
	cmd.AddCommand(newCapabilitiesCommand(seams, opts))
	return cmd
}

func newCapabilitiesCommand(seams Seams, opts Options) *cobra.Command {
	var channel string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:          "capabilities",
		Short:        "Show channel capabilities",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := seams.LoadConfig()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			reports, err := channelcaps.BuildCapabilityReports(channelcaps.CapabilityOptions{
				Configured: seams.ConfiguredDetails(cfg),
				Channel:    channel,
			})
			if err != nil {
				return gormescli.NewExitCodeError(1, err)
			}
			if jsonOutput {
				return renderCapabilitiesJSON(cmd, reports, opts)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), RenderCapabilitiesText(reports))
			return err
		},
	}
	cmd.Flags().StringVar(&channel, "channel", "", "channel to inspect")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print capabilities as JSON")
	return cmd
}

func renderCapabilitiesJSON(cmd *cobra.Command, reports []channelcaps.CapabilityReport, opts Options) error {
	payload := struct {
		Build    gormescli.BuildProvenance      `json:"build"`
		Channels []channelcaps.CapabilityReport `json:"channels"`
	}{
		Build:    opts.buildProvenance(),
		Channels: reports,
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func RenderCapabilitiesText(reports []channelcaps.CapabilityReport) string {
	var b strings.Builder
	for i, report := range reports {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "%s (%s)\n", report.DisplayName, report.Channel)
		status := "not configured"
		if report.Configured {
			status = "configured"
		}
		if report.ConfigDetail != "" {
			fmt.Fprintf(&b, "Status: %s (%s)\n", status, report.ConfigDetail)
		} else {
			fmt.Fprintf(&b, "Status: %s\n", status)
		}
		fmt.Fprintf(&b, "Support: %s\n", strings.Join(report.Features, " "))
		if len(report.Intents) > 0 {
			fmt.Fprintf(&b, "Intents: %s\n", strings.Join(report.Intents, ", "))
		}
		if len(report.Scopes) > 0 {
			fmt.Fprintf(&b, "Scopes: %s\n", strings.Join(report.Scopes, ", "))
		}
		if len(report.FormatLimitations) > 0 {
			fmt.Fprintf(&b, "Format limitations: %s\n", strings.Join(report.FormatLimitations, ", "))
		}
		if len(report.Degraded) > 0 {
			fmt.Fprintf(&b, "Degraded: %s\n", strings.Join(report.Degraded, ", "))
		}
	}
	if len(reports) > 0 {
		b.WriteString("\n")
	}
	return b.String()
}

func defaultConfiguredChannelDetails(cfg config.Config) map[string]string {
	details := map[string]string{}
	if strings.TrimSpace(cfg.Telegram.BotToken) != "" || cfg.Telegram.BotTokenRef != nil {
		allowed := cfg.Telegram.AllowedChatIDs()
		if len(allowed) > 0 {
			details["telegram"] = "allowed_chat_ids=" + strconv.Itoa(len(allowed))
		} else {
			details["telegram"] = "bot_token_set=true"
		}
	}
	if cfg.Discord.Enabled() {
		detail := "first_run_discovery=" + strconv.FormatBool(cfg.Discord.FirstRunDiscovery)
		if cfg.Discord.AllowedChannelID != "" {
			detail = "allowed_channel_id=" + cfg.Discord.AllowedChannelID
		}
		details["discord"] = detail
	}
	if cfg.Slack.Enabled {
		if cfg.Slack.AllowedChannelID != "" {
			details["slack"] = "allowed_channel_id=" + cfg.Slack.AllowedChannelID
		} else {
			details["slack"] = "enabled=true"
		}
	}
	if cfg.Teams.Enabled {
		details["teams"] = cfg.Teams.RedactedStatus()
	}
	if cfg.Yuanbao.Enabled {
		details["yuanbao"] = cfg.Yuanbao.RedactedStatus()
	}
	return details
}
