package jsoninput

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type ErrorReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	Error  string          `json:"error"`
}

type ExitCodeError struct {
	Code int
	Err  error
}

func (e ExitCodeError) Error() string { return e.Err.Error() }
func (e ExitCodeError) Unwrap() error { return e.Err }
func (e ExitCodeError) ExitCode() int { return e.Code }

func Emit(cmd *cobra.Command, action, errMsg string, build BuildProvenance) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(ErrorReportJSON{Build: build, Action: action, Error: errMsg})
	return ExitCodeError{Code: 1, Err: fmt.Errorf("%s", errMsg)}
}

func ArgsIncludeJSONFlag(args []string) bool {
	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "--json" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}
