package main

import (
	"github.com/spf13/cobra"

	ttscmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/tts"
)

func newTTSCommand() *cobra.Command { return ttscmd.NewCommand() }
