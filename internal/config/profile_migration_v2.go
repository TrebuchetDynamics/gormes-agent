package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// ProfileMigrationV2Options configures the legacy profile-state migration
// planner. Home defaults to GormesHome and ConfigPath defaults to
// $GORMES_HOME/config.toml. Now is used only by Apply for deterministic backup
// names in tests.
type ProfileMigrationV2Options struct {
	Home       string
	ConfigPath string
	Now        func() time.Time
}

type ProfileMigrationV2Plan struct {
	Home                string
	ConfigPath          string
	NoOp                bool
	ProfileAdditions    []ProfileMigrationV2ProfileAddition
	CredentialAdditions []ProfileMigrationV2CredentialAddition
	ProviderLinks       []ProfileMigrationV2ProviderLink
	ChannelLinks        []ProfileMigrationV2ChannelLink
	FallbackReads       []ProfileMigrationV2FallbackRead
	SecretMovements     []ProfileMigrationV2SecretMovement
	ManualActions       []ProfileMigrationV2ManualAction
	Conflicts           []ProfileMigrationV2Conflict
	ActiveProfile       string
	PreviewLines        []string
}

type ProfileMigrationV2ProfileAddition struct {
	ID          string
	Enabled     bool
	DisplayName string
	Workspaces  []string
	Providers   []string
	Channels    []string
	SourcePath  string
}

type ProfileMigrationV2CredentialAddition struct {
	ID           string
	Kind         string
	Provider     string
	Channel      string
	OwnerProfile string
	SecretRef    *SecretRef
}

type ProfileMigrationV2ProviderLink struct {
	ProfileID    string
	Provider     string
	CredentialID string
	DefaultModel string
	Endpoint     string
}

type ProfileMigrationV2ChannelLink struct {
	ProfileID        string
	Channel          string
	CredentialID     string
	AllowedChats     []string
	AllowedUsers     []string
	RequireMention   bool
	ToolProgress     string
	SlackAppTokenEnv string
}

type ProfileMigrationV2FallbackRead struct {
	Code string
	Path string
}

type ProfileMigrationV2SecretMovement struct {
	Source       string
	TargetEnv    string
	CredentialID string
	Redacted     bool
}

type ProfileMigrationV2ManualAction struct {
	Code    string
	Message string
}

type ProfileMigrationV2Conflict struct {
	Kind       string
	ID         string
	SourcePath string
	Resolution string
}

type ProfileMigrationV2ApplyResult struct {
	Path       string
	BackupPath string
	NoOp       bool
	Wrote      bool
	Plan       ProfileMigrationV2Plan
}

type profileMigrationV2Source struct {
	ID     string
	Path   string
	Code   string
	Raw    map[string]any
	Config Config
}

