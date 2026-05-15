package tools

import (
	"encoding/json"
	"regexp"
	"strings"
)

type CommandClass int

const (
	CommandSafe CommandClass = iota
	CommandUnsafe
	CommandUncertain
)

func (c CommandClass) String() string {
	switch c {
	case CommandSafe:
		return "safe"
	case CommandUnsafe:
		return "unsafe"
	case CommandUncertain:
		return "uncertain"
	default:
		return "unknown"
	}
}

type CommandClassifier struct {
	allowedPrefixes []string
	blockedPatterns []*regexp.Regexp
}

type CommandClassifierConfig struct {
	AllowedPrefixes []string
	BlockedPatterns []string
}

type CommandAuditEntry struct {
	Command          string `json:"command"`
	Classification   string `json:"classification"`
	Action           string `json:"action"`
	Reason           string `json:"reason,omitempty"`
	Blocked          bool   `json:"blocked"`
	RequiresSnapshot bool   `json:"requires_snapshot"`
}

type CommandDecision struct {
	Command          string
	Class            CommandClass
	Blocked          bool
	RequiresSnapshot bool
	Audit            CommandAuditEntry
}

func NewCommandClassifier() *CommandClassifier {
	return NewCommandClassifierWithConfig(CommandClassifierConfig{})
}

func NewCommandClassifierWithConfig(cfg CommandClassifierConfig) *CommandClassifier {
	classifier := &CommandClassifier{
		allowedPrefixes: []string{
			"echo", "ls", "cat", "head", "tail", "wc", "grep", "find",
			"pwd", "whoami", "date", "uname", "uptime",
			"go test", "go build", "go vet", "go fmt",
			"npm test", "npm run", "npm install",
			"git status", "git diff", "git log", "git branch",
			"python", "node", "make",
		},
		blockedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`rm\s+(-rf?|--recursive)`),
			regexp.MustCompile(`sudo`),
			regexp.MustCompile(`mkfs`),
			regexp.MustCompile(`dd\s+if=`),
			regexp.MustCompile(`>/etc/`),
			regexp.MustCompile(`curl.*\|.*sh`),
			regexp.MustCompile(`wget.*\|.*sh`),
			regexp.MustCompile(`chmod\s+[0-7]*7`),
			regexp.MustCompile(`git\s+push\s+--force`),
			regexp.MustCompile(`(?i)` + findRootWalkCommand),
		},
	}
	classifier.allowedPrefixes = append(classifier.allowedPrefixes, cfg.AllowedPrefixes...)
	for _, pattern := range cfg.BlockedPatterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		classifier.blockedPatterns = append(classifier.blockedPatterns, regexp.MustCompile(pattern))
	}
	return classifier
}

func (c *CommandClassifier) Classify(command string) CommandClass {
	return c.ClassifyDetailed(command).Class
}

func (c *CommandClassifier) ClassifyDetailed(command string) CommandDecision {
	if c == nil {
		c = NewCommandClassifier()
	}
	trimmed := strings.TrimSpace(command)
	decision := CommandDecision{Command: trimmed}
	if trimmed == "" {
		decision.Class = CommandSafe
		decision.Audit = commandAudit(trimmed, CommandSafe, "execute_direct", "", false, false)
		return decision
	}

	for _, pattern := range c.blockedPatterns {
		if pattern.MatchString(trimmed) {
			decision.Class = CommandUnsafe
			decision.Blocked = true
			decision.Audit = commandAudit(trimmed, CommandUnsafe, "blocked", "matched unsafe command pattern", true, false)
			return decision
		}
	}

	lower := strings.ToLower(trimmed)
	for _, prefix := range c.allowedPrefixes {
		if commandPrefixMatches(lower, strings.ToLower(prefix)) {
			decision.Class = CommandSafe
			decision.Audit = commandAudit(trimmed, CommandSafe, "execute_direct", "matched safe command prefix", false, false)
			return decision
		}
	}

	decision.Class = CommandUncertain
	decision.RequiresSnapshot = true
	decision.Audit = commandAudit(trimmed, CommandUncertain, "snapshot_required", "no safe or unsafe rule matched", false, true)
	return decision
}

func (c *CommandClassifier) ClassifyToolRequest(req ToolRequest) CommandDecision {
	command := extractToolCommand(req)
	return c.ClassifyDetailed(command)
}

func commandAudit(command string, class CommandClass, action, reason string, blocked, snapshot bool) CommandAuditEntry {
	return CommandAuditEntry{
		Command:          command,
		Classification:   class.String(),
		Action:           action,
		Reason:           reason,
		Blocked:          blocked,
		RequiresSnapshot: snapshot,
	}
}

func commandPrefixMatches(command, prefix string) bool {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || !strings.HasPrefix(command, prefix) {
		return false
	}
	if len(command) == len(prefix) {
		return true
	}
	switch command[len(prefix)] {
	case ' ', '\t', '\n', '\r':
		return true
	default:
		return false
	}
}

func extractToolCommand(req ToolRequest) string {
	raw := strings.TrimSpace(string(req.Input))
	if raw == "" {
		return strings.TrimSpace(req.ToolName)
	}
	var asString string
	if err := json.Unmarshal(req.Input, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	var payload map[string]any
	if err := json.Unmarshal(req.Input, &payload); err != nil {
		return raw
	}
	for _, key := range []string{"command", "cmd", "code", "script"} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return strings.TrimSpace(req.ToolName)
}
