package safety

import (
	"regexp"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/shellblocklist"
)

type BlocklistCategory = shellblocklist.BlocklistCategory

const (
	BlocklistDestructive BlocklistCategory = shellblocklist.BlocklistDestructive
	BlocklistNetwork     BlocklistCategory = shellblocklist.BlocklistNetwork
	BlocklistPrivilege   BlocklistCategory = shellblocklist.BlocklistPrivilege
	BlocklistCryptoMine  BlocklistCategory = shellblocklist.BlocklistCryptoMine
	BlocklistDataExfil   BlocklistCategory = shellblocklist.BlocklistDataExfil
)

type BlocklistPattern struct {
	Pattern     *regexp.Regexp
	Category    BlocklistCategory
	Description string
}

type ShellBlocklistResult = shellblocklist.ShellBlocklistResult

var ShellBlocklistPatterns = shellBlocklistPatterns()

func shellBlocklistPatterns() []BlocklistPattern {
	patterns := shellblocklist.ShellBlocklistPatterns
	result := make([]BlocklistPattern, 0, len(patterns))
	for _, pattern := range patterns {
		result = append(result, BlocklistPattern{
			Pattern:     pattern.Pattern,
			Category:    pattern.Category,
			Description: pattern.Description,
		})
	}
	return result
}

func CheckShellBlocklist(command string) ShellBlocklistResult {
	return shellblocklist.CheckShellBlocklist(command)
}

func IsShellCommandBlocked(command string) bool {
	return shellblocklist.IsShellCommandBlocked(command)
}

func GetBlocklistCoverage() map[string]int {
	return shellblocklist.GetBlocklistCoverage()
}

func GetBlocklistTotal() int {
	return shellblocklist.GetBlocklistTotal()
}

func IsHardlineCommand(command string) bool {
	return shellblocklist.IsHardlineCommand(command)
}

func IsRecoverableCommand(command string) bool {
	return shellblocklist.IsRecoverableCommand(command)
}
