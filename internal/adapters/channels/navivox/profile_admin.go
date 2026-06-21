package navivox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/pelletier/go-toml/v2"
)

// profileAdminBackend resolves per-profile config paths under a Gormes home.
type profileAdminBackend struct {
	gormesHome func() string
}

func defaultProfileAdminBackend() profileAdminBackend {
	return profileAdminBackend{gormesHome: config.GormesHome}
}

// WithProfileAdminHome overrides the Gormes home used by the profile admin
// backend. This is primarily used in tests to point at a temp directory.
func WithProfileAdminHome(gormesHome func() string) ChannelOption {
	return func(c *Channel) {
		c.profileAdmin = profileAdminBackend{gormesHome: gormesHome}
	}
}

type profileAdminEntry struct {
	ProfileID   string `json:"profile_id"`
	DisplayName string `json:"display_name,omitempty"`
	ConfigPath  string `json:"-"`
}

var profileIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$|^[a-z0-9]$`)

func (b profileAdminBackend) home() string {
	if b.gormesHome != nil {
		return b.gormesHome()
	}
	return config.GormesHome()
}

func (b profileAdminBackend) profilesRoot() string {
	return filepath.Join(b.home(), "profiles")
}

func (b profileAdminBackend) profileConfigPath(profileID string) string {
	return filepath.Join(b.profilesRoot(), profileID, "config.toml")
}

func (b profileAdminBackend) listProfiles() ([]profileAdminEntry, error) {
	root := b.profilesRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var profiles []profileAdminEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if !profileIDPattern.MatchString(id) {
			continue
		}
		cfgPath := filepath.Join(root, id, "config.toml")
		if _, err := os.Stat(cfgPath); err != nil {
			continue
		}
		profiles = append(profiles, profileAdminEntry{
			ProfileID:   id,
			DisplayName: profileDisplayName(id),
			ConfigPath:  cfgPath,
		})
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].ProfileID < profiles[j].ProfileID })
	return profiles, nil
}

// profileAdminSchema returns the editable fields for a per-profile config.
func profileAdminSchema() []configAdminField {
	return []configAdminField{
		{Key: "hermes.provider", Type: "string", Title: "Provider", Description: "LLM provider (openai, anthropic, openrouter, etc.)", Reload: "restart_or_reload"},
		{Key: "hermes.model", Type: "string", Title: "Model", Description: "Default model name", Reload: "restart_or_reload"},
		{Key: "agents.defaults.workspace", Type: "string", Title: "Workspace", Description: "Primary workspace root path", Reload: "restart_or_reload"},
		{Key: "agents.defaults.workspaces", Type: "string_list", Title: "Workspaces", Description: "All workspace root paths", Reload: "restart_or_reload"},
	}
}

type profileAdminReadModel struct {
	Hermes struct {
		Provider string `toml:"provider"`
		Model    string `toml:"model"`
	} `toml:"hermes"`
	Agents struct {
		Defaults struct {
			Workspace  string   `toml:"workspace"`
			Workspaces []string `toml:"workspaces"`
		} `toml:"defaults"`
	} `toml:"agents"`
}

func (b profileAdminBackend) readProfileConfig(profileID string) (profileAdminReadModel, error) {
	cfgPath := b.profileConfigPath(profileID)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return profileAdminReadModel{}, nil
		}
		return profileAdminReadModel{}, err
	}
	var m profileAdminReadModel
	if err := parseTOMLIntoProfileAdmin(data, &m); err != nil {
		return profileAdminReadModel{}, nil
	}
	return m, nil
}

func parseTOMLIntoProfileAdmin(data []byte, out *profileAdminReadModel) error {
	return toml.Unmarshal(data, out)
}

func profileAdminFieldValue(m profileAdminReadModel, key string) any {
	switch key {
	case "hermes.provider":
		return m.Hermes.Provider
	case "hermes.model":
		return m.Hermes.Model
	case "agents.defaults.workspace":
		return m.Agents.Defaults.Workspace
	case "agents.defaults.workspaces":
		return cloneConfigAdminStrings(m.Agents.Defaults.Workspaces)
	default:
		return nil
	}
}

func (b profileAdminBackend) applyProfileChange(profileID string, field configAdminField, change configAdminChange) error {
	cfgPath := b.profileConfigPath(profileID)
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		return err
	}
	value := strings.TrimSpace(change.Value)
	return config.WriteTOMLValue(cfgPath, field.Key, value)
}

func (b profileAdminBackend) validateProfileID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("profile_id is required")
	}
	if !profileIDPattern.MatchString(id) {
		return fmt.Errorf("profile_id %q is not a valid profile identifier", id)
	}
	cfgPath := b.profileConfigPath(id)
	if _, err := os.Stat(cfgPath); err != nil {
		return fmt.Errorf("profile %q not found", id)
	}
	return nil
}

// handleProfileAdmin handles /v1/navivox/profile-admin and
// /v1/navivox/profile-admin/ routes.
func (c *Channel) handleProfileAdmin(w http.ResponseWriter, r *http.Request, _ string) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/navivox/profile-admin")
	path = strings.Trim(path, "/")

	// List: GET /v1/navivox/profile-admin
	if path == "" && r.Method == http.MethodGet {
		profiles, err := c.profileAdmin.listProfiles()
		if err != nil {
			writeNavivoxError(w, http.StatusServiceUnavailable, "", "profile_admin_unavailable", "Profile admin is unavailable")
			return
		}
		if profiles == nil {
			profiles = []profileAdminEntry{}
		}
		writeNavivoxJSON(w, http.StatusOK, map[string]any{"action": "profile_admin.list", "profiles": profiles})
		return
	}

	// Extract profile ID and sub-path from the remainder
	parts := strings.SplitN(path, "/", 2)
	profileID := parts[0]
	subpath := ""
	if len(parts) > 1 {
		subpath = parts[1]
	}

	if profileID == "" {
		writeNavivoxError(w, http.StatusNotFound, "", "not_found", "Profile admin route not found")
		return
	}

	// Schema: GET /v1/navivox/profile-admin/{id}/schema
	if subpath == "schema" && r.Method == http.MethodGet {
		if err := c.profileAdmin.validateProfileID(profileID); err != nil {
			writeNavivoxError(w, http.StatusNotFound, "", "profile_not_found", err.Error())
			return
		}
		writeNavivoxJSON(w, http.StatusOK, map[string]any{
			"action":     "profile_admin.schema",
			"profile_id": profileID,
			"fields":     profileAdminSchema(),
		})
		return
	}

	// Get values: GET /v1/navivox/profile-admin/{id}
	if subpath == "" && r.Method == http.MethodGet {
		if err := c.profileAdmin.validateProfileID(profileID); err != nil {
			writeNavivoxError(w, http.StatusNotFound, "", "profile_not_found", err.Error())
			return
		}
		m, err := c.profileAdmin.readProfileConfig(profileID)
		if err != nil {
			writeNavivoxError(w, http.StatusServiceUnavailable, "", "profile_admin_read_failed", "Could not read profile config")
			return
		}
		fields := profileAdminSchema()
		values := make([]configAdminValueState, 0, len(fields))
		for _, field := range fields {
			values = append(values, configAdminValueState{
				Key:   field.Key,
				Type:  field.Type,
				Value: profileAdminFieldValue(m, field.Key),
			})
		}
		writeNavivoxJSON(w, http.StatusOK, map[string]any{
			"action":      "profile_admin.get",
			"profile_id":  profileID,
			"values":      values,
		})
		return
	}

	// Diff/validate/apply: POST /v1/navivox/profile-admin/{id}/{diff|validate|apply}
	if r.Method == http.MethodPost && (subpath == "diff" || subpath == "validate" || subpath == "apply") {
		if err := c.profileAdmin.validateProfileID(profileID); err != nil {
			writeNavivoxError(w, http.StatusNotFound, "", "profile_not_found", err.Error())
			return
		}
		var req configAdminRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeNavivoxError(w, http.StatusBadRequest, "", "bad_request", "Invalid profile admin request")
			return
		}
		changes := normalizeConfigAdminChanges(req.Changes)
		if len(changes) == 0 {
			writeNavivoxJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"action": "profile_admin." + subpath,
				"valid":  false,
				"errors": []configAdminFieldError{{Key: "changes", Code: "empty", Message: "At least one config change is required."}},
			})
			return
		}
		schema := profileAdminSchema()
		var fieldErrors []configAdminFieldError
		for _, change := range changes {
			field, ok := profileAdminFieldByKey(schema, change.Key)
			if !ok {
				fieldErrors = append(fieldErrors, configAdminFieldError{Key: change.Key, Code: "unsupported", Message: "This config field is not editable through profile admin."})
				continue
			}
			_ = field
		}
		if len(fieldErrors) > 0 {
			writeNavivoxJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"action":     "profile_admin." + subpath,
				"profile_id": profileID,
				"valid":      false,
				"errors":     fieldErrors,
			})
			return
		}

		m, err := c.profileAdmin.readProfileConfig(profileID)
		if err != nil {
			writeNavivoxError(w, http.StatusServiceUnavailable, "", "profile_admin_read_failed", "Could not read profile config")
			return
		}
		diffs := profileAdminDiffs(m, changes, schema)

		if subpath == "diff" || subpath == "validate" {
			writeNavivoxJSON(w, http.StatusOK, map[string]any{
				"action":     "profile_admin." + subpath,
				"profile_id": profileID,
				"valid":      true,
				"changes":    diffs,
			})
			return
		}

		// Apply
		for _, change := range changes {
			field, ok := profileAdminFieldByKey(schema, change.Key)
			if !ok {
				continue
			}
			if err := c.profileAdmin.applyProfileChange(profileID, field, change); err != nil {
				writeNavivoxError(w, http.StatusServiceUnavailable, "", "profile_admin_apply_failed", "Could not apply profile config change")
				return
			}
		}
		writeNavivoxJSON(w, http.StatusOK, map[string]any{
			"action":          "profile_admin.apply",
			"profile_id":      profileID,
			"valid":           true,
			"applied":         true,
			"changes":         diffs,
			"pending_restart": false,
		})
		return
	}

	writeNavivoxError(w, http.StatusNotFound, "", "not_found", "Profile admin route not found")
}

func profileAdminFieldByKey(schema []configAdminField, key string) (configAdminField, bool) {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, f := range schema {
		if f.Key == key {
			return f, true
		}
	}
	return configAdminField{}, false
}

func profileAdminDiffs(m profileAdminReadModel, changes []configAdminChange, schema []configAdminField) []configAdminDiffEntry {
	diffs := make([]configAdminDiffEntry, 0, len(changes))
	for _, change := range changes {
		field, ok := profileAdminFieldByKey(schema, change.Key)
		if !ok {
			continue
		}
		before := profileAdminFieldValue(m, field.Key)
		after := change.Value
		diffs = append(diffs, configAdminDiffEntry{
			Key:    field.Key,
			Type:   field.Type,
			Before: before,
			After:  after,
		})
	}
	return diffs
}
