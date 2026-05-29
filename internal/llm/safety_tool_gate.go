package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type ToolCallRequest struct {
	ToolName   string
	Arguments  json.RawMessage
	CallerRole string
}

type ToolGateDecision struct {
	Allowed bool
	Reason  string
}

type ToolGate interface {
	CheckTool(ctx context.Context, call ToolCallRequest) (ToolGateDecision, error)
}

type DefaultToolGate struct {
	rules []ToolRule
}

type ToolRule interface {
	Check(ctx context.Context, call ToolCallRequest) (allowed bool, reason string)
}

func NewDefaultToolGate() *DefaultToolGate {
	return &DefaultToolGate{
		rules: []ToolRule{
			ToolIntentRule{},
			ToolPermissionRule{},
		},
	}
}

func (g *DefaultToolGate) AddRule(r ToolRule) {
	g.rules = append(g.rules, r)
}

func (g *DefaultToolGate) CheckTool(ctx context.Context, call ToolCallRequest) (ToolGateDecision, error) {
	for _, rule := range g.rules {
		allowed, reason := rule.Check(ctx, call)
		if !allowed {
			return ToolGateDecision{Allowed: false, Reason: reason}, nil
		}
	}
	return ToolGateDecision{Allowed: true}, nil
}

type ToolIntentRule struct{}

var blockedToolPatterns = []*regexp.Regexp{
	regexp.MustCompile(`rm\s+-rf\s+/`),
	regexp.MustCompile(`sudo\s+rm`),
	regexp.MustCompile(`chmod\s+777`),
	regexp.MustCompile(`curl.*\|.*sh`),
	regexp.MustCompile(`wget.*\|.*sh`),
}

func (ToolIntentRule) Check(ctx context.Context, call ToolCallRequest) (bool, string) {
	argsStr := strings.ToLower(string(call.Arguments))
	for _, pattern := range blockedToolPatterns {
		if pattern.MatchString(argsStr) {
			return false, fmt.Sprintf("tool call matches blocked pattern: %s", pattern.String())
		}
	}
	return true, ""
}

type ToolPermissionRule struct{}

var systemOnlyTools = map[string]bool{
	"gateway_restart": true,
	"config_write":    true,
	"session_delete":  true,
}

func (ToolPermissionRule) Check(ctx context.Context, call ToolCallRequest) (bool, string) {
	if systemOnlyTools[call.ToolName] && call.CallerRole != "system" {
		return false, fmt.Sprintf("tool %q requires system role, caller is %q", call.ToolName, call.CallerRole)
	}
	return true, ""
}
