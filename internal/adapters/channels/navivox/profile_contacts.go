package navivox

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	cliprofile "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/profile"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const (
	navivoxDefaultServerID    = "navivox-gateway"
	navivoxDefaultServerLabel = "Gormes Gateway"

	ProfileContactHealthOnline    = "online"
	ProfileContactHealthOffline   = "offline"
	ProfileContactHealthNeedsAuth = "needs_auth"
	ProfileContactHealthWarning   = "warning"

	ProfileContactTurnIdle   = "idle"
	ProfileContactTurnActive = "active"
)

type ProfileContact struct {
	ServerID              string   `json:"server_id"`
	ProfileID             string   `json:"profile_id"`
	DisplayName           string   `json:"display_name"`
	ServerLabel           string   `json:"server_label"`
	AvatarSeed            string   `json:"avatar_seed"`
	LatestPreview         string   `json:"latest_preview"`
	LatestPreviewKind     string   `json:"latest_preview_kind"`
	LatestPreviewAt       string   `json:"latest_preview_at,omitempty"`
	Health                string   `json:"health"`
	WorkspaceRootCount    int      `json:"workspace_root_count"`
	WorkspaceRootsOK      bool     `json:"workspace_roots_ok"`
	WorkspaceRootsWarning int      `json:"workspace_roots_warning"`
	WorkspaceRootsError   int      `json:"workspace_roots_error"`
	AttentionBadges       []string `json:"attention_badges"`
	MicAvailable          bool     `json:"mic_available"`
	ActiveTurnState       string   `json:"active_turn_state"`
}

type profileContactSnapshot struct {
	Contacts []ProfileContact `json:"contacts"`
}

type profileConfigReadModel struct {
	Hermes struct {
		Provider string `toml:"provider" yaml:"provider"`
		Model    string `toml:"model" yaml:"model"`
	} `toml:"hermes" yaml:"hermes"`
	Agents struct {
		Defaults struct {
			Workspace  string   `toml:"workspace" yaml:"workspace"`
			Workspaces []string `toml:"workspaces" yaml:"workspaces"`
		} `toml:"defaults" yaml:"defaults"`
	} `toml:"agents" yaml:"agents"`
}

var (
	navivoxSecretAssignmentPattern = regexp.MustCompile(`(?i)\b(api[_-]?key|authorization|bearer|password|secret|token)\s*[:=]\s*("[^"]*"|'[^']*'|[^\s,;}]+)`)
	navivoxQuotedSecretPattern     = regexp.MustCompile(`(?i)"(api[_-]?key|authorization|password|secret|token)"\s*:\s*"[^"]*"`)
	navivoxPathPattern             = regexp.MustCompile(`(?:~|/[A-Za-z0-9._-]+)(?:/[A-Za-z0-9._@%+=:,;-]+)+`)
)

func (c *Channel) profileContactSnapshot(ctx context.Context) ([]ProfileContact, error) {
	loader := c.loadContacts
	if loader == nil {
		loader = c.defaultProfileContacts
	}
	contacts, err := loader(ctx)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	merged := make([]ProfileContact, 0, len(contacts)+len(c.profileContacts))
	seen := map[string]struct{}{}
	for _, contact := range contacts {
		contact = normalizeProfileContact(contact)
		key := profileContactKey(contact.ServerID, contact.ProfileID)
		if overlay, ok := c.profileContacts[key]; ok {
			contact = mergeProfileContact(contact, overlay)
		}
		merged = append(merged, contact)
		seen[key] = struct{}{}
	}
	for key, contact := range c.profileContacts {
		if _, ok := seen[key]; ok {
			continue
		}
		merged = append(merged, normalizeProfileContact(contact))
	}
	sortProfileContacts(merged)
	return merged, nil
}

func (c *Channel) defaultProfileContacts(ctx context.Context) ([]ProfileContact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	baseHome := config.GormesBaseHome()
	known := navivoxKnownProfileNames(baseHome)
	contacts := make([]ProfileContact, 0, len(known))
	for _, name := range known {
		root := ""
		if resolved, err := cliprofile.ResolveProfileRuntimeRoot(baseHome, name); err == nil {
			root = resolved
		}
		contacts = append(contacts, c.profileContactFromRoot(name, root))
	}
	sortProfileContacts(contacts)
	return contacts, nil
}

