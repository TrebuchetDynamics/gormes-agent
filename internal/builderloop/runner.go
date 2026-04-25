package builderloop

import "github.com/TrebuchetDynamics/gormes-agent/internal/cmdrunner"

// The runner plumbing now lives in internal/cmdrunner so plannerloop does not
// have to import builderloop purely for it. These aliases keep existing
// builderloop call sites compiling against the unqualified names.

type Command = cmdrunner.Command
type Result = cmdrunner.Result
type Runner = cmdrunner.Runner
type ExecRunner = cmdrunner.ExecRunner
type FakeRunner = cmdrunner.FakeRunner

var ErrUnexpectedCommand = cmdrunner.ErrUnexpectedCommand
