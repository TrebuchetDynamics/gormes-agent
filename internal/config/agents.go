package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const DefaultAgentID = "main"

var agentIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

type AgentsCfg struct {
	Defaults AgentDefaultsCfg `toml:"defaults" yaml:"defaults"`
	List     []AgentCfg       `toml:"list" yaml:"list"`
}

type AgentDefaultsCfg struct {
	Workspace string `toml:"workspace" yaml:"workspace"`
	// Workspaces is the Gormes-owned per-profile list of additional workspace
	// directories the agent may access (set via `gormes setup profiles`).
	// Workspace stays the single primary path for backward compatibility;
	// Workspaces is the multi-workspace list. TOML array, round-tripped.
	Workspaces []string `toml:"workspaces" yaml:"workspaces"`
	// Channels is the Gormes-owned per-profile list of messaging channels the
	// profile uses (telegram/whatsapp/discord/slack), set via
	// `gormes setup profiles`. It records WHICH channels the profile uses —
	// distinct from [[bindings]] routing and per-channel credential cfg. TOML
	// array, round-tripped (symmetric with Workspaces).
	Channels []string `toml:"channels" yaml:"channels"`
	AgentDir string   `toml:"agent_dir" yaml:"agent_dir"`
	Skills   []string `toml:"skills" yaml:"skills"`
}

type AgentCfg struct {
	ID        string            `toml:"id" yaml:"id"`
	Name      string            `toml:"name" yaml:"name"`
	Workspace string            `toml:"workspace" yaml:"workspace"`
	AgentDir  string            `toml:"agent_dir" yaml:"agent_dir"`
	Default   bool              `toml:"default" yaml:"default"`
	Model     string            `toml:"model" yaml:"model"`
	Skills    []string          `toml:"skills" yaml:"skills"`
	Sandbox   AgentSandboxCfg   `toml:"sandbox" yaml:"sandbox"`
	Tools     AgentToolPolicy   `toml:"tools" yaml:"tools"`
	GroupChat AgentGroupChatCfg `toml:"group_chat" yaml:"group_chat"`
}

type AgentSandboxCfg struct {
	Mode   string                `toml:"mode" yaml:"mode"`
	Scope  string                `toml:"scope" yaml:"scope"`
	Docker AgentSandboxDockerCfg `toml:"docker" yaml:"docker"`
}

type AgentSandboxDockerCfg struct {
	SetupCommand string `toml:"setup_command" yaml:"setup_command"`
}

type AgentToolPolicy struct {
	Allow []string `toml:"allow" yaml:"allow"`
	Deny  []string `toml:"deny" yaml:"deny"`
}

type AgentGroupChatCfg struct {
	MentionPatterns []string `toml:"mention_patterns" yaml:"mention_patterns"`
}

type AgentBindingCfg struct {
	AgentID string               `toml:"agent_id" yaml:"agent_id"`
	Match   AgentBindingMatchCfg `toml:"match" yaml:"match"`
}

type AgentBindingMatchCfg struct {
	Channel    string            `toml:"channel" yaml:"channel"`
	AccountID  string            `toml:"account_id" yaml:"account_id"`
	Peer       AgentPeerMatchCfg `toml:"peer" yaml:"peer"`
	ParentPeer AgentPeerMatchCfg `toml:"parent_peer" yaml:"parent_peer"`
	GuildID    string            `toml:"guild_id" yaml:"guild_id"`
	TeamID     string            `toml:"team_id" yaml:"team_id"`
	Roles      []string          `toml:"roles" yaml:"roles"`
}

type AgentPeerMatchCfg struct {
	Kind string `toml:"kind" yaml:"kind"`
	ID   string `toml:"id" yaml:"id"`
}

func (a AgentsCfg) DefaultAgentID() string {
	for _, agent := range a.List {
		if agent.Default {
			if id := strings.TrimSpace(agent.ID); id != "" {
				return id
			}
		}
	}
	if len(a.List) > 0 {
		if id := strings.TrimSpace(a.List[0].ID); id != "" {
			return id
		}
	}
	return DefaultAgentID
}

func (a AgentsCfg) AgentByID(id string) (AgentCfg, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, agent := range a.List {
		if strings.ToLower(strings.TrimSpace(agent.ID)) == id {
			return agent, true
		}
	}
	return AgentCfg{}, false
}

func defaultAgentsCfg(home string) AgentsCfg {
	return AgentsCfg{
		List: []AgentCfg{{
			ID:        DefaultAgentID,
			Name:      "Main",
			Workspace: filepath.Join(home, "workspace"),
			AgentDir:  filepath.Join(home, "agents", DefaultAgentID, "agent"),
			Default:   true,
		}},
	}
}

