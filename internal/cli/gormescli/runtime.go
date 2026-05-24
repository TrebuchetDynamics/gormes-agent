package gormescli

import "fmt"

// BuildProvenance is the shared `{version, git_commit}` block emitted by
// command JSON reports. The binary entrypoint owns the actual values and
// injects them into feature modules.
type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// RowBackedStatus is the status string used by command surfaces that are
// intentionally present but still implemented by a future progress row.
const RowBackedStatus = "row_backed"

// ExitCodeError lets importable command modules return process exit intent
// without importing cmd/gormes. cmd/gormes already dispatches by the ExitCode
// interface, so this type preserves that contract.
type ExitCodeError struct {
	code int
	err  error
}

// NewExitCodeError wraps err with an explicit process exit code.
func NewExitCodeError(code int, err error) error {
	if err == nil {
		err = fmt.Errorf("exit code %d", code)
	}
	return ExitCodeError{code: code, err: err}
}

func (e ExitCodeError) Error() string {
	return e.err.Error()
}

func (e ExitCodeError) Unwrap() error {
	return e.err
}

func (e ExitCodeError) ExitCode() int {
	return e.code
}
