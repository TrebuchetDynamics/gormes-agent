package kernel

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// TrustClass names the caller boundary used by noninteractive tool-safety
// policy. Keep this small and explicit; gateway/child-agent callers must not
// inherit operator-local approval behavior by accident.
type TrustClass string

const (
	TrustClassOperator   TrustClass = "operator"
	TrustClassSystem     TrustClass = "system"
	TrustClassGateway    TrustClass = "gateway"
	TrustClassChildAgent TrustClass = "child-agent"
)

// ToolSafetyPolicy can intercept a model-requested tool call before registry
// lookup or execution. A denied decision is returned to the model as a normal
// role=tool result so the turn can continue without blocking for UI approval.
type ToolSafetyPolicy interface {
	DecideToolCall(call llm.ToolCall) ToolSafetyDecision
}

type ToolSafetyDecision struct {
	Allow   bool
	Status  string
	Content json.RawMessage
	Err     error
}

type OneshotToolSafetyOptions struct {
	TrustClass     TrustClass
	ApprovalBypass bool
}

type OneshotToolSafetyPolicy struct {
	trustClass     TrustClass
	approvalBypass bool
}

type compositeToolSafetyPolicy struct {
	policies []ToolSafetyPolicy
}

type AgentToolSafetyOptions struct {
	AgentID string
	Allow   []string
	Deny    []string
}

type AgentToolSafetyPolicy struct {
	agentID string
	allow   map[string]struct{}
	deny    map[string]struct{}
}

func ComposeToolSafetyPolicies(policies ...ToolSafetyPolicy) ToolSafetyPolicy {
	clean := make([]ToolSafetyPolicy, 0, len(policies))
	for _, policy := range policies {
		if policy != nil {
			clean = append(clean, policy)
		}
	}
	switch len(clean) {
	case 0:
		return nil
	case 1:
		return clean[0]
	default:
		return compositeToolSafetyPolicy{policies: clean}
	}
}

func NewAgentToolSafetyPolicy(opts AgentToolSafetyOptions) ToolSafetyPolicy {
	allow := toolNameSet(opts.Allow)
	deny := toolNameSet(opts.Deny)
	if len(allow) == 0 && len(deny) == 0 {
		return nil
	}
	agentID := strings.TrimSpace(opts.AgentID)
	if agentID == "" {
		agentID = "main"
	}
	return AgentToolSafetyPolicy{
		agentID: agentID,
		allow:   allow,
		deny:    deny,
	}
}

func (p compositeToolSafetyPolicy) DecideToolCall(call llm.ToolCall) ToolSafetyDecision {
	for _, policy := range p.policies {
		if policy == nil {
			continue
		}
		decision := policy.DecideToolCall(call)
		if !decision.Allow {
			return decision
		}
	}
	return ToolSafetyDecision{Allow: true}
}

func (p AgentToolSafetyPolicy) DecideToolCall(call llm.ToolCall) ToolSafetyDecision {
	name := normalizeToolPolicyName(call.Name)
	if name == "" {
		return ToolSafetyDecision{Allow: true}
	}
	if _, denied := p.deny[name]; denied {
		return p.denyDecision(call, "tool denied by agent policy")
	}
	if len(p.allow) > 0 {
		if _, allowed := p.allow[name]; !allowed {
			return p.denyDecision(call, "tool not in agent allowlist")
		}
	}
	return ToolSafetyDecision{Allow: true}
}

func (p AgentToolSafetyPolicy) denyDecision(call llm.ToolCall, reason string) ToolSafetyDecision {
	payload, _ := json.Marshal(map[string]any{
		"status":   "agent_tool_policy_denied",
		"agent_id": p.agentID,
		"tool":     strings.TrimSpace(call.Name),
		"reason":   reason,
	})
	return ToolSafetyDecision{
		Allow:   false,
		Status:  "agent_tool_policy_denied",
		Content: payload,
		Err:     errors.New("agent tool policy denied " + strings.TrimSpace(call.Name)),
	}
}

func toolNameSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		if name := normalizeToolPolicyName(value); name != "" {
			out[name] = struct{}{}
		}
	}
	return out
}

func normalizeToolPolicyName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func NewOneshotToolSafetyPolicy(opts OneshotToolSafetyOptions) (*OneshotToolSafetyPolicy, error) {
	trustClass := opts.TrustClass
	if trustClass == "" {
		trustClass = TrustClassOperator
	}
	if !validTrustClass(trustClass) {
		return nil, fmt.Errorf("kernel: unknown trust class %q", trustClass)
	}
	if opts.ApprovalBypass && trustClass != TrustClassOperator {
		return nil, fmt.Errorf("kernel: oneshot approval bypass is operator-local only, got trust class %q", trustClass)
	}
	return &OneshotToolSafetyPolicy{
		trustClass:     trustClass,
		approvalBypass: opts.ApprovalBypass,
	}, nil
}

