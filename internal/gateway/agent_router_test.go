package gateway

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestAgentRouter_MostSpecificBindingWins(t *testing.T) {
	router := NewAgentRouter(config.AgentsCfg{
		List: []config.AgentCfg{
			{ID: "chat", Workspace: "/agents/chat/workspace", AgentDir: "/agents/chat/agent", Default: true},
			{ID: "opus", Workspace: "/agents/opus/workspace", AgentDir: "/agents/opus/agent"},
			{ID: "family", Workspace: "/agents/family/workspace", AgentDir: "/agents/family/agent"},
		},
	}, []config.AgentBindingCfg{
		{AgentID: "family", Match: config.AgentBindingMatchCfg{Channel: "whatsapp", Peer: config.AgentPeerMatchCfg{Kind: "group", ID: "120363999@g.us"}}},
		{AgentID: "opus", Match: config.AgentBindingMatchCfg{Channel: "whatsapp", Peer: config.AgentPeerMatchCfg{Kind: "direct", ID: "+15551234567"}}},
		{AgentID: "chat", Match: config.AgentBindingMatchCfg{Channel: "whatsapp"}},
	})

	group := router.Resolve(AgentRouteRequest{Channel: "whatsapp", PeerKind: "group", PeerID: "120363999@g.us", MainKey: "whatsapp:120363999@g.us"})
	if group.AgentID != "family" || group.Workspace != "/agents/family/workspace" {
		t.Fatalf("group route = %+v, want family", group)
	}
	if got := group.SessionKey(); got != "agent:family:whatsapp:120363999@g.us" {
		t.Fatalf("group SessionKey = %q, want agent-prefixed key", got)
	}

	dm := router.Resolve(AgentRouteRequest{Channel: "whatsapp", PeerKind: "direct", PeerID: "+15551234567", MainKey: "whatsapp:+15551234567"})
	if dm.AgentID != "opus" {
		t.Fatalf("dm route = %+v, want opus", dm)
	}

	fallback := router.Resolve(AgentRouteRequest{Channel: "whatsapp", PeerKind: "direct", PeerID: "+15550000000", MainKey: "whatsapp:+15550000000"})
	if fallback.AgentID != "chat" || fallback.BindingTier != AgentBindingTierAccount {
		t.Fatalf("fallback route = %+v, want chat channel fallback", fallback)
	}
}

func TestAgentRouter_DefaultAgentWhenNoBindingMatches(t *testing.T) {
	router := NewAgentRouter(config.AgentsCfg{
		List: []config.AgentCfg{
			{ID: "chat", Workspace: "/agents/chat/workspace", AgentDir: "/agents/chat/agent"},
			{ID: "work", Workspace: "/agents/work/workspace", AgentDir: "/agents/work/agent", Default: true},
		},
	}, nil)

	decision := router.Resolve(AgentRouteRequest{Channel: "telegram", MainKey: "telegram:42"})
	if decision.AgentID != "work" || decision.BindingTier != AgentBindingTierDefault {
		t.Fatalf("decision = %+v, want default work route", decision)
	}
	if got := decision.SessionKey(); got != "agent:work:telegram:42" {
		t.Fatalf("SessionKey = %q, want agent-prefixed key", got)
	}
}

func TestAgentRouter_AccountScopeAndTieBreak(t *testing.T) {
	router := NewAgentRouter(config.AgentsCfg{
		List: []config.AgentCfg{
			{ID: "main", Workspace: "/agents/main/workspace", AgentDir: "/agents/main/agent", Default: true},
			{ID: "alerts", Workspace: "/agents/alerts/workspace", AgentDir: "/agents/alerts/agent"},
			{ID: "catchall", Workspace: "/agents/catchall/workspace", AgentDir: "/agents/catchall/agent"},
		},
	}, []config.AgentBindingCfg{
		{AgentID: "alerts", Match: config.AgentBindingMatchCfg{Channel: "telegram", AccountID: "alerts"}},
		{AgentID: "catchall", Match: config.AgentBindingMatchCfg{Channel: "telegram", AccountID: "*"}},
		{AgentID: "main", Match: config.AgentBindingMatchCfg{Channel: "telegram"}},
	})

	defaultAccount := router.Resolve(AgentRouteRequest{Channel: "telegram"})
	if defaultAccount.AgentID != "main" || defaultAccount.BindingTier != AgentBindingTierAccount {
		t.Fatalf("default account route = %+v, want channel binding for main", defaultAccount)
	}
	if got := defaultAccount.SessionKey(); got != "agent:main:main" {
		t.Fatalf("default account SessionKey = %q, want main fallback key", got)
	}

	alerts := router.Resolve(AgentRouteRequest{Channel: "telegram", AccountID: "alerts", MainKey: "telegram:alerts"})
	if alerts.AgentID != "alerts" || alerts.BindingTier != AgentBindingTierAccount {
		t.Fatalf("alerts account route = %+v, want explicit account binding", alerts)
	}

	other := router.Resolve(AgentRouteRequest{Channel: "telegram", AccountID: "other", MainKey: "telegram:other"})
	if other.AgentID != "catchall" || other.BindingTier != AgentBindingTierChannel {
		t.Fatalf("other account route = %+v, want account wildcard fallback", other)
	}
}