// PlanProfileConfigV2Migration returns a deterministic, redacted plan for
// moving legacy root/per-profile config.toml state into the v2 single-root
// profile schema. It never mutates files.
func PlanProfileConfigV2Migration(opts ProfileMigrationV2Options) (ProfileMigrationV2Plan, error) {
	home := profileMigrationV2Home(opts)
	configPath := profileMigrationV2ConfigPath(opts, home)
	plan := ProfileMigrationV2Plan{Home: home, ConfigPath: configPath}

	rootSource, err := readProfileMigrationV2Source(DefaultProfileID, configPath, "root_config")
	if err != nil {
		return plan, err
	}
	plan.FallbackReads = append(plan.FallbackReads, ProfileMigrationV2FallbackRead{Code: "root_config", Path: configPath})
	if profileMigrationV2AlreadyCurrent(rootSource.Raw) {
		plan.NoOp = true
		plan.PreviewLines = []string{fmt.Sprintf("no-op: %s already has config_version=%d profile config", configPath, CurrentConfigVersion)}
		return plan, nil
	}

	if _, err := os.Stat(filepath.Join(home, ".env")); err == nil {
		plan.FallbackReads = append(plan.FallbackReads, ProfileMigrationV2FallbackRead{Code: "env_file", Path: filepath.Join(home, ".env")})
	} else if err != nil && !os.IsNotExist(err) {
		return plan, fmt.Errorf("config: stat %s: %w", filepath.Join(home, ".env"), err)
	}
	if body, err := os.ReadFile(filepath.Join(home, "active_profile")); err == nil {
		plan.FallbackReads = append(plan.FallbackReads, ProfileMigrationV2FallbackRead{Code: "active_profile", Path: filepath.Join(home, "active_profile")})
		plan.ActiveProfile = strings.TrimSpace(string(body))
		if plan.ActiveProfile != "" {
			plan.ManualActions = append(plan.ManualActions, ProfileMigrationV2ManualAction{
				Code:    "legacy_active_profile_compatibility",
				Message: fmt.Sprintf("legacy active_profile=%s is compatibility state only; v2 keeps all enabled profiles active", plan.ActiveProfile),
			})
		}
	} else if err != nil && !os.IsNotExist(err) {
		return plan, fmt.Errorf("config: read %s: %w", filepath.Join(home, "active_profile"), err)
	}

	sources := []profileMigrationV2Source{rootSource}
	profileSources, err := readProfileMigrationV2ProfileSources(home)
	if err != nil {
		return plan, err
	}
	for _, source := range profileSources {
		plan.FallbackReads = append(plan.FallbackReads, ProfileMigrationV2FallbackRead{Code: "profile_config", Path: source.Path})
		sources = append(sources, source)
	}

	existingProfiles := profileMigrationV2TableKeys(rootSource.Raw["profiles"])
	existingCredentials := profileMigrationV2TableKeys(rootSource.Raw["credentials"])
	for _, source := range sources {
		if existingProfiles[source.ID] {
			plan.Conflicts = append(plan.Conflicts, ProfileMigrationV2Conflict{Kind: "profile_id", ID: source.ID, SourcePath: source.Path, Resolution: "rename_or_skip"})
			continue
		}
		profile := profileMigrationV2ProfileFromSource(source)
		plan.ProfileAdditions = append(plan.ProfileAdditions, profile)
		plan.ProviderLinks, plan.CredentialAdditions, plan.SecretMovements, plan.Conflicts = profileMigrationV2AppendProvider(plan.ProviderLinks, plan.CredentialAdditions, plan.SecretMovements, plan.Conflicts, source, existingCredentials)
		plan.ChannelLinks, plan.CredentialAdditions, plan.SecretMovements, plan.Conflicts = profileMigrationV2AppendChannels(plan.ChannelLinks, plan.CredentialAdditions, plan.SecretMovements, plan.Conflicts, source, existingCredentials)
	}

	plan.ProfileAdditions = sortProfileMigrationV2Profiles(plan.ProfileAdditions)
	plan.CredentialAdditions = sortProfileMigrationV2Credentials(dedupeProfileMigrationV2Credentials(plan.CredentialAdditions))
	plan.ProviderLinks = sortProfileMigrationV2ProviderLinks(plan.ProviderLinks)
	plan.ChannelLinks = sortProfileMigrationV2ChannelLinks(plan.ChannelLinks)
	plan.SecretMovements = sortProfileMigrationV2SecretMovements(dedupeProfileMigrationV2SecretMovements(plan.SecretMovements))
	plan.FallbackReads = sortProfileMigrationV2FallbackReads(plan.FallbackReads)
	plan.ManualActions = sortProfileMigrationV2ManualActions(plan.ManualActions)
	plan.Conflicts = sortProfileMigrationV2Conflicts(plan.Conflicts)
	plan.PreviewLines = profileMigrationV2Preview(plan)
	return plan, nil
}

// ApplyProfileConfigV2Migration writes the planned v2 root config after first
// creating a backup. It refuses to apply plans with id/profile conflicts and
// never deletes legacy profile directories.
func ApplyProfileConfigV2Migration(opts ProfileMigrationV2Options) (ProfileMigrationV2ApplyResult, error) {
	plan, err := PlanProfileConfigV2Migration(opts)
	if err != nil {
		return ProfileMigrationV2ApplyResult{}, err
	}
	result := ProfileMigrationV2ApplyResult{Path: plan.ConfigPath, Plan: plan, NoOp: plan.NoOp}
	if plan.NoOp {
		return result, nil
	}
	if len(plan.Conflicts) > 0 {
		return result, fmt.Errorf("config: profile v2 migration has unresolved conflict %s (%s); choose rename_or_skip before applying", plan.Conflicts[0].ID, plan.Conflicts[0].Kind)
	}
	backupPath, err := backupProfileMigrationV2Config(plan.ConfigPath, profileMigrationV2Now(opts))
	if err != nil {
		return result, err
	}
	if err := writeTOMLAtomic(plan.ConfigPath, profileMigrationV2Document(plan)); err != nil {
		return result, err
	}
	result.BackupPath = backupPath
	result.Wrote = true
	return result, nil
}

