package toolgate

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type CallRequest struct {
	ToolName   string
	Arguments  json.RawMessage
	CallerRole string
}

type Decision struct {
	Allowed bool
	Reason  string
}

type Gate interface {
	CheckTool(ctx context.Context, call CallRequest) (Decision, error)
}

type DefaultGate struct {
	rules []Rule
}

type Rule interface {
	Check(ctx context.Context, call CallRequest) (allowed bool, reason string)
}

func NewDefaultGate() *DefaultGate {
	return &DefaultGate{
		rules: []Rule{
			IntentRule{},
			PermissionRule{},
		},
	}
}

func (g *DefaultGate) AddRule(r Rule) {
	g.rules = append(g.rules, r)
}

func (g *DefaultGate) CheckTool(ctx context.Context, call CallRequest) (Decision, error) {
	for _, rule := range g.rules {
		allowed, reason := rule.Check(ctx, call)
		if !allowed {
			return Decision{Allowed: false, Reason: reason}, nil
		}
	}
	return Decision{Allowed: true}, nil
}

type IntentRule struct{}

var blockedToolPatterns = []*regexp.Regexp{
	regexp.MustCompile(`rm\s+-rf\s+/`),
	regexp.MustCompile(`sudo\s+rm`),
	regexp.MustCompile(`chmod\s+777`),
	regexp.MustCompile(`curl.*\|.*sh`),
	regexp.MustCompile(`wget.*\|.*sh`),
}

func (IntentRule) Check(ctx context.Context, call CallRequest) (bool, string) {
	argsStr := strings.ToLower(string(call.Arguments))
	for _, pattern := range blockedToolPatterns {
		if pattern.MatchString(argsStr) {
			return false, fmt.Sprintf("tool call matches blocked pattern: %s", pattern.String())
		}
	}
	return true, ""
}

type PermissionRule struct{}

var systemOnlyTools = map[string]bool{
	"gateway_restart": true,
	"config_write":    true,
	"session_delete":  true,
}

func (PermissionRule) Check(ctx context.Context, call CallRequest) (bool, string) {
	if systemOnlyTools[call.ToolName] && call.CallerRole != "system" {
		return false, fmt.Sprintf("tool %q requires system role, caller is %q", call.ToolName, call.CallerRole)
	}
	return true, ""
}
