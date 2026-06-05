package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/safety"

type CommandClass = safety.CommandClass

const (
	CommandSafe      = safety.CommandSafe
	CommandUnsafe    = safety.CommandUnsafe
	CommandUncertain = safety.CommandUncertain
)

type CommandClassifierConfig = safety.CommandClassifierConfig
type CommandAuditEntry = safety.CommandAuditEntry
type CommandDecision = safety.CommandDecision

type CommandClassifier struct {
	inner *safety.CommandClassifier
}

func NewCommandClassifier() *CommandClassifier {
	return &CommandClassifier{inner: safety.NewCommandClassifier()}
}

func NewCommandClassifierWithConfig(cfg CommandClassifierConfig) *CommandClassifier {
	return &CommandClassifier{inner: safety.NewCommandClassifierWithConfig(cfg)}
}

func (c *CommandClassifier) Classify(command string) CommandClass {
	return c.ensure().Classify(command)
}

func (c *CommandClassifier) ClassifyDetailed(command string) CommandDecision {
	return c.ensure().ClassifyDetailed(command)
}

func (c *CommandClassifier) ClassifyToolRequest(req ToolRequest) CommandDecision {
	return c.ensure().ClassifyToolInput(req.ToolName, []byte(req.Input))
}

func (c *CommandClassifier) ensure() *safety.CommandClassifier {
	if c == nil || c.inner == nil {
		return safety.NewCommandClassifier()
	}
	return c.inner
}
