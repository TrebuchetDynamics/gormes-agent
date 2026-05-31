package safety

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/commandpolicy"

type CommandClass = commandpolicy.CommandClass

const (
	CommandSafe      CommandClass = commandpolicy.CommandSafe
	CommandUnsafe    CommandClass = commandpolicy.CommandUnsafe
	CommandUncertain CommandClass = commandpolicy.CommandUncertain
)

type CommandClassifier = commandpolicy.CommandClassifier
type CommandClassifierConfig = commandpolicy.CommandClassifierConfig
type CommandAuditEntry = commandpolicy.CommandAuditEntry
type CommandDecision = commandpolicy.CommandDecision

func NewCommandClassifier() *CommandClassifier {
	return commandpolicy.NewCommandClassifier()
}

func NewCommandClassifierWithConfig(cfg CommandClassifierConfig) *CommandClassifier {
	return commandpolicy.NewCommandClassifierWithConfig(cfg)
}
