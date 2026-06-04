package gormescli

import (
	"github.com/spf13/cobra"

	appcompletion "github.com/TrebuchetDynamics/gormes-agent/internal/app/shellcompletion"
)

func NewShellCompletionCommand() *cobra.Command { return appcompletion.NewCommand() }