func navivoxKnownProfileNames(baseHome string) []string {
	known := []string{config.DefaultProfileID}
	seen := map[string]struct{}{config.DefaultProfileID: {}}
	addName := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		if err := cliprofile.ValidateProfileName(name); err != nil {
			return
		}
		seen[name] = struct{}{}
		known = append(known, name)
	}
	if cfg, err := navivoxLoadConfigFromBaseHome(baseHome); err == nil {
		for name := range cfg.Profiles {
			addName(name)
		}
	}
	profilesDir := filepath.Join(baseHome, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return known
	}
	for _, entry := range entries {
		if entry.IsDir() {
			addName(entry.Name())
		}
	}
	return known
}

func navivoxLoadConfigFromBaseHome(baseHome string) (config.Config, error) {
	baseHome = strings.TrimSpace(baseHome)
	if baseHome == "" || filepath.Clean(config.GormesHome()) == filepath.Clean(baseHome) {
		return config.Load(nil)
	}
	rawHome, hadHome := os.LookupEnv("GORMES_HOME")
	if err := os.Setenv("GORMES_HOME", baseHome); err != nil {
		return config.Config{}, err
	}
	defer func() {
		if hadHome {
			_ = os.Setenv("GORMES_HOME", rawHome)
		} else {
			_ = os.Unsetenv("GORMES_HOME")
		}
	}()
	return config.Load(nil)
}

func (c *Channel) profileContactFromRoot(name, root string) ProfileContact {
	contact := ProfileContact{
		ServerID:          navivoxDefaultServerID,
		ProfileID:         name,
		DisplayName:       profileDisplayName(name),
		ServerLabel:       navivoxDefaultServerLabel,
		AvatarSeed:        navivoxDefaultServerID + ":" + name,
		LatestPreview:     "Profile ready",
		LatestPreviewKind: "status",
		LatestPreviewAt:   c.now().UTC().Format(time.RFC3339),
		Health:            ProfileContactHealthOnline,
		WorkspaceRootsOK:  true,
		ActiveTurnState:   ProfileContactTurnIdle,
	}
	root = strings.TrimSpace(root)
	if root == "" || !dirExists(root) {
		contact.Health = ProfileContactHealthOffline
		contact.WorkspaceRootsOK = false
		contact.WorkspaceRootsError = 1
		contact.AttentionBadges = []string{"offline"}
		return contact
	}
	cfg, present, cfgErr := readProfileContactConfig(root)
	roots := configuredWorkspaceRoots(root, cfg, present)
	contact.WorkspaceRootCount, contact.WorkspaceRootsWarning, contact.WorkspaceRootsError = workspaceRootSummary(roots)
	contact.WorkspaceRootsOK = contact.WorkspaceRootsWarning == 0 && contact.WorkspaceRootsError == 0
	if cfgErr != nil {
		contact.Health = ProfileContactHealthWarning
		contact.AttentionBadges = appendBadge(contact.AttentionBadges, "config")
	}
	if present && strings.TrimSpace(cfg.Hermes.Provider) == "" {
		contact.Health = ProfileContactHealthNeedsAuth
		contact.AttentionBadges = appendBadge(contact.AttentionBadges, "auth")
	}
	if !contact.WorkspaceRootsOK {
		if contact.Health == ProfileContactHealthOnline {
			contact.Health = ProfileContactHealthWarning
		}
		contact.AttentionBadges = appendBadge(contact.AttentionBadges, "workspace")
	}
	contact.MicAvailable = contact.Health == ProfileContactHealthOnline
	return normalizeProfileContact(contact)
}

func readProfileContactConfig(root string) (profileConfigReadModel, bool, error) {
	var cfg profileConfigReadModel
	tomlPath := filepath.Join(root, "config.toml")
	data, err := os.ReadFile(tomlPath)
	if err == nil {
		if err := toml.NewDecoder(bytes.NewReader(data)).EnableUnmarshalerInterface().Decode(&cfg); err != nil {
			return cfg, true, err
		}
		return cfg, true, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return cfg, true, err
	}
	yamlPath := filepath.Join(root, "config.yaml")
	data, err = os.ReadFile(yamlPath)
	if err == nil {
		if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&cfg); err != nil {
			return cfg, true, err
		}
		return cfg, true, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return cfg, true, err
	}
	return cfg, false, nil
}

