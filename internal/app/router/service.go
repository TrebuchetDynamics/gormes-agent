package router

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/routercmd"
	routerpkg "github.com/TrebuchetDynamics/gormes-agent/internal/provider/router"
)

type CommandOptions = routercmd.Options
type ReadModel = routerpkg.ReadModel

func NewCommand(opts CommandOptions) *cobra.Command {
	return routercmd.New(opts)
}

func PrintDryRun(out io.Writer, model ReadModel) {
	routercmd.PrintDryRun(out, model)
}

func OpenAIBaseURL(listen string) string {
	return routercmd.OpenAIBaseURL(listen)
}