func profileMigrationV2Home(opts ProfileMigrationV2Options) string {
	if strings.TrimSpace(opts.Home) != "" {
		return strings.TrimSpace(opts.Home)
	}
	return GormesHome()
}

func profileMigrationV2ConfigPath(opts ProfileMigrationV2Options, home string) string {
	if strings.TrimSpace(opts.ConfigPath) != "" {
		return strings.TrimSpace(opts.ConfigPath)
	}
	return filepath.Join(home, "config.toml")
}

func profileMigrationV2Now(opts ProfileMigrationV2Options) time.Time {
	if opts.Now != nil {
		return opts.Now().UTC()
	}
	return time.Now().UTC()
}

func readProfileMigrationV2Source(id, path, code string) (profileMigrationV2Source, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return profileMigrationV2Source{}, fmt.Errorf("config: read %s: %w", path, err)
	}
	raw := map[string]any{}
	if err := toml.Unmarshal(body, &raw); err != nil {
		return profileMigrationV2Source{}, fmt.Errorf("config: parse %s: %w", path, err)
	}
	var cfg Config
	if err := toml.Unmarshal(body, &cfg); err != nil {
		return profileMigrationV2Source{}, fmt.Errorf("config: decode %s: %w", path, err)
	}
	return profileMigrationV2Source{ID: id, Path: path, Code: code, Raw: raw, Config: cfg}, nil
}

func readProfileMigrationV2ProfileSources(home string) ([]profileMigrationV2Source, error) {
	profilesDir := filepath.Join(home, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", profilesDir, err)
	}
	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := strings.ToLower(strings.TrimSpace(entry.Name()))
		if id == "" || id == DefaultProfileID || !agentIDPattern.MatchString(id) {
			continue
		}
		if _, err := os.Stat(filepath.Join(profilesDir, entry.Name(), "config.toml")); err == nil {
			ids = append(ids, entry.Name())
		} else if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("config: stat %s: %w", filepath.Join(profilesDir, entry.Name(), "config.toml"), err)
		}
	}
	sort.Strings(ids)
	out := make([]profileMigrationV2Source, 0, len(ids))
	for _, id := range ids {
		path := filepath.Join(profilesDir, id, "config.toml")
		source, err := readProfileMigrationV2Source(strings.ToLower(id), path, "profile_config")
		if err != nil {
			return nil, err
		}
		out = append(out, source)
	}
	return out, nil
}

func profileMigrationV2AlreadyCurrent(raw map[string]any) bool {
	version := readConfigVersion(raw)
	if version == 0 {
		version = 1
	}
	return version == CurrentConfigVersion && len(profileMigrationV2TableKeys(raw["profiles"])) > 0
}

func profileMigrationV2TableKeys(raw any) map[string]bool {
	out := map[string]bool{}
	table, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	for key := range table {
		out[strings.TrimSpace(key)] = true
	}
	return out
}

func profileMigrationV2ProfileFromSource(source profileMigrationV2Source) ProfileMigrationV2ProfileAddition {
	profile := ProfileMigrationV2ProfileAddition{ID: source.ID, Enabled: true, DisplayName: profileMigrationV2DisplayName(source.ID), SourcePath: source.Path}
	profile.Workspaces = profileMigrationV2Workspaces(source.Config)
	if provider := profileMigrationV2ProviderName(source.Config); provider != "" {
		profile.Providers = []string{provider}
	}
	profile.Channels = profileMigrationV2Channels(source.Config)
	return profile
}

func profileMigrationV2DisplayName(id string) string {
	if id == DefaultProfileID {
		return ""
	}
	return id
}

func profileMigrationV2Workspaces(cfg Config) []string {
	var values []string
	if strings.TrimSpace(cfg.Agents.Defaults.Workspace) != "" {
		values = append(values, cfg.Agents.Defaults.Workspace)
	}
	values = append(values, cfg.Agents.Defaults.Workspaces...)
	for _, agent := range cfg.Agents.List {
		if strings.TrimSpace(agent.Workspace) != "" {
			values = append(values, agent.Workspace)
		}
	}
	return compactStrings(values)
}

