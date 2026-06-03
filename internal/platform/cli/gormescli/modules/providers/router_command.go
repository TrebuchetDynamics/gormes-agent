package providers

import (
	"io"

	approuter "github.com/TrebuchetDynamics/gormes-agent/internal/app/router"
	"github.com/spf13/cobra"
)

type RouterCommandOptions = approuter.CommandOptions

func NewRouterCommand(opts RouterCommandOptions) *cobra.Command {
	return approuter.NewCommand(opts)
}

func printRouterDryRun(out io.Writer, model approuter.ReadModel) {
	approuter.PrintDryRun(out, model)
}

func routerOpenAIBaseURL(listen string) string {
	return approuter.OpenAIBaseURL(listen)
}
