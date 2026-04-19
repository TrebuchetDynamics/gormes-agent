package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is the Phase-1 semantic marker. Bump when Phase-1 success criteria
// are marked complete in ARCH_PLAN.md.
const Version = "0.1.0-ignition"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print gormes version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("gormes", Version)
	},
}