func profileMigrationV2ProviderName(cfg Config) string {
	provider := strings.ToLower(strings.TrimSpace(cfg.Hermes.Provider))
	if provider == "" && (strings.TrimSpace(cfg.Hermes.Model) != "" || strings.TrimSpace(cfg.Hermes.Endpoint) != "" || strings.TrimSpace(cfg.Hermes.APIKey) != "" || cfg.Hermes.APIKeyRef != nil) {
		provider = "openai-codex"
	}
	return provider
}

func profileMigrationV2Channels(cfg Config) []string {
	seen := map[string]bool{}
	for _, channel := range cfg.Agents.Defaults.Channels {
		channel = strings.ToLower(strings.TrimSpace(channel))
		if channel != "" {
			seen[channel] = true
		}
	}
	if cfg.Telegram.BotToken != "" || cfg.Telegram.BotTokenRef != nil || cfg.Telegram.AllowedChatID != 0 || len(cfg.Telegram.AllowedChatIDs()) > 0 {
		seen["telegram"] = true
	}
	if cfg.Discord.Token != "" || cfg.Discord.TokenRef != nil || strings.TrimSpace(cfg.Discord.AllowedChannelID) != "" || len(cfg.Discord.AllowedChannelIDs()) > 0 {
		seen["discord"] = true
	}
	if cfg.Slack.BotToken != "" || cfg.Slack.BotTokenRef != nil || cfg.Slack.AppToken != "" || cfg.Slack.AppTokenRef != nil || strings.TrimSpace(cfg.Slack.AllowedChannelID) != "" || len(cfg.Slack.AllowedChannelIDs()) > 0 {
		seen["slack"] = true
	}
	out := make([]string, 0, len(seen))
	for channel := range seen {
		out = append(out, channel)
	}
	sort.Strings(out)
	return out
}

func profileMigrationV2AppendProvider(links []ProfileMigrationV2ProviderLink, credentials []ProfileMigrationV2CredentialAddition, moves []ProfileMigrationV2SecretMovement, conflicts []ProfileMigrationV2Conflict, source profileMigrationV2Source, existing map[string]bool) ([]ProfileMigrationV2ProviderLink, []ProfileMigrationV2CredentialAddition, []ProfileMigrationV2SecretMovement, []ProfileMigrationV2Conflict) {
	provider := profileMigrationV2ProviderName(source.Config)
	if provider == "" {
		return links, credentials, moves, conflicts
	}
	credID := source.ID + "-" + provider
	if existing[credID] {
		conflicts = append(conflicts, ProfileMigrationV2Conflict{Kind: "credential_id", ID: credID, SourcePath: source.Path, Resolution: "rename_or_skip"})
		return links, credentials, moves, conflicts
	}
	ref := source.Config.Hermes.APIKeyRef
	targetEnv := profileMigrationV2ProviderEnv(source.ID, provider)
	if ref == nil {
		ref = &SecretRef{Source: SecretRefSourceEnv, ID: targetEnv}
	}
	credentials = append(credentials, ProfileMigrationV2CredentialAddition{ID: credID, Kind: "provider", Provider: provider, OwnerProfile: source.ID, SecretRef: ref})
	links = append(links, ProfileMigrationV2ProviderLink{ProfileID: source.ID, Provider: provider, CredentialID: credID, DefaultModel: strings.TrimSpace(source.Config.Hermes.Model), Endpoint: strings.TrimSpace(source.Config.Hermes.Endpoint)})
	if strings.TrimSpace(source.Config.Hermes.APIKey) != "" {
		moves = append(moves, ProfileMigrationV2SecretMovement{Source: source.Code + ".hermes.api_key", TargetEnv: targetEnv, CredentialID: credID, Redacted: true})
	}
	return links, credentials, moves, conflicts
}

