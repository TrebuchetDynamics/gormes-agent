package gormescli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/commandruntime"

// BuildProvenance is the shared `{version, git_commit}` block emitted by
// command JSON reports. The binary entrypoint owns the actual values and
// injects them into feature modules.
type BuildProvenance = commandruntime.BuildProvenance

// RowBackedStatus is the status string used by command surfaces that are
// intentionally present but still implemented by a future progress row.
const RowBackedStatus = commandruntime.RowBackedStatus

// ExitCodeError lets importable command modules return process exit intent
// without importing cmd/gormes. cmd/gormes already dispatches by the ExitCode
// interface, so this type preserves that contract.
type ExitCodeError = commandruntime.ExitCodeError

// NewExitCodeError wraps err with an explicit process exit code.
func NewExitCodeError(code int, err error) error {
	return commandruntime.NewExitCodeError(code, err)
}