func normalizeAgentsConfig(home string, agents *AgentsCfg, bindings []AgentBindingCfg) error {
	if len(agents.List) == 0 {
		defaults := agents.Defaults
		*agents = defaultAgentsCfg(home)
		agents.Defaults = defaults
		if workspace := strings.TrimSpace(defaults.Workspace); workspace != "" {
			agents.List[0].Workspace = workspace
		}
		if agentDir := strings.TrimSpace(defaults.AgentDir); agentDir != "" {
			agents.List[0].AgentDir = agentDir
		}
		agents.List[0].Skills = cleanStringSlice(defaults.Skills)
		return nil
	}

	seen := map[string]struct{}{}
	defaults := 0
	for i := range agents.List {
		agent := &agents.List[i]
		agent.ID = strings.ToLower(strings.TrimSpace(agent.ID))
		agent.Name = strings.TrimSpace(agent.Name)
		agent.Workspace = strings.TrimSpace(agent.Workspace)
		agent.AgentDir = strings.TrimSpace(agent.AgentDir)
		agent.Model = strings.TrimSpace(agent.Model)
		agent.Sandbox.Mode = strings.TrimSpace(agent.Sandbox.Mode)
		agent.Sandbox.Scope = strings.TrimSpace(agent.Sandbox.Scope)
		agent.Sandbox.Docker.SetupCommand = strings.TrimSpace(agent.Sandbox.Docker.SetupCommand)
		agent.Skills = cleanStringSlice(agent.Skills)
		agent.Tools.Allow = cleanStringSlice(agent.Tools.Allow)
		agent.Tools.Deny = cleanStringSlice(agent.Tools.Deny)
		agent.GroupChat.MentionPatterns = cleanStringSlice(agent.GroupChat.MentionPatterns)
		if agent.ID == "" {
			return fmt.Errorf("config: agents.list[%d].id is required", i)
		}
		if !agentIDPattern.MatchString(agent.ID) {
			return fmt.Errorf("config: agents.list[%d].id %q is invalid", i, agent.ID)
		}
		if _, ok := seen[agent.ID]; ok {
			return fmt.Errorf("config: duplicate agent id %q", agent.ID)
		}
		seen[agent.ID] = struct{}{}
		if agent.Default {
			defaults++
		}
		if agent.Name == "" {
			agent.Name = agent.ID
		}
		if agent.Workspace == "" {
			if agent.ID == DefaultAgentID {
				agent.Workspace = filepath.Join(home, "workspace")
			} else {
				agent.Workspace = filepath.Join(home, "workspace-"+agent.ID)
			}
		}
		if agent.AgentDir == "" {
			agent.AgentDir = filepath.Join(home, "agents", agent.ID, "agent")
		}
		if len(agent.Skills) == 0 && len(agents.Defaults.Skills) > 0 {
			agent.Skills = cleanStringSlice(agents.Defaults.Skills)
		}
	}
	if defaults > 1 {
		return fmt.Errorf("config: only one agent can be default")
	}
	if defaults == 0 {
		agents.List[0].Default = true
	}

	for i, binding := range bindings {
		agentID := strings.ToLower(strings.TrimSpace(binding.AgentID))
		if agentID == "" {
			return fmt.Errorf("config: bindings[%d].agent_id is required", i)
		}
		if _, ok := seen[agentID]; !ok {
			return fmt.Errorf("config: bindings[%d].agent_id %q does not match a configured agent", i, agentID)
		}
		if strings.TrimSpace(binding.Match.Channel) == "" {
			return fmt.Errorf("config: bindings[%d].match.channel is required", i)
		}
	}
	return nil
}

func normalizeAgentBindings(bindings []AgentBindingCfg) []AgentBindingCfg {
	out := make([]AgentBindingCfg, 0, len(bindings))
	for _, binding := range bindings {
		binding.AgentID = strings.ToLower(strings.TrimSpace(binding.AgentID))
		binding.Match.Channel = strings.ToLower(strings.TrimSpace(binding.Match.Channel))
		binding.Match.AccountID = strings.TrimSpace(binding.Match.AccountID)
		binding.Match.Peer = normalizeAgentPeerMatch(binding.Match.Peer)
		binding.Match.ParentPeer = normalizeAgentPeerMatch(binding.Match.ParentPeer)
		binding.Match.GuildID = strings.TrimSpace(binding.Match.GuildID)
		binding.Match.TeamID = strings.TrimSpace(binding.Match.TeamID)
		binding.Match.Roles = cleanStringSlice(binding.Match.Roles)
		sort.Strings(binding.Match.Roles)
		out = append(out, binding)
	}
	return out
}

func normalizeAgentPeerMatch(peer AgentPeerMatchCfg) AgentPeerMatchCfg {
	return AgentPeerMatchCfg{
		Kind: strings.ToLower(strings.TrimSpace(peer.Kind)),
		ID:   strings.TrimSpace(peer.ID),
	}
}

func cleanStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
