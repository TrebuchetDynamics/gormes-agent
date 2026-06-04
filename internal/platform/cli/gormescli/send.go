package gormescli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	appsend "github.com/TrebuchetDynamics/gormes-agent/internal/app/send"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

const SendBackendUnavailableEvidence = appsend.BackendUnavailableEvidence

type SendDeliveryFunc = appsend.DeliveryFunc
type SendResult = appsend.Result
type SendOptions = appsend.Options

func NewSendCommand(options SendOptions) *cobra.Command { return appsend.NewCommand(options) }

func DefaultSendBackend(ctx context.Context, target, message string) (SendResult, error) {
	return appsend.DefaultBackend(ctx, target, message)
}

func EmitSendResult(stdout, stderr io.Writer, result SendResult, jsonMode, quiet bool) error {
	return appsend.EmitResult(stdout, stderr, result, jsonMode, quiet)
}

func SendSanitizerEvidence(metas ...cli.TerminalResponseSanitizerMeta) string {
	return appsend.SanitizerEvidence(metas...)
}

func SanitizeSendResult(result SendResult) SendResult { return appsend.SanitizeResult(result) }

func RunSendListCommand(cmd *cobra.Command, platformFilter string, jsonMode bool) error {
	return appsend.RunListCommand(cmd, platformFilter, jsonMode)
}
