package send

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	appsend "github.com/TrebuchetDynamics/gormes-agent/internal/app/send"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

const BackendUnavailableEvidence = appsend.BackendUnavailableEvidence

type DeliveryFunc = appsend.DeliveryFunc
type Result = appsend.Result
type Options = appsend.Options

func NewCommand(options Options) *cobra.Command { return appsend.NewCommand(options) }

func DefaultBackend(ctx context.Context, target, message string) (Result, error) {
	return appsend.DefaultBackend(ctx, target, message)
}

func EmitResult(stdout, stderr io.Writer, result Result, jsonMode, quiet bool) error {
	return appsend.EmitResult(stdout, stderr, result, jsonMode, quiet)
}

func SanitizerEvidence(metas ...cli.TerminalResponseSanitizerMeta) string {
	return appsend.SanitizerEvidence(metas...)
}

func SanitizeResult(result Result) Result { return appsend.SanitizeResult(result) }

func RunListCommand(cmd *cobra.Command, platformFilter string, jsonMode bool) error {
	return appsend.RunListCommand(cmd, platformFilter, jsonMode)
}
