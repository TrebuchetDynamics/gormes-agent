package main

import (
	"context"

	"github.com/spf13/cobra"

	sendcmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/send"
)

const sendBackendUnavailableEvidence = sendcmd.BackendUnavailableEvidence

type sendCommandDeliveryFunc = sendcmd.DeliveryFunc
type sendCommandResult = sendcmd.Result

func newSendCommand(runtime rootRuntime) *cobra.Command {
	return sendcmd.NewCommand(sendcmd.Options{
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
	return sendcmd.DefaultBackend(ctx, target, message)
}
