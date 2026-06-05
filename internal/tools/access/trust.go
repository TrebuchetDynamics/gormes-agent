package access

import "fmt"

// TrustClass is the trust label attached to a caller (channel, system path,
// or child agent). The MCP host boundary uses these labels to decide whether
// a tool declaration may be exposed or invoked. The string values match the
// progress.json trust_class vocabulary used elsewhere in Gormes (operator,
// gateway, child-agent, system).
type TrustClass string

// Trust class constants. Keep these in sync with the progress.json
// trust_class enum and with subagent.TrustClass; we re-declare the values
// here so tools access policy does not depend on subagent.
const (
	TrustClassOperator   TrustClass = "operator"
	TrustClassGateway    TrustClass = "gateway"
	TrustClassChildAgent TrustClass = "child-agent"
	TrustClassSystem     TrustClass = "system"
)

type TrustClassTool struct {
	Name           string
	AllowedClasses []TrustClass
	Handler        func(args map[string]interface{}) (interface{}, error)
}

type TrustClassExecutor struct {
	tools map[string]TrustClassTool
}

func NewTrustClassExecutor() *TrustClassExecutor {
	return &TrustClassExecutor{tools: make(map[string]TrustClassTool)}
}

func (e *TrustClassExecutor) Register(tool TrustClassTool) {
	e.tools[tool.Name] = tool
}

func (e *TrustClassExecutor) Execute(name string, caller TrustClass, args map[string]interface{}) (interface{}, error) {
	tool, ok := e.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool_not_found: %s", name)
	}

	if !isTrustClassAllowed(caller, tool.AllowedClasses) {
		return nil, fmt.Errorf("trust_class_denied: tool=%s requested=%s allowed=%v", name, caller, tool.AllowedClasses)
	}

	return tool.Handler(args)
}

func (e *TrustClassExecutor) ListTools(caller TrustClass) []string {
	var result []string
	for name, tool := range e.tools {
		if isTrustClassAllowed(caller, tool.AllowedClasses) {
			result = append(result, name)
		}
	}
	return result
}

func isTrustClassAllowed(caller TrustClass, allowed []TrustClass) bool {
	for _, a := range allowed {
		if a == caller {
			return true
		}
	}
	return false
}
