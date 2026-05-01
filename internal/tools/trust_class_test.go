package tools

import (
	"strings"
	"testing"
)

func TestTrustClass_ExecuteAllowed(t *testing.T) {
	exec := NewTrustClassExecutor()
	exec.Register(TrustClassTool{
		Name:           "test_tool",
		AllowedClasses: []TrustClass{TrustClassOperator, TrustClassSystem},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			return "ok", nil
		},
	})

	result, err := exec.Execute("test_tool", TrustClassOperator, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected ok, got %v", result)
	}
}

func TestTrustClass_ExecuteDenied(t *testing.T) {
	exec := NewTrustClassExecutor()
	exec.Register(TrustClassTool{
		Name:           "operator_tool",
	})

	_, err := exec.Execute("operator_tool", TrustClassGateway, nil)
	if err == nil {
		t.Fatal("expected error for disallowed trust class")
	}
	if !strings.Contains(err.Error(), "trust_class_denied") {
		t.Fatalf("expected trust_class_denied error, got %v", err)
	}
	if !strings.Contains(err.Error(), string(TrustClassGateway)) {
		t.Fatalf("expected error to contain gateway, got %v", err)
	}
}

func TestTrustClass_ListToolsFiltersByTrustClass(t *testing.T) {
	exec := NewTrustClassExecutor()
	exec.Register(TrustClassTool{
		Name:           "operator_only",
		AllowedClasses: []TrustClass{TrustClassOperator},
	})
	exec.Register(TrustClassTool{
		Name:           "gateway_safe",
		AllowedClasses: []TrustClass{TrustClassOperator, TrustClassGateway},
	})
	exec.Register(TrustClassTool{
		Name:           "universal",
		AllowedClasses: []TrustClass{TrustClassOperator, TrustClassGateway, TrustClassChildAgent, TrustClassSystem},
	})

	operatorTools := exec.ListTools(TrustClassOperator)
	if len(operatorTools) != 3 {
		t.Fatalf("expected 3 operator tools, got %d", len(operatorTools))
	}

	gatewayTools := exec.ListTools(TrustClassGateway)
	if len(gatewayTools) != 2 {
		t.Fatalf("expected 2 gateway tools, got %d", len(gatewayTools))
	}

	childTools := exec.ListTools(TrustClassChildAgent)
	if len(childTools) != 1 {
		t.Fatalf("expected 1 child-agent tool, got %d", len(childTools))
	}
}

func TestTrustClass_ChildAgentCannotSeeOperatorLocal(t *testing.T) {
	exec := NewTrustClassExecutor()
	exec.Register(TrustClassTool{
		Name:           "config_edit",
		AllowedClasses: []TrustClass{TrustClassOperator},
	})

	childTools := exec.ListTools(TrustClassChildAgent)
	for _, name := range childTools {
		if name == "config_edit" {
			t.Fatal("child-agent should not see operator-local tools")
		}
	}

	_, err := exec.Execute("config_edit", TrustClassChildAgent, nil)
	if err == nil || !strings.Contains(err.Error(), "trust_class_denied") {
		t.Fatal("child-agent should be denied access to operator-local tools")
	}
}

func TestTrustClass_SystemCanAccessAll(t *testing.T) {
	exec := NewTrustClassExecutor()
	exec.Register(TrustClassTool{
		Name:           "system_tool",
		Handler: func(args map[string]interface{}) (interface{}, error) { return "ok", nil },
		AllowedClasses: []TrustClass{TrustClassSystem},
	})
	exec.Register(TrustClassTool{
		Name:           "operator_tool",
		Handler: func(args map[string]interface{}) (interface{}, error) { return "ok", nil },
		AllowedClasses: []TrustClass{TrustClassOperator},
	})

	_, err := exec.Execute("system_tool", TrustClassSystem, nil)
	if err != nil {
		t.Fatalf("system should access system tools: %v", err)
	}

	_, err = exec.Execute("operator_tool", TrustClassSystem, nil)
	if err == nil {
		t.Fatal("system should not access operator-only tools unless explicitly allowed")
	}
}
