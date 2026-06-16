package commandpolicy

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety/commandpolicy/classifier"

type CommandClass = classifier.CommandClass

const (
	CommandSafe      CommandClass = classifier.CommandSafe
	CommandUnsafe    CommandClass = classifier.CommandUnsafe
	CommandUncertain CommandClass = classifier.CommandUncertain
)

type CommandClassifier = classifier.CommandClassifier
type CommandClassifierConfig = classifier.CommandClassifierConfig
type CommandAuditEntry = classifier.CommandAuditEntry
type CommandDecision = classifier.CommandDecision

func NewCommandClassifier() *CommandClassifier {
	return classifier.NewCommandClassifier()
}

func NewCommandClassifierWithConfig(cfg CommandClassifierConfig) *CommandClassifier {
	return classifier.NewCommandClassifierWithConfig(cfg)
}
