package channels

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	channelcaps "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
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
	var channel string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:          "channels [channel]",
		Aliases:      []string{"channel"},
		Short:        "Inspect channel capability metadata",
		SilenceUsage: true,
		Args:         validateChannelCapabilityArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := resolveChannelSelection(channel, args)
			if err != nil {
				return err
			}
			return runCapabilitiesCommand(cmd, seams, opts, selected, jsonOutput)
		},
	}
	if cmd.SuggestionsMinimumDistance <= 0 {
		cmd.SuggestionsMinimumDistance = 2
	}
	cmd.Flags().StringVar(&channel, "channel", "", "channel to inspect")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print capabilities as JSON")
	cmd.AddCommand(newCapabilitiesCommand(seams, opts))
	cmd.AddCommand(newChannelSetupGuidanceCommand())
	return cmd
}

func validateChannelCapabilityArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments for %q: pass one channel or use --channel", cmd.CommandPath())
	}
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		return fmt.Errorf("unknown command %q for %q; did you mean %q?", args[0], cmd.CommandPath(), suggestions[0])
	}
	return nil
}

func resolveChannelSelection(flag string, args []string) (string, error) {
	flag = strings.TrimSpace(flag)
	if len(args) == 0 {
		return flag, nil
	}
	arg := strings.TrimSpace(args[0])
	if flag != "" && !strings.EqualFold(flag, arg) {
		return "", fmt.Errorf("channel_selection_conflict: positional channel %q conflicts with --channel %q", arg, flag)
	}
	if flag != "" {
		return flag, nil
	}
	return arg, nil
}

func newCapabilitiesCommand(seams Seams, opts Options) *cobra.Command {
	var channel string
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:          "capabilities [channel]",
		Short:        "Show channel capabilities",
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := resolveChannelSelection(channel, args)
			if err != nil {
				return err
			}
			return runCapabilitiesCommand(cmd, seams, opts, selected, jsonOutput)
		},
	}
	cmd.Flags().StringVar(&channel, "channel", "", "channel to inspect")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print capabilities as JSON")
	return cmd
}

func newChannelSetupGuidanceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "setup [channel]",
		Short:        "Show channel setup commands",
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && textvalue.IsNonBlank(args[0]) {
				return renderChannelSetupGuidance(cmd, args[0])
			}
			return renderAllChannelSetupGuidance(cmd)
		},
	}
	return cmd
}

func renderAllChannelSetupGuidance(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Channel setup commands:")
	fmt.Fprintln(out, "Profile scope: setup writes profiles.<id>.channels.<channel> plus profile-owned credential SecretRefs.")
	for _, channel := range []string{"telegram", "discord", "slack", "whatsapp", "navivox"} {
		fmt.Fprintf(out, "  %s: gormes setup %s", channel, channel)
		if isQuickSetupChannel(channel) {
			fmt.Fprintf(out, " (or gormes setup --quick --target %s)", channel)
		}
		fmt.Fprintln(out)
	}
	fmt.Fprintln(out, "Other channels: run `gormes channels setup <channel>` for row-backed guidance.")
	reports, err := channelcaps.BuildCapabilityReports(channelcaps.CapabilityOptions{})
	if err != nil {
		return gormescli.NewExitCodeError(1, err)
	}
	other := make([]string, 0, len(reports))
	for _, report := range reports {
		if !isQuickSetupChannel(report.Channel) {
			other = append(other, report.Channel)
		}
	}
	if len(other) > 0 {
		fmt.Fprintf(out, "Known manifest channels: %s\n", strings.Join(other, ", "))
	}
	return nil
}

func renderChannelSetupGuidance(cmd *cobra.Command, rawChannel string) error {
	channel := textvalue.LowerTrim(rawChannel)
	if channel == "navivox" {
		out := cmd.OutOrStdout()
		fmt.Fprintln(out, "Channel setup commands for Navivox (navivox):")
		fmt.Fprintln(out, "  Profile scope: profiles.<id>.channels.navivox")
		fmt.Fprintln(out, "  gormes setup navivox")
		fmt.Fprintln(out, "  gormes setup --quick --target navivox")
		fmt.Fprintln(out, "  gormes navivox pair")
		fmt.Fprintln(out, "  gormes channels")
		return nil
	}
	reports, err := channelcaps.BuildCapabilityReports(channelcaps.CapabilityOptions{Channel: channel})
	if err != nil {
		return gormescli.NewExitCodeError(1, err)
	}
	if len(reports) == 0 {
		return gormescli.NewExitCodeError(1, channelcaps.UnknownChannelError{Channel: channel})
	}
	report := reports[0]
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Channel setup commands for %s (%s):\n", report.DisplayName, report.Channel)
	fmt.Fprintf(out, "  Profile scope: profiles.<id>.channels.%s\n", report.Channel)
	if isQuickSetupChannel(report.Channel) {
		fmt.Fprintf(out, "  gormes setup %s\n", report.Channel)
		fmt.Fprintf(out, "  gormes setup --quick --target %s\n", report.Channel)
		fmt.Fprintln(out, "  gormes setup gateway --plan")
	} else {
		fmt.Fprintln(out, "  gormes setup gateway --plan")
		fmt.Fprintln(out, "  gormes config edit")
	}
	fmt.Fprintf(out, "  gormes channels %s\n", report.Channel)
	if textvalue.IsNonBlank(report.BacklogOwner) {
		fmt.Fprintf(out, "  Backlog: %s\n", report.BacklogOwner)
	}
	if textvalue.IsNonBlank(report.Notes) {
		fmt.Fprintf(out, "  Note: %s\n", report.Notes)
	}
	return nil
}

func isQuickSetupChannel(channel string) bool {
	switch textvalue.LowerTrim(channel) {
	case "telegram", "discord", "slack", "whatsapp", "navivox":
		return true
	default:
		return false
	}
}

func runCapabilitiesCommand(cmd *cobra.Command, seams Seams, opts Options, channel string, jsonOutput bool) error {
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
	if textvalue.IsNonBlank(cfg.Telegram.BotToken) || cfg.Telegram.BotTokenRef != nil {
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
	if detail := whatsappConfiguredDetailFromEnv(); detail != "" {
		details["whatsapp"] = detail
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

func whatsappConfiguredDetailFromEnv() string {
	if !strings.EqualFold(strings.TrimSpace(os.Getenv("WHATSAPP_ENABLED")), "true") {
		return ""
	}
	mode := strings.TrimSpace(os.Getenv("WHATSAPP_MODE"))
	if mode == "" {
		return "enabled=true"
	}
	return "mode=" + mode
}
