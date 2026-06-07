package routing

import (
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

type AgentBindingTier string

const (
	AgentBindingTierPeer       AgentBindingTier = "peer"
	AgentBindingTierParentPeer AgentBindingTier = "parent_peer"
	AgentBindingTierGuildRoles AgentBindingTier = "guild_roles"
	AgentBindingTierGuild      AgentBindingTier = "guild"
	AgentBindingTierTeam       AgentBindingTier = "team"
	AgentBindingTierAccount    AgentBindingTier = "account"
	AgentBindingTierChannel    AgentBindingTier = "channel"
	AgentBindingTierDefault    AgentBindingTier = "default"
	agentRouteDefaultAccountID                  = "default"
)

type AgentRouteRequest struct {
	Channel        string
	AccountID      string
	PeerKind       string
	PeerID         string
	ParentPeerKind string
	ParentPeerID   string
	GuildID        string
	TeamID         string
	Roles          []string
	MainKey        string
}

type AgentRouteDecision struct {
	AgentID      string
	Name         string
	Workspace    string
	AgentDir     string
	Model        string
	Skills       []string
	Tools        config.AgentToolPolicy
	BindingIndex int
	BindingTier  AgentBindingTier
	MainKey      string
}

func (d AgentRouteDecision) SessionKey() string {
	mainKey := strings.TrimSpace(d.MainKey)
	agentID := strings.TrimSpace(d.AgentID)
	if agentID == "" {
		agentID = config.DefaultAgentID
	}
	if mainKey == "" {
		mainKey = "main"
	}
	return "agent:" + agentID + ":" + mainKey
}

type AgentRouter struct {
	agents   config.AgentsCfg
	bindings []config.AgentBindingCfg
}

func NewAgentRouter(agents config.AgentsCfg, bindings []config.AgentBindingCfg) AgentRouter {
	return AgentRouter{agents: cloneAgentsCfg(agents), bindings: cloneAgentBindings(bindings)}
}

func cloneAgentsCfg(agents config.AgentsCfg) config.AgentsCfg {
	agents.List = slices.Clone(agents.List)
	for i := range agents.List {
		agents.List[i].Skills = slices.Clone(agents.List[i].Skills)
		agents.List[i].Tools = cloneAgentToolPolicy(agents.List[i].Tools)
	}
	return agents
}

func cloneAgentBindings(bindings []config.AgentBindingCfg) []config.AgentBindingCfg {
	out := slices.Clone(bindings)
	for i := range out {
		out[i].Match.Roles = slices.Clone(out[i].Match.Roles)
	}
	return out
}

func (r AgentRouter) Resolve(req AgentRouteRequest) AgentRouteDecision {
	req = normalizeAgentRouteRequest(req)
	bestIndex := -1
	bestTier := AgentBindingTierDefault
	bestRank := -1
	for i, binding := range r.bindings {
		tier, ok := matchAgentBinding(binding.Match, req)
		if !ok {
			continue
		}
		rank := agentBindingTierRank(tier)
		if rank > bestRank {
			bestIndex = i
			bestTier = tier
			bestRank = rank
		}
	}
	agentID := r.agents.DefaultAgentID()
	if bestIndex >= 0 {
		agentID = strings.TrimSpace(r.bindings[bestIndex].AgentID)
	}
	agent, ok := r.agents.AgentByID(agentID)
	if !ok {
		agent = config.AgentCfg{ID: agentID}
	}
	return AgentRouteDecision{
		AgentID:      textvalue.FirstNonEmptyTrimmed(agent.ID, agentID, config.DefaultAgentID),
		Name:         agent.Name,
		Workspace:    agent.Workspace,
		AgentDir:     agent.AgentDir,
		Model:        agent.Model,
		Skills:       slices.Clone(agent.Skills),
		Tools:        cloneAgentToolPolicy(agent.Tools),
		BindingIndex: bestIndex,
		BindingTier:  bestTier,
		MainKey:      req.MainKey,
	}
}

func cloneAgentToolPolicy(policy config.AgentToolPolicy) config.AgentToolPolicy {
	return config.AgentToolPolicy{
		Allow: slices.Clone(policy.Allow),
		Deny:  slices.Clone(policy.Deny),
	}
}

func normalizeAgentRouteRequest(req AgentRouteRequest) AgentRouteRequest {
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.AccountID = strings.TrimSpace(req.AccountID)
	if req.AccountID == "" {
		req.AccountID = agentRouteDefaultAccountID
	}
	req.PeerKind = strings.ToLower(strings.TrimSpace(req.PeerKind))
	req.PeerID = strings.TrimSpace(req.PeerID)
	req.ParentPeerKind = strings.ToLower(strings.TrimSpace(req.ParentPeerKind))
	req.ParentPeerID = strings.TrimSpace(req.ParentPeerID)
	req.GuildID = strings.TrimSpace(req.GuildID)
	req.TeamID = strings.TrimSpace(req.TeamID)
	req.MainKey = strings.TrimSpace(req.MainKey)
	req.Roles = cleanRouteStrings(req.Roles)
	return req
}

func matchAgentBinding(match config.AgentBindingMatchCfg, req AgentRouteRequest) (AgentBindingTier, bool) {
	match.Channel = strings.ToLower(strings.TrimSpace(match.Channel))
	match.AccountID = strings.TrimSpace(match.AccountID)
	match.Peer = normalizeRoutePeer(match.Peer)
	match.ParentPeer = normalizeRoutePeer(match.ParentPeer)
	match.GuildID = strings.TrimSpace(match.GuildID)
	match.TeamID = strings.TrimSpace(match.TeamID)
	match.Roles = cleanRouteStrings(match.Roles)
	if match.Channel != "" && match.Channel != req.Channel {
		return "", false
	}
	if match.AccountID == "" {
		if req.AccountID != agentRouteDefaultAccountID {
			return "", false
		}
	} else if match.AccountID != "*" && match.AccountID != req.AccountID {
		return "", false
	}
	if match.Peer.ID != "" && !sameRoutePeer(match.Peer.Kind, match.Peer.ID, req.PeerKind, req.PeerID) {
		return "", false
	}
	if match.ParentPeer.ID != "" && !sameRoutePeer(match.ParentPeer.Kind, match.ParentPeer.ID, req.ParentPeerKind, req.ParentPeerID) {
		return "", false
	}
	if match.GuildID != "" && match.GuildID != req.GuildID {
		return "", false
	}
	if match.TeamID != "" && match.TeamID != req.TeamID {
		return "", false
	}
	if len(match.Roles) > 0 && !rolesContainAll(req.Roles, match.Roles) {
		return "", false
	}
	return mostSpecificAgentTier(match), true
}

func mostSpecificAgentTier(match config.AgentBindingMatchCfg) AgentBindingTier {
	accountID := strings.TrimSpace(match.AccountID)
	channel := strings.TrimSpace(match.Channel)
	switch {
	case strings.TrimSpace(match.Peer.ID) != "":
		return AgentBindingTierPeer
	case strings.TrimSpace(match.ParentPeer.ID) != "":
		return AgentBindingTierParentPeer
	case strings.TrimSpace(match.GuildID) != "" && len(match.Roles) > 0:
		return AgentBindingTierGuildRoles
	case strings.TrimSpace(match.GuildID) != "":
		return AgentBindingTierGuild
	case strings.TrimSpace(match.TeamID) != "":
		return AgentBindingTierTeam
	case channel != "" && accountID != "*":
		return AgentBindingTierAccount
	case channel != "":
		return AgentBindingTierChannel
	default:
		return AgentBindingTierDefault
	}
}

func agentBindingTierRank(tier AgentBindingTier) int {
	switch tier {
	case AgentBindingTierPeer:
		return 7
	case AgentBindingTierParentPeer:
		return 6
	case AgentBindingTierGuildRoles:
		return 5
	case AgentBindingTierGuild:
		return 4
	case AgentBindingTierTeam:
		return 3
	case AgentBindingTierAccount:
		return 2
	case AgentBindingTierChannel:
		return 1
	default:
		return 0
	}
}

func normalizeRoutePeer(peer config.AgentPeerMatchCfg) config.AgentPeerMatchCfg {
	return config.AgentPeerMatchCfg{
		Kind: strings.ToLower(strings.TrimSpace(peer.Kind)),
		ID:   strings.TrimSpace(peer.ID),
	}
}

func sameRoutePeer(matchKind, matchID, gotKind, gotID string) bool {
	if strings.TrimSpace(matchID) != strings.TrimSpace(gotID) {
		return false
	}
	if matchKind == "" {
		return true
	}
	return strings.EqualFold(matchKind, gotKind)
}

func rolesContainAll(got, want []string) bool {
	if len(want) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(got))
	for _, role := range got {
		set[role] = struct{}{}
	}
	for _, role := range want {
		if _, ok := set[role]; !ok {
			return false
		}
	}
	return true
}

func cleanRouteStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}
