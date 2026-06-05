package rowbacked

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

const GatewayCronRow = "Gateway, platform, webhook, and cron management CLI"

// Options carries binary-owned values into the gateway module without making
// importable command code depend on cmd/gormes.
type Options struct {
	BuildProvenance func() gormescli.BuildProvenance
	ExitError       func(code int, err error) error

	TermuxDetected              func() bool
	TermuxLifecycleGuidanceLine string
	TermuxNotificationStatus    func() string
}

func rowBackedOptions(opts Options) gormescli.RowBackedCommandOptions {
	return gormescli.RowBackedCommandOptions{BuildProvenance: opts.BuildProvenance}
}

func NewWebhookCommand(opts Options) *cobra.Command {
	return newRowBackedParent(
		"webhook",
		"Manage Hermes-compatible webhook subscriptions",
		opts,
		newRowBackedCommand("subscribe <target>", []string{"add"}, "Subscribe a webhook target", false, nil, opts),
		newRowBackedCommand("list", []string{"ls"}, "List webhook subscriptions", false, nil, opts),
		newRowBackedCommand("remove <id>", []string{"rm"}, "Remove a webhook subscription", true, yesFlag, opts),
		newRowBackedCommand("test <id>", nil, "Send a test webhook event", false, nil, opts),
	)
}

func NewHooksCommand(opts Options) *cobra.Command {
	return newRowBackedParent(
		"hooks",
		"Manage local Hermes-compatible hook registrations",
		opts,
		newRowBackedCommand("list", []string{"ls"}, "List hook registrations", false, nil, opts),
		newRowBackedCommand("test <name>", nil, "Test a hook registration", false, nil, opts),
		newRowBackedCommand("revoke <name>", []string{"remove", "rm"}, "Revoke a hook registration", true, yesFlag, opts),
		newRowBackedCommand("doctor", nil, "Inspect hook configuration health", false, nil, opts),
	)
}

func NewPairingCommand(opts Options) *cobra.Command {
	return newRowBackedParent(
		"pairing",
		"Manage gateway pairing requests",
		opts,
		newRowBackedCommand("list", nil, "List pending pairing requests", false, nil, opts),
		newRowBackedCommand("approve <id>", nil, "Approve a pairing request", false, nil, opts),
		newRowBackedCommand("revoke <id>", nil, "Revoke an approved pairing", true, nil, opts),
		newRowBackedCommand("clear-pending", nil, "Clear pending pairing requests", true, yesFlag, opts),
	)
}

func newRowBackedParent(use, short string, opts Options, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	cmd.AddCommand(children...)
	return cmd
}

func newRowBackedCommand(use string, aliases []string, short string, destructive bool, flagSet func(*cobra.Command), opts Options) *cobra.Command {
	return gormescli.NewRowBackedCommand(gormescli.RowBackedCommandSpec{
		Use:         use,
		Aliases:     aliases,
		Short:       short,
		Row:         GatewayCronRow,
		Destructive: destructive,
		FlagSet:     flagSet,
	}, rowBackedOptions(opts))
}

func yesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
}