func profileMigrationV2AppendChannels(links []ProfileMigrationV2ChannelLink, credentials []ProfileMigrationV2CredentialAddition, moves []ProfileMigrationV2SecretMovement, conflicts []ProfileMigrationV2Conflict, source profileMigrationV2Source, existing map[string]bool) ([]ProfileMigrationV2ChannelLink, []ProfileMigrationV2CredentialAddition, []ProfileMigrationV2SecretMovement, []ProfileMigrationV2Conflict) {
	for _, channel := range profileMigrationV2Channels(source.Config) {
		credID := source.ID + "-" + channel
		if existing[credID] {
			conflicts = append(conflicts, ProfileMigrationV2Conflict{Kind: "credential_id", ID: credID, SourcePath: source.Path, Resolution: "rename_or_skip"})
			continue
		}
		ref, targetEnv, rawSecret := profileMigrationV2ChannelSecret(source.ID, channel, source.Config)
		credentials = append(credentials, ProfileMigrationV2CredentialAddition{ID: credID, Kind: "channel", Channel: channel, OwnerProfile: source.ID, SecretRef: ref})
		links = append(links, profileMigrationV2ChannelLink(source.ID, channel, credID, source.Config))
		if rawSecret {
			moves = append(moves, ProfileMigrationV2SecretMovement{Source: source.Code + "." + channel + ".token", TargetEnv: targetEnv, CredentialID: credID, Redacted: true})
		}
	}
	return links, credentials, moves, conflicts
}

func profileMigrationV2ChannelSecret(profileID, channel string, cfg Config) (*SecretRef, string, bool) {
	targetEnv := profileMigrationV2ChannelEnv(profileID, channel)
	switch channel {
	case "telegram":
		if cfg.Telegram.BotTokenRef != nil {
			return cfg.Telegram.BotTokenRef, cfg.Telegram.BotTokenRef.ID, false
		}
		return &SecretRef{Source: SecretRefSourceEnv, ID: targetEnv}, targetEnv, strings.TrimSpace(cfg.Telegram.BotToken) != ""
	case "discord":
		if cfg.Discord.TokenRef != nil {
			return cfg.Discord.TokenRef, cfg.Discord.TokenRef.ID, false
		}
		return &SecretRef{Source: SecretRefSourceEnv, ID: targetEnv}, targetEnv, strings.TrimSpace(cfg.Discord.Token) != ""
	case "slack":
		if cfg.Slack.BotTokenRef != nil {
			return cfg.Slack.BotTokenRef, cfg.Slack.BotTokenRef.ID, false
		}
		return &SecretRef{Source: SecretRefSourceEnv, ID: targetEnv}, targetEnv, strings.TrimSpace(cfg.Slack.BotToken) != ""
	default:
		return &SecretRef{Source: SecretRefSourceEnv, ID: targetEnv}, targetEnv, false
	}
}

func profileMigrationV2ChannelLink(profileID, channel, credID string, cfg Config) ProfileMigrationV2ChannelLink {
	link := ProfileMigrationV2ChannelLink{ProfileID: profileID, Channel: channel, CredentialID: credID}
	switch channel {
	case "telegram":
		link.AllowedChats = profileMigrationV2TelegramChats(cfg.Telegram)
		for _, id := range cfg.Telegram.AllowedUserIDs {
			if id != 0 {
				link.AllowedUsers = append(link.AllowedUsers, strconv.FormatInt(id, 10))
			}
		}
		link.RequireMention = cfg.Telegram.RequireMention
		if cfg.Display.Platforms != nil {
			link.ToolProgress = strings.TrimSpace(cfg.Display.Platforms["telegram"].ToolProgress)
		}
	case "discord":
		if strings.TrimSpace(cfg.Discord.AllowedChannelID) != "" {
			link.AllowedChats = append(link.AllowedChats, strings.TrimSpace(cfg.Discord.AllowedChannelID))
		}
		link.AllowedChats = append(link.AllowedChats, cfg.Discord.AllowedChannelIDs()...)
	case "slack":
		if strings.TrimSpace(cfg.Slack.AllowedChannelID) != "" {
			link.AllowedChats = append(link.AllowedChats, strings.TrimSpace(cfg.Slack.AllowedChannelID))
		}
		link.AllowedChats = append(link.AllowedChats, cfg.Slack.AllowedChannelIDs()...)
		if strings.TrimSpace(cfg.Slack.AppToken) != "" {
			link.SlackAppTokenEnv = profileMigrationV2ChannelEnv(profileID, "slack_app")
		}
	}
	link.AllowedChats = compactStrings(link.AllowedChats)
	link.AllowedUsers = compactStrings(link.AllowedUsers)
	return link
}

