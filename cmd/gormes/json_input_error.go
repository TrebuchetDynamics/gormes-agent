package main

import (
	"github.com/spf13/cobra"

	jsoninput "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/jsoninput"
)

// jsonInputErrorReportJSON is the wire shape for `--json` invalid-input
// failures: missing required argument, missing required flag, unknown
// subcommand under a parent, or "no data yet" (e.g. logs --json before
// any log file exists). Same shape across every command that opts into
// this conformance fence so fleet automation parses one type and
// distinguishes "user mistake" from "command crashed" by reading
// `action`.
type jsonInputErrorReportJSON = jsoninput.ErrorReportJSON

// emitJSONInputError writes a structured invalid-input report to cmd's
// stdout and returns a non-zero exit-code error so the cobra runner
// exits 1 while the JSON document on stdout stays parseable for fleet
// automation. Mirrors the kanban not-found contract (slice 42) and the
// session delete --json action: "not_found" path.
//
// `action` discriminates the failure kind so scripts can branch:
//   - "missing_argument"   — required positional arg was omitted
//   - "missing_flag"       — required flag was omitted
//   - "unknown_subcommand" — typo / unknown verb under a parent
//   - "no_logs"            — no log file/state exists yet
//   - "no_data"            — generic "lookup target absent" (rare)
func emitJSONInputError(cmd *cobra.Command, action, errMsg string) error {
	err := jsoninput.Emit(cmd, action, errMsg, jsoninput.BuildProvenance(newBuildProvenance()))
	return newExitCodeError(1, err)
}

func argsIncludeJSONFlag(args []string) bool {
	return jsoninput.ArgsIncludeJSONFlag(args)
}
