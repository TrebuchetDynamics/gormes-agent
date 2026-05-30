package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands"

var ErrBusyCommandActive = commands.ErrBusyCommandActive
var ErrBusyCommandInvalid = commands.ErrBusyCommandInvalid

type BusyCommandGuard = commands.BusyCommandGuard

type BusyInputVerdict = commands.BusyInputVerdict

func NewBusyCommandGuard() *BusyCommandGuard { return commands.NewBusyCommandGuard() }