func profileMigrationV2TelegramChats(cfg TelegramCfg) []string {
	var chats []string
	if cfg.AllowedChatID != 0 {
		chats = append(chats, strconv.FormatInt(cfg.AllowedChatID, 10))
	}
	chats = append(chats, cfg.AllowedChatIDs()...)
	return compactStrings(chats)
}

func profileMigrationV2ProviderEnv(profileID, provider string) string {
	return "GORMES_" + profileMigrationV2EnvPart(profileID) + "_" + profileMigrationV2EnvPart(provider) + "_API_KEY"
}

func profileMigrationV2ChannelEnv(profileID, channel string) string {
	prefix := "GORMES_" + profileMigrationV2EnvPart(profileID) + "_"
	switch channel {
	case "telegram":
		return prefix + "TELEGRAM_BOT_TOKEN"
	case "slack_app":
		return prefix + "SLACK_APP_TOKEN"
	default:
		return prefix + profileMigrationV2EnvPart(channel) + "_TOKEN"
	}
}

func profileMigrationV2EnvPart(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.NewReplacer("-", "_", ".", "_", "/", "_").Replace(value)
	return value
}

func profileMigrationV2Document(plan ProfileMigrationV2Plan) map[string]any {
	doc := map[string]any{"config_version": int64(CurrentConfigVersion)}
	profiles := map[string]any{}
	for _, profile := range plan.ProfileAdditions {
		entry := map[string]any{
			"enabled":    profile.Enabled,
			"name":       profile.DisplayName,
			"workspaces": profile.Workspaces,
		}
		providers := map[string]any{}
		for _, link := range plan.ProviderLinks {
			if link.ProfileID != profile.ID {
				continue
			}
			provider := map[string]any{"enabled": true, "credential": link.CredentialID}
			if link.DefaultModel != "" {
				provider["default_model"] = link.DefaultModel
			}
			if link.Endpoint != "" {
				provider["endpoint"] = link.Endpoint
			}
			providers[link.Provider] = provider
		}
		if len(providers) > 0 {
			entry["providers"] = providers
		}
		channels := map[string]any{}
		for _, link := range plan.ChannelLinks {
			if link.ProfileID != profile.ID {
				continue
			}
			channel := map[string]any{"enabled": true, "credential": link.CredentialID}
			if len(link.AllowedChats) > 0 {
				channel["allowed_chats"] = link.AllowedChats
			}
			if len(link.AllowedUsers) > 0 {
				channel["allowed_users"] = link.AllowedUsers
			}
			if link.RequireMention {
				channel["require_mention"] = true
			}
			if link.ToolProgress != "" {
				channel["tool_progress"] = link.ToolProgress
			}
			channels[link.Channel] = channel
		}
		if len(channels) > 0 {
			entry["channels"] = channels
		}
		profiles[profile.ID] = entry
	}
	if len(profiles) == 0 {
		profiles[DefaultProfileID] = map[string]any{"enabled": true, "name": ""}
	}
	doc["profiles"] = profiles
	credentials := map[string]any{}
	for _, credential := range plan.CredentialAdditions {
		entry := map[string]any{"kind": credential.Kind, "owner_profile": credential.OwnerProfile}
		if credential.Provider != "" {
			entry["provider"] = credential.Provider
		}
		if credential.Channel != "" {
			entry["channel"] = credential.Channel
		}
		if credential.SecretRef != nil {
			ref := map[string]any{"source": string(credential.SecretRef.Source), "id": credential.SecretRef.ID}
			if credential.SecretRef.Provider != "" {
				ref["provider"] = credential.SecretRef.Provider
			}
			entry["secret_ref"] = ref
		}
		credentials[credential.ID] = entry
	}
	if len(credentials) > 0 {
		doc["credentials"] = credentials
	}
	return doc
}

func backupProfileMigrationV2Config(path string, now time.Time) (string, error) {
	backup := fmt.Sprintf("%s.bak.%s", path, now.Format("20060102T150405Z"))
	in, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("config: open backup source %s: %w", path, err)
	}
	defer in.Close()
	out, err := os.OpenFile(backup, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("config: create backup %s: %w", backup, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(backup)
		return "", fmt.Errorf("config: write backup %s: %w", backup, err)
	}
	if err := out.Close(); err != nil {
		os.Remove(backup)
		return "", fmt.Errorf("config: close backup %s: %w", backup, err)
	}
	return backup, nil
}