func configuredWorkspaceRoots(root string, cfg profileConfigReadModel, present bool) []string {
	if present {
		roots := compactProfileStrings(cfg.Agents.Defaults.Workspaces)
		if len(roots) > 0 {
			return roots
		}
		if workspace := strings.TrimSpace(cfg.Agents.Defaults.Workspace); workspace != "" {
			return []string{workspace}
		}
	}
	return []string{filepath.Join(root, "workspace")}
}

func workspaceRootSummary(roots []string) (count, warnings, errors int) {
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		count++
		info, err := os.Stat(root)
		switch {
		case err == nil && info.IsDir():
		case err == nil:
			warnings++
		case os.IsNotExist(err):
			warnings++
		default:
			errors++
		}
	}
	return count, warnings, errors
}

func (c *Channel) profileContactRuntimeUpdateLocked(serverID, profileID, preview, previewKind, turnState string) ProfileContact {
	serverID, profileID = normalizeProfileScope(serverID, profileID)
	key := profileContactKey(serverID, profileID)
	contact, ok := c.profileContacts[key]
	if !ok {
		contact = ProfileContact{
			ServerID:            serverID,
			ProfileID:           profileID,
			DisplayName:         profileDisplayName(profileID),
			ServerLabel:         navivoxDefaultServerLabel,
			AvatarSeed:          serverID + ":" + profileID,
			Health:              ProfileContactHealthOnline,
			WorkspaceRootCount:  1,
			WorkspaceRootsOK:    true,
			MicAvailable:        true,
			ActiveTurnState:     ProfileContactTurnIdle,
			LatestPreviewKind:   "status",
			LatestPreview:       "Profile ready",
			LatestPreviewAt:     c.now().UTC().Format(time.RFC3339),
			AttentionBadges:     []string{},
			WorkspaceRootsError: 0,
		}
	}
	contact.ServerID = serverID
	contact.ProfileID = profileID
	contact.LatestPreview = safeNavivoxProfilePreview(preview)
	contact.LatestPreviewKind = strings.TrimSpace(previewKind)
	contact.LatestPreviewAt = c.now().UTC().Format(time.RFC3339)
	contact.ActiveTurnState = strings.TrimSpace(turnState)
	contact = normalizeProfileContact(contact)
	c.profileContacts[key] = contact
	return contact
}

func (c *Channel) updateProfileContactForSession(sessionID, preview, previewKind, turnState string) {
	c.mu.Lock()
	state := c.sessions[sessionID]
	if state == nil || strings.TrimSpace(state.ProfileID) == "" {
		c.mu.Unlock()
		return
	}
	contact := c.profileContactRuntimeUpdateLocked(state.ProfileServer, state.ProfileID, preview, previewKind, turnState)
	c.mu.Unlock()
	c.broadcastProfileContact(contact)
}

func (c *Channel) broadcastProfileContact(contact ProfileContact) {
	contact = normalizeProfileContact(contact)
	c.mu.Lock()
	clients := make([]*client, 0, len(c.clients))
	for cl := range c.clients {
		clients = append(clients, cl)
	}
	c.mu.Unlock()
	for _, cl := range clients {
		_ = cl.write(ServerEvent{Type: "profile_contact_update", Contact: &contact})
	}
}

func profileScopeFromMetadata(metadata map[string]any) (string, string) {
	serverID := navivoxDefaultServerID
	profileID := "main"
	if metadata != nil {
		if value := strings.TrimSpace(anyString(metadata["server_id"])); value != "" {
			serverID = value
		}
		if value := strings.TrimSpace(anyString(metadata["profile_id"])); value != "" {
			profileID = value
		}
	}
	return normalizeProfileScope(serverID, profileID)
}

func normalizeProfileScope(serverID, profileID string) (string, string) {
	serverID = strings.TrimSpace(serverID)
	profileID = strings.TrimSpace(profileID)
	if serverID == "" {
		serverID = navivoxDefaultServerID
	}
	if profileID == "" || profileID == "default" {
		profileID = "main"
	}
	return serverID, profileID
}

