package config

import (
	"fmt"
	"sort"
	"strings"

	profileconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/profile"
)

const DefaultProfileID = profileconfig.DefaultID

type ProfileCfg = profileconfig.Config
type ProfileRuntimeCfg = profileconfig.RuntimeConfig
type ProfileProviderCfg = profileconfig.ProviderConfig
type ProfileChannelCfg = profileconfig.ChannelConfig
type ProfileVoiceProfileCfg = profileconfig.VoiceProfileConfig
type ProfileVoiceProviderMatrix = profileconfig.VoiceProviderMatrix
type ProfileVoiceProfileValidation = profileconfig.VoiceProfileValidation
type ProfileVoiceProfileFieldError = profileconfig.VoiceProfileFieldError
type ProfileVoiceCredentialStatus = profileconfig.VoiceCredentialStatus
type CredentialCfg = profileconfig.CredentialConfig
type ProfileService = profileconfig.Service
type NavivoxProfileRoutingReport = profileconfig.NavivoxRoutingReport
type NavivoxServerRoute = profileconfig.NavivoxServerRoute
type NavivoxProfileRoute = profileconfig.NavivoxRoute
type NavivoxProfileRouteWarning = profileconfig.NavivoxRouteWarning

func (c Config) ProfileConfigV2Available() bool {
	return len(c.Profiles) > 0
}

func (c Config) EnabledProfileServices() []ProfileService {
	ids := make([]string, 0, len(c.Profiles))
	for id, profile := range c.Profiles {
		if profile.Enabled {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]ProfileService, 0, len(ids))
	for _, id := range ids {
		out = append(out, ProfileService{ID: id, Profile: c.Profiles[id]})
	}
	return out
}

func (c Config) NavivoxProfileRouting() NavivoxProfileRoutingReport {
	if len(c.Navivox.Servers) > 0 {
		return c.navivoxServerScopedProfileRouting()
	}
	services := c.EnabledProfileServices()
	routes := make([]NavivoxProfileRoute, 0, len(services))
	for _, service := range services {
		routes = append(routes, navivoxProfileRouteFromProfile(service.ID, service.Profile))
	}
	return NavivoxProfileRoutingReport{Profiles: routes}
}

func (c Config) navivoxServerScopedProfileRouting() NavivoxProfileRoutingReport {
	serverIDs := profileconfig.SortedKeys(c.Navivox.Servers)
	servers := make([]NavivoxServerRoute, 0, len(serverIDs))
	routedProfiles := map[string]NavivoxProfileRoute{}
	for _, serverID := range serverIDs {
		server := c.Navivox.Servers[serverID]
		if !server.Enabled {
			continue
		}
		serverRoute := NavivoxServerRoute{
			ServerID:     serverID,
			Bind:         strings.TrimSpace(server.Bind),
			Transports:   navivoxRoutingStrings(server.Transports),
			Capabilities: navivoxRoutingStrings(server.Capabilities),
		}
		for _, profileID := range server.Profiles {
			profile, ok := c.Profiles[profileID]
			if !ok {
				serverRoute.Warnings = append(serverRoute.Warnings, navivoxProfileUnavailableWarning(profileID, "profile is not configured"))
				continue
			}
			if !profile.Enabled {
				serverRoute.Warnings = append(serverRoute.Warnings, navivoxProfileUnavailableWarning(profileID, "profile is disabled"))
				continue
			}
			navivoxChannel, ok := profile.Channels["navivox"]
			if !ok || !navivoxChannel.Enabled || !navivoxChannelReferencesServer(navivoxChannel, serverID) {
				serverRoute.Warnings = append(serverRoute.Warnings, navivoxProfileUnavailableWarning(profileID, "profile is not opted into this Navivox server"))
				continue
			}
			route := navivoxProfileRouteFromProfile(profileID, profile)
			route.ServerIDs = []string{serverID}
			route.CredentialConfigured = strings.TrimSpace(navivoxChannel.Credential) != ""
			route.VoiceProfileConfigured = strings.TrimSpace(navivoxChannel.VoiceProfile) != ""
			route.Ready = true
			serverRoute.Profiles = append(serverRoute.Profiles, route)

			union := routedProfiles[profileID]
			if union.ProfileID == "" {
				union = navivoxProfileRouteFromProfile(profileID, profile)
			}
			union.ServerIDs = appendNavivoxRoutingString(union.ServerIDs, serverID)
			union.CredentialConfigured = union.CredentialConfigured || route.CredentialConfigured
			union.VoiceProfileConfigured = union.VoiceProfileConfigured || route.VoiceProfileConfigured
			union.Ready = union.Ready || route.Ready
			routedProfiles[profileID] = union
		}
		servers = append(servers, serverRoute)
	}

	profileIDs := profileconfig.SortedKeys(routedProfiles)
	profiles := make([]NavivoxProfileRoute, 0, len(profileIDs))
	for _, profileID := range profileIDs {
		profiles = append(profiles, routedProfiles[profileID])
	}
	return NavivoxProfileRoutingReport{Profiles: profiles, Servers: servers}
}

