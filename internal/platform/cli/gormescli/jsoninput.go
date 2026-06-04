package gormescli

import (
	"github.com/spf13/cobra"

	appjsoninput "github.com/TrebuchetDynamics/gormes-agent/internal/app/jsoninput"
)

type JSONInputErrorReport = appjsoninput.ErrorReportJSON

func EmitJSONInputError(cmd *cobra.Command, action, errMsg string, build BuildProvenance) error {
	return appjsoninput.Emit(cmd, action, errMsg, appjsoninput.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit})
}

func ArgsIncludeJSONFlag(args []string) bool {
	return appjsoninput.ArgsIncludeJSONFlag(args)
}
