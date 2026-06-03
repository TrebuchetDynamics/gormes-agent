package gormescli

import (
	"context"

	"github.com/spf13/cobra"

	appsend "github.com/TrebuchetDynamics/gormes-agent/internal/app/send"
)

type SendCommandResult = appsend.Result
type SendCommandDeliveryFunc = appsend.DeliveryFunc

type SendCommandOptions struct {
	Deliver    SendCommandDeliveryFunc
	IsStdinTTY func() bool
}

func NewSendCommand(options SendCommandOptions) *cobra.Command {
	return appsend.NewCommand(appsend.Options{
		Deliver:    appsend.DeliveryFunc(options.Deliver),
		IsStdinTTY: options.IsStdinTTY,
	})
}

func DefaultSendCommandBackend(ctx context.Context, target, message string) (SendCommandResult, error) {
	return appsend.DefaultBackend(ctx, target, message)
}