func validTrustClass(trustClass TrustClass) bool {
	switch trustClass {
	case TrustClassOperator, TrustClassSystem, TrustClassGateway, TrustClassChildAgent:
		return true
	default:
		return false
	}
}

func (p *OneshotToolSafetyPolicy) ApprovalMode() string {
	if p != nil && p.approvalBypass {
		return "operator_local_bypass"
	}
	return "default_block"
}

func (p *OneshotToolSafetyPolicy) DecideToolCall(call llm.ToolCall) ToolSafetyDecision {
	if p == nil {
		return ToolSafetyDecision{Allow: true}
	}
	if isClarifyTool(call.Name) {
		payload := p.clarifyPayload(call)
		return ToolSafetyDecision{
			Allow:   false,
			Status:  "clarify_unavailable",
			Content: payload,
			Err:     errors.New("clarify unavailable in noninteractive oneshot mode"),
		}
	}
	if blocked, kind, command := dangerousApprovalRequest(call); blocked {
		if p.approvalBypass {
			return ToolSafetyDecision{Allow: true}
		}
		payload := p.blockedApprovalPayload(call.Name, kind, command)
		return ToolSafetyDecision{
			Allow:   false,
			Status:  "dangerous_command_blocked",
			Content: payload,
			Err:     errors.New("dangerous command or hook approval blocked by noninteractive oneshot policy"),
		}
	}
	return ToolSafetyDecision{Allow: true}
}

func isClarifyTool(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "clarify")
}

func (p *OneshotToolSafetyPolicy) clarifyPayload(call llm.ToolCall) json.RawMessage {
	args := decodeObject(call.Arguments)
	question, _ := args["question"].(string)
	choices := stringSlice(args["choices"])
	assumption := "oneshot mode has no interactive user available; pick the best option using existing context and continue"
	payload := map[string]any{
		"status":         "clarify_unavailable",
		"noninteractive": true,
		"trust_class":    string(p.trustClass),
		"approval_mode":  p.ApprovalMode(),
		"assumption":     assumption,
	}
	if strings.TrimSpace(question) != "" {
		payload["question"] = question
	}
	if len(choices) > 0 {
		payload["choices"] = choices
	}
	out, _ := json.Marshal(payload)
	return out
}

func (p *OneshotToolSafetyPolicy) blockedApprovalPayload(toolName, kind, command string) json.RawMessage {
	payload := map[string]any{
		"status":         "dangerous_command_blocked",
		"noninteractive": true,
		"trust_class":    string(p.trustClass),
		"approval_mode":  p.ApprovalMode(),
		"tool":           toolName,
		"approval_kind":  kind,
		"reason":         "oneshot mode does not request interactive approval or auto-approve dangerous commands/hooks by default",
	}
	if strings.TrimSpace(command) != "" {
		payload["command"] = command
	}
	out, _ := json.Marshal(payload)
	return out
}

func dangerousApprovalRequest(call llm.ToolCall) (bool, string, string) {
	name := strings.ToLower(strings.TrimSpace(call.Name))
	if name == "" {
		return false, "", ""
	}
	if name == "hook_approval" || name == "shell_hook" || strings.HasSuffix(name, "_hook") {
		return true, "hook_approval", extractCommand(call.Arguments)
	}
	switch name {
	case "execute_code":
		args := decodeObject(call.Arguments)
		language, _ := args["language"].(string)
		code, _ := args["code"].(string)
		if isShellLanguage(language) && looksDangerousCommand(code) {
			return true, "dangerous_command", code
		}
	case "terminal", "terminal_run", "shell", "run_shell", "execute_shell":
		command := extractCommand(call.Arguments)
		if looksDangerousCommand(command) {
			return true, "dangerous_command", command
		}
	}
	return false, "", ""
}

func decodeObject(raw json.RawMessage) map[string]any {
	var out map[string]any
	if len(raw) == 0 {
		return map[string]any{}
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func stringSlice(v any) []string {
	switch typed := v.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func isShellLanguage(language string) bool {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "sh", "shell", "bash", "zsh":
		return true
	default:
		return false
	}
}

func extractCommand(raw json.RawMessage) string {
	args := decodeObject(raw)
	for _, key := range []string{"command", "cmd", "code", "script"} {
		if value, ok := args[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func looksDangerousCommand(command string) bool {
	lower := strings.ToLower(strings.TrimSpace(command))
	if lower == "" {
		return false
	}
	dangerousFragments := []string{
		"rm -rf",
		"rm -fr",
		"mkfs",
		":(){",
		"dd if=",
		"chmod -r 777 /",
		"chown -r",
		"curl ",
		"wget ",
		"sudo ",
	}
	for _, fragment := range dangerousFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}