func navivoxProfileRouteFromProfile(profileID string, profile ProfileCfg) NavivoxProfileRoute {
	displayName := strings.TrimSpace(profile.Name)
	if displayName == "" {
		displayName = profileID
	}
	return NavivoxProfileRoute{
		ProfileID:   profileID,
		DisplayName: displayName,
		Workspaces:  navivoxRoutingStrings(profile.Workspaces),
		Providers:   navivoxRoutingProviderIDs(profile.Providers),
		Channels:    navivoxRoutingChannelIDs(profile.Channels),
	}
}

func navivoxChannelReferencesServer(channel ProfileChannelCfg, serverID string) bool {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return false
	}
	for _, candidate := range channel.Servers {
		if strings.EqualFold(strings.TrimSpace(candidate), serverID) {
			return true
		}
	}
	return false
}

func navivoxProfileUnavailableWarning(profileID, message string) NavivoxProfileRouteWarning {
	return NavivoxProfileRouteWarning{Code: "navivox_profile_unavailable", ProfileID: profileID, Message: message}
}

func appendNavivoxRoutingString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func navivoxRoutingProviderIDs(providers map[string]ProfileProviderCfg) []string {
	ids := make([]string, 0, len(providers))
	for id, provider := range providers {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" && provider.Enabled {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return navivoxRoutingStrings(ids)
}

func navivoxRoutingChannelIDs(channels map[string]ProfileChannelCfg) []string {
	ids := make([]string, 0, len(channels))
	for id, channel := range channels {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" && channel.Enabled {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return navivoxRoutingStrings(ids)
}

func navivoxRoutingStrings(values []string) []string {
	values = cleanStringSlice(values)
	if len(values) == 0 {
		return nil
	}
	return values
}

// WriteProfileConfigV2 writes the canonical profile-service and credential
// registry portions of cfg into path as one root TOML transaction. Existing
// non-profile sections are preserved in the root document; profile data never
// lands in per-profile config.toml files and raw secret values are represented
// only as SecretRef metadata.
func WriteProfileConfigV2(path string, cfg Config) error {
	cfg.ConfigVersion = CurrentConfigVersion
	if err := normalizeProfileConfigV2(&cfg); err != nil {
		return err
	}
	doc, err := readTOMLDoc(path)
	if err != nil {
		return err
	}
	doc["config_version"] = int64(CurrentConfigVersion)
	if len(cfg.Profiles) == 0 {
		delete(doc, "profiles")
	} else {
		doc["profiles"] = profileconfig.ProfilesDocument(cfg.Profiles)
	}
	if len(cfg.Credentials) == 0 {
		delete(doc, "credentials")
	} else {
		doc["credentials"] = profileconfig.CredentialsDocument(cfg.Credentials)
	}
	return writeTOMLDoc(path, doc)
}

func normalizeProfileConfigV2(cfg *Config) error {
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = nil
	} else {
		profiles := make(map[string]ProfileCfg, len(cfg.Profiles))
		for id, profile := range cfg.Profiles {
			normalizedID := strings.TrimSpace(id)
			if normalizedID != id || !agentIDPattern.MatchString(normalizedID) {
				return fmt.Errorf("config: profile id %q is invalid", id)
			}
			profile.Name = strings.TrimSpace(profile.Name)
			profile.Description = strings.TrimSpace(profile.Description)
			profile.Workspaces = cleanStringSlice(profile.Workspaces)
			profile.Tags = cleanStringSlice(profile.Tags)
			profile.Runtime.SessionResetPolicy = strings.TrimSpace(profile.Runtime.SessionResetPolicy)
			profile.Runtime.GonchoWorkspace = strings.TrimSpace(profile.Runtime.GonchoWorkspace)
			profile.Providers = normalizeProfileProviders(profile.Providers)
			profile.Channels = normalizeProfileChannels(profile.Channels)
			profile.VoiceProfile = profileconfig.NormalizeVoiceProfile(profile.VoiceProfile)
			profiles[normalizedID] = profile
		}
		cfg.Profiles = profiles
	}

	if len(cfg.Credentials) == 0 {
		cfg.Credentials = nil
		return nil
	}
	credentials := make(map[string]CredentialCfg, len(cfg.Credentials))
	for id, credential := range cfg.Credentials {
		normalizedID := strings.TrimSpace(id)
		if normalizedID != id || !agentIDPattern.MatchString(normalizedID) {
			return fmt.Errorf("config: credential id %q is invalid", id)
		}
		credential.Kind = strings.ToLower(strings.TrimSpace(credential.Kind))
		credential.Provider = strings.ToLower(strings.TrimSpace(credential.Provider))
		credential.Channel = strings.ToLower(strings.TrimSpace(credential.Channel))
		credential.OwnerProfile = strings.TrimSpace(credential.OwnerProfile)
		if credential.OwnerProfile != "" {
			if !agentIDPattern.MatchString(credential.OwnerProfile) {
				return fmt.Errorf("config: credentials.%s.owner_profile %q is invalid", normalizedID, credential.OwnerProfile)
			}
			if len(cfg.Profiles) > 0 {
				if _, ok := cfg.Profiles[credential.OwnerProfile]; !ok {
					return fmt.Errorf("config: credentials.%s.owner_profile %q does not match a configured profile", normalizedID, credential.OwnerProfile)
				}
			}
		}
		if credential.SecretRef != nil {
			ref := normalizeSecretRef(*credential.SecretRef)
			switch ref.Source {
			case SecretRefSourceEnv, SecretRefSourceFile, SecretRefSourceExec:
			default:
				return fmt.Errorf("config: credentials.%s.secret_ref.source %q is invalid", normalizedID, ref.Source)
			}
			if strings.TrimSpace(ref.ID) == "" {
				return fmt.Errorf("config: credentials.%s.secret_ref.id is required", normalizedID)
			}
			credential.SecretRef = &ref
		}
		credentials[normalizedID] = credential
	}
	cfg.Credentials = credentials
	return nil
}

func normalizeProfileVoiceProfile(voice ProfileVoiceProfileCfg) ProfileVoiceProfileCfg {
	return profileconfig.NormalizeVoiceProfile(voice)
}

func ValidateProfileVoiceProfile(profileID string, voice ProfileVoiceProfileCfg, matrix ProfileVoiceProviderMatrix) ProfileVoiceProfileValidation {
	return profileconfig.ValidateVoiceProfile(profileID, voice, matrix)
}

func normalizeProfileProviders(in map[string]ProfileProviderCfg) map[string]ProfileProviderCfg {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ProfileProviderCfg, len(in))
	for provider, cfg := range in {
		key := strings.ToLower(strings.TrimSpace(provider))
		if key == "" {
			continue
		}
		cfg.Credential = strings.TrimSpace(cfg.Credential)
		cfg.DefaultModel = strings.TrimSpace(cfg.DefaultModel)
		cfg.AllowedModels = cleanStringSlice(cfg.AllowedModels)
		cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
		out[key] = cfg
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeProfileChannels(in map[string]ProfileChannelCfg) map[string]ProfileChannelCfg {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ProfileChannelCfg, len(in))
	for channel, cfg := range in {
		key := strings.ToLower(strings.TrimSpace(channel))
		if key == "" {
			continue
		}
		cfg.Credential = strings.TrimSpace(cfg.Credential)
		cfg.AllowedChats = cleanStringSlice(cfg.AllowedChats)
		cfg.AllowedUsers = cleanStringSlice(cfg.AllowedUsers)
		cfg.ToolProgress = strings.ToLower(strings.TrimSpace(cfg.ToolProgress))
		cfg.Servers = cleanStringSlice(cfg.Servers)
		cfg.VoiceProfile = strings.TrimSpace(cfg.VoiceProfile)
		out[key] = cfg
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
