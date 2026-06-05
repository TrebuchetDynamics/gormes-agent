package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type gatewayFleetSupervisor interface {
	Status(context.Context) (runtimegateway.FleetStatus, error)
	StartAll(context.Context) (runtimegateway.FleetOperationReport, error)
	StopAll(context.Context) (runtimegateway.FleetOperationReport, error)
	RestartAll(context.Context) (runtimegateway.FleetOperationReport, error)
}

var newGatewayFleetSupervisor = func(cfg config.Config) gatewayFleetSupervisor {
	return runtimegateway.NewFleetSupervisor(cfg, runtimegateway.FleetSupervisorOptions{
		HomeRoot: config.GormesHome(),
		Worker:   runtimegateway.NewCommandFleetWorker(runtimegateway.CommandFleetWorkerOptions{}),
	})
}

func NewFleetCommand(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "fleet",
		Short:        "Inspect and coordinate profile-scoped gateway services",
		SilenceUsage: true,
		RunE:         func(cmd *cobra.Command, args []string) error { return runGatewayFleetStatus(cmd, args, opts) },
	}
	cmd.Flags().Bool("json", false, "print profile fleet status as JSON")
	cmd.AddCommand(newGatewayFleetOperationCommand("start-all", runtimegateway.FleetOperationStartAll, opts))
	cmd.AddCommand(newGatewayFleetOperationCommand("stop-all", runtimegateway.FleetOperationStopAll, opts))
	cmd.AddCommand(newGatewayFleetOperationCommand("restart-all", runtimegateway.FleetOperationRestartAll, opts))
	return cmd
}

func newGatewayFleetOperationCommand(name string, action runtimegateway.FleetOperation, opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          name,
		Short:        fmt.Sprintf("%s profile-scoped gateway services through the fleet supervisor", gatewayFleetOperationVerb(action)),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGatewayFleetOperation(cmd, action, opts)
		},
	}
	cmd.Flags().Bool("json", false, "print profile fleet operation result as JSON")
	return cmd
}

func runGatewayFleetStatus(cmd *cobra.Command, _ []string, opts Options) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	status, err := newGatewayFleetSupervisor(cfg).Status(cmd.Context())
	if err != nil {
		return fmt.Errorf("gateway fleet status: %w", err)
	}
	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		return renderGatewayFleetStatusJSON(cmd.OutOrStdout(), opts, status)
	}
	return renderGatewayFleetStatusText(cmd.OutOrStdout(), status)
}

func runGatewayFleetOperation(cmd *cobra.Command, action runtimegateway.FleetOperation, opts Options) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	supervisor := newGatewayFleetSupervisor(cfg)
	var report runtimegateway.FleetOperationReport
	switch action {
	case runtimegateway.FleetOperationStartAll:
		report, err = supervisor.StartAll(cmd.Context())
	case runtimegateway.FleetOperationStopAll:
		report, err = supervisor.StopAll(cmd.Context())
	case runtimegateway.FleetOperationRestartAll:
		report, err = supervisor.RestartAll(cmd.Context())
	default:
		err = fmt.Errorf("unknown fleet operation %q", action)
	}
	if err != nil {
		return fmt.Errorf("gateway fleet %s: %w", action, err)
	}
	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		return renderGatewayFleetOperationJSON(cmd.OutOrStdout(), opts, report)
	}
	return renderGatewayFleetOperationText(cmd.OutOrStdout(), report)
}

type gatewayFleetStatusJSON struct {
	Build  gormescli.BuildProvenance  `json:"build"`
	Status runtimegateway.FleetStatus `json:"status"`
}

type gatewayFleetOperationJSON struct {
	Build  gormescli.BuildProvenance           `json:"build"`
	Report runtimegateway.FleetOperationReport `json:"report"`
}

func renderGatewayFleetStatusJSON(out io.Writer, opts Options, status runtimegateway.FleetStatus) error {
	payload := gatewayFleetStatusJSON{Build: gatewayBuildProvenance(opts), Status: status}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderGatewayFleetOperationJSON(out io.Writer, opts Options, report runtimegateway.FleetOperationReport) error {
	payload := gatewayFleetOperationJSON{Build: gatewayBuildProvenance(opts), Report: report}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderGatewayFleetStatusText(out io.Writer, status runtimegateway.FleetStatus) error {
	if _, err := fmt.Fprintf(out, "profile fleet: configured=%d enabled=%d healthy=%d degraded=%d disabled=%d\n", status.Summary.ConfiguredProfiles, status.Summary.EnabledProfiles, status.Summary.HealthyProfiles, status.Summary.DegradedProfiles, status.Summary.DisabledProfiles); err != nil {
		return err
	}
	if len(status.Profiles) == 0 {
		_, err := fmt.Fprintln(out, "profiles: none configured")
		return err
	}
	for _, profile := range status.Profiles {
		channels := gatewayFleetChannelNames(profile.Channels)
		if channels == "" {
			channels = "none"
		}
		state := profile.Runtime.State
		if state == "" {
			state = "disabled"
		}
		if _, err := fmt.Fprintf(out, "- %s: enabled=%t health=%s runtime=%s owner=%s channels=%s\n", profile.ProfileID, profile.Enabled, profile.Health, state, profile.Runtime.Owner, channels); err != nil {
			return err
		}
	}
	return nil
}

func renderGatewayFleetOperationText(out io.Writer, report runtimegateway.FleetOperationReport) error {
	if _, err := fmt.Fprintf(out, "gateway fleet %s: targeted=%d succeeded=%d unavailable=%d failed=%d\n", report.Action, report.Summary.TargetedProfiles, report.Summary.Succeeded, report.Summary.Unavailable, report.Summary.Failed); err != nil {
		return err
	}
	for _, result := range report.Results {
		line := fmt.Sprintf("- %s: %s owner=%s", result.ProfileID, result.Status, result.RuntimeOwner)
		if strings.TrimSpace(result.Message) != "" {
			line += " message=" + fmt.Sprintf("%q", result.Message)
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func gatewayFleetChannelNames(channels []runtimegateway.FleetProfileChannelStatus) string {
	if len(channels) == 0 {
		return ""
	}
	names := make([]string, 0, len(channels))
	for _, channel := range channels {
		name := strings.TrimSpace(channel.Channel)
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ",")
}

func gatewayFleetOperationVerb(action runtimegateway.FleetOperation) string {
	switch action {
	case runtimegateway.FleetOperationStartAll:
		return "Start"
	case runtimegateway.FleetOperationStopAll:
		return "Stop"
	case runtimegateway.FleetOperationRestartAll:
		return "Restart"
	default:
		return "Operate on"
	}
}
