package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"

type BlocklistCategory = safety.BlocklistCategory

const (
	BlocklistDestructive = safety.BlocklistDestructive
	BlocklistNetwork     = safety.BlocklistNetwork
	BlocklistPrivilege   = safety.BlocklistPrivilege
	BlocklistCryptoMine  = safety.BlocklistCryptoMine
	BlocklistDataExfil   = safety.BlocklistDataExfil
)

type BlocklistPattern = safety.BlocklistPattern
type ShellBlocklistResult = safety.ShellBlocklistResult

var ShellBlocklistPatterns = safety.ShellBlocklistPatterns

func CheckShellBlocklist(command string) ShellBlocklistResult {
	return safety.CheckShellBlocklist(command)
}

func IsShellCommandBlocked(command string) bool {
	return safety.IsShellCommandBlocked(command)
}

func GetBlocklistCoverage() map[string]int {
	return safety.GetBlocklistCoverage()
}

func GetBlocklistTotal() int {
	return safety.GetBlocklistTotal()
}

func IsHardlineCommand(command string) bool {
	return safety.IsHardlineCommand(command)
}

func IsRecoverableCommand(command string) bool {
	return safety.IsRecoverableCommand(command)
}