func profileMigrationV2Preview(plan ProfileMigrationV2Plan) []string {
	var lines []string
	for _, profile := range plan.ProfileAdditions {
		lines = append(lines, "add profiles."+profile.ID)
	}
	for _, credential := range plan.CredentialAdditions {
		if credential.SecretRef != nil {
			lines = append(lines, fmt.Sprintf("add credentials.%s secret_ref=%s:%s redacted=true", credential.ID, credential.SecretRef.Source, credential.SecretRef.ID))
		} else {
			lines = append(lines, "add credentials."+credential.ID+" secret_ref=missing")
		}
	}
	for _, action := range plan.ManualActions {
		lines = append(lines, action.Message)
	}
	for _, conflict := range plan.Conflicts {
		lines = append(lines, fmt.Sprintf("conflict %s.%s requires %s", conflict.Kind, conflict.ID, conflict.Resolution))
	}
	sort.Strings(lines)
	return lines
}

func sortProfileMigrationV2Profiles(in []ProfileMigrationV2ProfileAddition) []ProfileMigrationV2ProfileAddition {
	sort.Slice(in, func(i, j int) bool { return in[i].ID < in[j].ID })
	for i := range in {
		sort.Strings(in[i].Providers)
		sort.Strings(in[i].Channels)
		in[i].Workspaces = compactStrings(in[i].Workspaces)
	}
	return in
}

func dedupeProfileMigrationV2Credentials(in []ProfileMigrationV2CredentialAddition) []ProfileMigrationV2CredentialAddition {
	seen := map[string]bool{}
	out := make([]ProfileMigrationV2CredentialAddition, 0, len(in))
	for _, credential := range in {
		if credential.ID == "" || seen[credential.ID] {
			continue
		}
		seen[credential.ID] = true
		out = append(out, credential)
	}
	return out
}

func sortProfileMigrationV2Credentials(in []ProfileMigrationV2CredentialAddition) []ProfileMigrationV2CredentialAddition {
	sort.Slice(in, func(i, j int) bool { return in[i].ID < in[j].ID })
	return in
}

func sortProfileMigrationV2ProviderLinks(in []ProfileMigrationV2ProviderLink) []ProfileMigrationV2ProviderLink {
	sort.Slice(in, func(i, j int) bool {
		if in[i].ProfileID == in[j].ProfileID {
			return in[i].Provider < in[j].Provider
		}
		return in[i].ProfileID < in[j].ProfileID
	})
	return in
}

func sortProfileMigrationV2ChannelLinks(in []ProfileMigrationV2ChannelLink) []ProfileMigrationV2ChannelLink {
	sort.Slice(in, func(i, j int) bool {
		if in[i].ProfileID == in[j].ProfileID {
			return in[i].Channel < in[j].Channel
		}
		return in[i].ProfileID < in[j].ProfileID
	})
	return in
}

func dedupeProfileMigrationV2SecretMovements(in []ProfileMigrationV2SecretMovement) []ProfileMigrationV2SecretMovement {
	seen := map[string]bool{}
	out := make([]ProfileMigrationV2SecretMovement, 0, len(in))
	for _, move := range in {
		key := move.CredentialID + "\x00" + move.TargetEnv
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, move)
	}
	return out
}

func sortProfileMigrationV2SecretMovements(in []ProfileMigrationV2SecretMovement) []ProfileMigrationV2SecretMovement {
	sort.Slice(in, func(i, j int) bool { return in[i].TargetEnv < in[j].TargetEnv })
	return in
}

func sortProfileMigrationV2FallbackReads(in []ProfileMigrationV2FallbackRead) []ProfileMigrationV2FallbackRead {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Code == in[j].Code {
			return in[i].Path < in[j].Path
		}
		return in[i].Code < in[j].Code
	})
	return in
}

func sortProfileMigrationV2ManualActions(in []ProfileMigrationV2ManualAction) []ProfileMigrationV2ManualAction {
	sort.Slice(in, func(i, j int) bool { return in[i].Code < in[j].Code })
	return in
}

func sortProfileMigrationV2Conflicts(in []ProfileMigrationV2Conflict) []ProfileMigrationV2Conflict {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Kind == in[j].Kind {
			return in[i].ID < in[j].ID
		}
		return in[i].Kind < in[j].Kind
	})
	return in
}
