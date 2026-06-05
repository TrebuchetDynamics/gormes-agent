package gateway

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestAgentRouterCompatibilityWrapper(t *testing.T) {
	router := NewAgentRouter(config.AgentsCfg{List: []config.AgentCfg{{ID: "main", Default: true}}}, nil)
	decision := router.Resolve(AgentRouteRequest{Channel: "telegram", MainKey: "telegram:42"})
	if decision.AgentID != "main" || decision.BindingTier != AgentBindingTierDefault {
		t.Fatalf("decision = %+v, want default main route", decision)
	}
	if got := decision.SessionKey(); got != "agent:main:telegram:42" {
		t.Fatalf("SessionKey = %q, want agent-prefixed key", got)
	}
}
