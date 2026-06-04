package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

const sendBackendUnavailableEvidence = gormescli.SendBackendUnavailableEvidence

type sendCommandDeliveryFunc = gormescli.SendDeliveryFunc
type sendCommandResult = gormescli.SendResult

func newSendCommand(runtime rootRuntime) *cobra.Command {
	return gormescli.NewSendCommand(gormescli.SendOptions{
		Deliver:    runtime.sendMessage,
		IsStdinTTY: sendCommandIsTTY(runtime),
	})
}

func sendCommandIsTTY(runtime rootRuntime) func() bool {
	if runtime.isTTY != nil {
		return runtime.isTTY
	}
	return isStdinTTY
}

func defaultSendCommandBackend(ctx context.Context, target, message string) (sendCommandResult, error) {
	return gormescli.DefaultSendBackend(ctx, target, message)
}
