package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type gatewayFleetSupervisor interface {
	Status(context.Context) (gateway.FleetStatus, error)
	StartAll(context.Context) (gateway.FleetOperationReport, error)
	StopAll(context.Context) (gateway.FleetOperationReport, error)
	RestartAll(context.Context) (gateway.FleetOperationReport, error)
}

var newGatewayFleetSupervisor = func(cfg config.Config) gatewayFleetSupervisor {
	return gateway.NewFleetSupervisor(cfg, gateway.FleetSupervisorOptions{
		HomeRoot: config.GormesHome(),
		Worker:   gateway.NewCommandFleetWorker(gateway.CommandFleetWorkerOptions{}),
	})
}

func newGatewayFleetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "fleet",
		Short:        "Inspect and coordinate profile-scoped gateway services",
		SilenceUsage: true,
		RunE:         runGatewayFleetStatus,
	}
	cmd.Flags().Bool("json", false, "print profile fleet status as JSON")
	cmd.AddCommand(newGatewayFleetOperationCommand("start-all", gateway.FleetOperationStartAll))
	cmd.AddCommand(newGatewayFleetOperationCommand("stop-all", gateway.FleetOperationStopAll))
	cmd.AddCommand(newGatewayFleetOperationCommand("restart-all", gateway.FleetOperationRestartAll))
	return cmd
}

func newGatewayFleetOperationCommand(name string, action gateway.FleetOperation) *cobra.Command {
	cmd := &cobra.Command{
		Use:          name,
		Short:        fmt.Sprintf("%s profile-scoped gateway services through the fleet supervisor", gatewayFleetOperationVerb(action)),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runGatewayFleetOperation(cmd, action)
		},
	}
	cmd.Flags().Bool("json", false, "print profile fleet operation result as JSON")
	return cmd
}

func runGatewayFleetStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	status, err := newGatewayFleetSupervisor(cfg).Status(cmd.Context())
	if err != nil {
		return fmt.Errorf("gateway fleet status: %w", err)
	}
	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		return renderGatewayFleetStatusJSON(cmd.OutOrStdout(), status)
	}
	return renderGatewayFleetStatusText(cmd.OutOrStdout(), status)
}

func runGatewayFleetOperation(cmd *cobra.Command, action gateway.FleetOperation) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	supervisor := newGatewayFleetSupervisor(cfg)
	var report gateway.FleetOperationReport
	switch action {
	case gateway.FleetOperationStartAll:
		report, err = supervisor.StartAll(cmd.Context())
	case gateway.FleetOperationStopAll:
		report, err = supervisor.StopAll(cmd.Context())
	case gateway.FleetOperationRestartAll:
		report, err = supervisor.RestartAll(cmd.Context())
	default:
		err = fmt.Errorf("unknown fleet operation %q", action)
	}
	if err != nil {
		return fmt.Errorf("gateway fleet %s: %w", action, err)
	}
	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		return renderGatewayFleetOperationJSON(cmd.OutOrStdout(), report)
	}
	return renderGatewayFleetOperationText(cmd.OutOrStdout(), report)
}

type gatewayFleetStatusJSON struct {
	Build  buildProvenanceJSON `json:"build"`
	Status gateway.FleetStatus `json:"status"`
}

type gatewayFleetOperationJSON struct {
	Build  buildProvenanceJSON          `json:"build"`
	Report gateway.FleetOperationReport `json:"report"`
}

func renderGatewayFleetStatusJSON(out io.Writer, status gateway.FleetStatus) error {
	payload := gatewayFleetStatusJSON{Build: newBuildProvenance(), Status: status}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderGatewayFleetOperationJSON(out io.Writer, report gateway.FleetOperationReport) error {
	payload := gatewayFleetOperationJSON{Build: newBuildProvenance(), Report: report}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func renderGatewayFleetStatusText(out io.Writer, status gateway.FleetStatus) error {
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

func renderGatewayFleetOperationText(out io.Writer, report gateway.FleetOperationReport) error {
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

func gatewayFleetChannelNames(channels []gateway.FleetProfileChannelStatus) string {
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

func gatewayFleetOperationVerb(action gateway.FleetOperation) string {
	switch action {
	case gateway.FleetOperationStartAll:
		return "Start"
	case gateway.FleetOperationStopAll:
		return "Stop"
	case gateway.FleetOperationRestartAll:
		return "Restart"
	default:
		return "Operate on"
	}
}
