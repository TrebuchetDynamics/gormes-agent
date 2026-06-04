package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newTTSCommand() *cobra.Command { return gormescli.NewTTSCommand() }
