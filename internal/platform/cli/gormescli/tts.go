package gormescli

import (
	"github.com/spf13/cobra"

	apptts "github.com/TrebuchetDynamics/gormes-agent/internal/app/tts"
)

func NewTTSCommand() *cobra.Command { return apptts.NewCommand() }
