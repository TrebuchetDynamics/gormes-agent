package providers

import (
	"io"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/routercmd"
	routerpkg "github.com/TrebuchetDynamics/gormes-agent/internal/provider/router"
	"github.com/spf13/cobra"
)

type RouterCommandOptions = routercmd.Options

func NewRouterCommand(opts RouterCommandOptions) *cobra.Command {
	return routercmd.New(opts)
}

func printRouterDryRun(out io.Writer, model routerpkg.ReadModel) {
	routercmd.PrintDryRun(out, model)
}

func routerOpenAIBaseURL(listen string) string {
	return routercmd.OpenAIBaseURL(listen)
}
