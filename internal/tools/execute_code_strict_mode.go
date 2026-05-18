package tools

import (
	"context"
	"os/exec"
)

// StrictModeSandbox pins the current Gormes security-first sandbox contract
// as "strict mode": isolated tmpdir CWD, canonical interpreter resolved via
// the system PATH (no venv/conda detection), and the existing sandbox guard
// that blocks filesystem/network access patterns.
//
// This wrapper freezes the LocalCodeSandbox behavior so the subsequent
// ProjectModeSandbox can relax specific constraints without accidentally
// widening strict-mode access.
type StrictModeSandbox struct {
	inner *LocalCodeSandbox
}

// NewStrictModeSandbox creates a sandbox with canonical system-PATH
// interpreter resolution. The lookPath defaults to exec.LookPath so
// resolveRuntime finds the system shell without venv or conda paths.
func NewStrictModeSandbox() *StrictModeSandbox {
	return &StrictModeSandbox{
		inner: NewLocalCodeSandbox(),
	}
}

func newStrictModeSandboxWithSubprocessHome(resolve SubprocessHomeResolver) *StrictModeSandbox {
	s := NewStrictModeSandbox()
	s.inner.subprocessHome = resolve
	return s
}

// newStrictModeSandboxWithLookPath is a test seam that injects a fake
// lookPath function so tests can assert canonical resolution without
// depending on the host's real shell binary.
func newStrictModeSandboxWithLookPath(lookPath func(string) (string, error)) *StrictModeSandbox {
	s := &LocalCodeSandbox{
		lookPath: lookPath,
		languages: map[string]runtimeSpec{
			"sh":    {Binaries: []string{"sh"}, Extension: ".sh"},
			"shell": {Binaries: []string{"sh"}, Extension: ".sh"},
		},
	}
	return &StrictModeSandbox{inner: s}
}

// Execute delegates to the inner LocalCodeSandbox. Strict mode is the
// current (pre-project-mode) behavior: no relaxed constraints.
func (s *StrictModeSandbox) Execute(ctx context.Context, req CodeExecutionRequest) (CodeExecutionResult, error) {
	return s.inner.Execute(ctx, req)
}

// StrictModeBlockedResultEnvelope is the canonical reference shape for the
// 5.K blocked-result envelope. Every mode must preserve this field set so
// callers can rely on a stable JSON contract regardless of execution mode.
type StrictModeBlockedResultEnvelope struct {
	Status           string `json:"status"`
	Error            string `json:"error"`
	FilesystemAccess bool   `json:"filesystem_access"`
	NetworkAccess    bool   `json:"network_access"`
}

// StrictModeBlockedEnvelopeShape returns a zero-value reference of the
// canonical blocked-result envelope. Tests should marshal and unmarshal
// this to verify the JSON field names match CodeExecutionResult.
func StrictModeBlockedEnvelopeShape() StrictModeBlockedResultEnvelope {
	return StrictModeBlockedResultEnvelope{}
}

// StrictModeLookPath returns the canonical system lookPath used by strict
// mode — exec.LookPath without venv or conda prefix injection.
func StrictModeLookPath() func(string) (string, error) {
	return exec.LookPath
}
