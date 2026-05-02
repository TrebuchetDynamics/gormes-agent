package tools

import (
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

func NewCommandClassifier() *CommandClassifier {
	return &CommandClassifier{
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
		},
	}
}

func (c *CommandClassifier) Classify(command string) CommandClass {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return CommandSafe
	}

	for _, pattern := range c.blockedPatterns {
		if pattern.MatchString(trimmed) {
			return CommandUnsafe
		}
	}

	for _, prefix := range c.allowedPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return CommandSafe
		}
	}

	return CommandUncertain
}