func profileContactKey(serverID, profileID string) string {
	serverID, profileID = normalizeProfileScope(serverID, profileID)
	return strconv.Itoa(len(serverID)) + ":" + serverID + strconv.Itoa(len(profileID)) + ":" + profileID
}

func normalizeProfileContact(contact ProfileContact) ProfileContact {
	contact.ServerID, contact.ProfileID = normalizeProfileScope(contact.ServerID, contact.ProfileID)
	if strings.TrimSpace(contact.DisplayName) == "" {
		contact.DisplayName = profileDisplayName(contact.ProfileID)
	}
	if strings.TrimSpace(contact.ServerLabel) == "" {
		contact.ServerLabel = navivoxDefaultServerLabel
	}
	if strings.TrimSpace(contact.AvatarSeed) == "" {
		contact.AvatarSeed = contact.ServerID + ":" + contact.ProfileID
	}
	contact.LatestPreview = safeNavivoxProfilePreview(contact.LatestPreview)
	if strings.TrimSpace(contact.LatestPreview) == "" {
		contact.LatestPreview = "Profile ready"
	}
	if strings.TrimSpace(contact.LatestPreviewKind) == "" {
		contact.LatestPreviewKind = "status"
	}
	if strings.TrimSpace(contact.Health) == "" {
		contact.Health = ProfileContactHealthOnline
	}
	if strings.TrimSpace(contact.ActiveTurnState) == "" {
		contact.ActiveTurnState = ProfileContactTurnIdle
	}
	contact.AttentionBadges = compactProfileStrings(contact.AttentionBadges)
	return contact
}

func mergeProfileContact(base, overlay ProfileContact) ProfileContact {
	overlay = normalizeProfileContact(overlay)
	base.LatestPreview = overlay.LatestPreview
	base.LatestPreviewKind = overlay.LatestPreviewKind
	base.LatestPreviewAt = overlay.LatestPreviewAt
	base.ActiveTurnState = overlay.ActiveTurnState
	if overlay.Health != "" && overlay.Health != ProfileContactHealthOnline {
		// Only apply overlay health when it signals something meaningful.
		// Runtime updates default to "online" and should not overwrite
		// loader-computed health (warning/offline/needs_auth).
		base.Health = overlay.Health
	}
	if len(overlay.AttentionBadges) > 0 {
		base.AttentionBadges = overlay.AttentionBadges
	}
	base.MicAvailable = overlay.MicAvailable
	return normalizeProfileContact(base)
}

func sortProfileContacts(contacts []ProfileContact) {
	sort.SliceStable(contacts, func(i, j int) bool {
		if contacts[i].ProfileID == "main" && contacts[j].ProfileID != "main" {
			return true
		}
		if contacts[j].ProfileID == "main" && contacts[i].ProfileID != "main" {
			return false
		}
		if contacts[i].ServerID != contacts[j].ServerID {
			return contacts[i].ServerID < contacts[j].ServerID
		}
		return contacts[i].ProfileID < contacts[j].ProfileID
	})
}

func profileDisplayName(profileID string) string {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" || profileID == "main" {
		return "Gormes profile"
	}
	parts := strings.FieldsFunc(profileID, func(r rune) bool { return r == '-' || r == '_' || r == '.' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ") + " profile"
}

func safeNavivoxProfilePreview(raw string) string {
	raw = strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	raw = navivoxQuotedSecretPattern.ReplaceAllString(raw, "[redacted]")
	raw = navivoxSecretAssignmentPattern.ReplaceAllString(raw, "[redacted]")
	raw = navivoxPathPattern.ReplaceAllString(raw, "[path]")
	raw = strings.ReplaceAll(raw, "{", "")
	raw = strings.ReplaceAll(raw, "}", "")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Profile ready"
	}
	runes := []rune(raw)
	if len(runes) > 120 {
		return string(runes[:117]) + "..."
	}
	return raw
}

func appendBadge(badges []string, badge string) []string {
	badge = strings.TrimSpace(badge)
	if badge == "" {
		return badges
	}
	for _, existing := range badges {
		if existing == badge {
			return badges
		}
	}
	return append(badges, badge)
}

func compactProfileStrings(values []string) []string { return channelutil.UniqueStrings(values) }

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}
