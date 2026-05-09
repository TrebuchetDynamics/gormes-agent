package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// jsonInputErrorReportJSON is the wire shape for `--json` invalid-input
// failures: missing required argument, missing required flag, unknown
// subcommand under a parent, or "no data yet" (e.g. logs --json before
// any log file exists). Same shape across every command that opts into
// this conformance fence so fleet automation parses one type and
// distinguishes "user mistake" from "command crashed" by reading
// `action`.
//
// `error` carries the human-readable message that would have been
// rendered to stderr by cobra's default error path. Callers should not
// include secrets in `error`; it goes onto stdout in the JSON document.
type jsonInputErrorReportJSON struct {
	Build  buildProvenanceJSON `json:"build"`
	Action string              `json:"action"`
	Error  string              `json:"error"`
}

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
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(jsonInputErrorReportJSON{
		Build:  newBuildProvenance(),
		Action: action,
		Error:  errMsg,
	})
	return newExitCodeError(1, fmt.Errorf("%s", errMsg))
}

func argsIncludeJSONFlag(args []string) bool {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "--json" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}
