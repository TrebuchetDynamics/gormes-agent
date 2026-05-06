package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version marks the current operator-facing release line.
var Version = "0.1.03"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print gormes version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("gormes", Version)
	},
}
