package navivox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type configAdminBackend struct {
	configPath string
	envPath    string
	load       func() (config.Config, error)
	reload     func(context.Context) error
}

func WithConfigAdminReloader(reload func(context.Context) error) ChannelOption {
	return func(c *Channel) {
		c.configAdmin.reload = reload
	}
}

func defaultConfigAdminBackend() configAdminBackend {
	return configAdminBackend{
		configPath: config.ConfigPath(),
		envPath:    config.EnvPath(),
		load: func() (config.Config, error) {
			return config.Load(nil)
		},
	}
}

type configAdminField struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Secret      bool     `json:"secret"`
	Allowed     []string `json:"allowed,omitempty"`
	Actions     []string `json:"actions,omitempty"`
	Reload      string   `json:"reload"`
}

type configAdminValueState struct {
	Key          string `json:"key"`
	Type         string `json:"type"`
	Value        any    `json:"value,omitempty"`
	Secret       bool   `json:"secret"`
	SecretStatus string `json:"secret_status,omitempty"`
	Source       string `json:"source,omitempty"`
}

type configAdminRequest struct {
	Changes []configAdminChange `json:"changes"`
}

type configAdminChange struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Delete bool   `json:"delete,omitempty"`
}

type configAdminDiffEntry struct {
	Key            string `json:"key"`
	Type           string `json:"type"`
	Secret         bool   `json:"secret"`
	Before         any    `json:"before,omitempty"`
	After          any    `json:"after,omitempty"`
	BeforeRedacted bool   `json:"before_redacted,omitempty"`
	AfterRedacted  bool   `json:"after_redacted,omitempty"`
	SecretStatus   string `json:"secret_status,omitempty"`
}

type configAdminFieldError struct {
	Key     string `json:"key"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type configAdminValidation struct {
	cfg     config.Config
	next    config.Config
	changes []configAdminChange
	diffs   []configAdminDiffEntry
	errors  []configAdminFieldError
}

func (c *Channel) handleConfigAdmin(w http.ResponseWriter, r *http.Request, _ string) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/navivox/config-admin")
	path = strings.Trim(path, "/")
	switch {
	case r.Method == http.MethodGet && path == "schema":
		writeNavivoxJSON(w, http.StatusOK, map[string]any{"action": "config.schema", "fields": configAdminSchema()})
	case r.Method == http.MethodGet && path == "":
		cfg, err := c.configAdmin.loadConfig()
		if err != nil {
			writeNavivoxError(w, http.StatusServiceUnavailable, "", "config_unavailable", "Config admin state is unavailable")
			return
		}
		writeNavivoxJSON(w, http.StatusOK, map[string]any{"action": "config.get", "values": configAdminValues(cfg)})
	case r.Method == http.MethodPost && path == "diff":
		validation, ok := c.validateConfigAdminRequest(w, r)
		if !ok {
			return
		}
		if len(validation.errors) > 0 {
			writeNavivoxJSON(w, http.StatusUnprocessableEntity, map[string]any{"action": "config.diff", "valid": false, "errors": validation.errors})
			return
		}
		writeNavivoxJSON(w, http.StatusOK, map[string]any{"action": "config.diff", "valid": true, "changes": validation.diffs})
	case r.Method == http.MethodPost && path == "validate":
		validation, ok := c.validateConfigAdminRequest(w, r)
		if !ok {
			return
		}
		if len(validation.errors) > 0 {
			writeNavivoxJSON(w, http.StatusUnprocessableEntity, map[string]any{"action": "config.validate", "valid": false, "errors": validation.errors})
			return
		}
		writeNavivoxJSON(w, http.StatusOK, map[string]any{"action": "config.validate", "valid": true, "changes": validation.diffs})
	case r.Method == http.MethodPost && path == "apply":
		validation, ok := c.validateConfigAdminRequest(w, r)
		if !ok {
			return
		}
		if len(validation.errors) > 0 {
			writeNavivoxJSON(w, http.StatusUnprocessableEntity, map[string]any{"action": "config.apply", "applied": false, "valid": false, "errors": validation.errors})
			return
		}
		if err := c.configAdmin.apply(r.Context(), validation); err != nil {
			writeNavivoxError(w, http.StatusServiceUnavailable, "", "config_apply_failed", "Config admin apply failed")
			return
		}
		reloadApplied, pendingRestart, reloadError := c.configAdmin.reloadResult(r.Context())
		payload := map[string]any{
			"action":          "config.apply",
			"valid":           true,
			"applied":         true,
			"changes":         validation.diffs,
			"reload_applied":  reloadApplied,
			"pending_restart": pendingRestart,
		}
		if reloadError != "" {
			payload["reload_error"] = reloadError
		}
		writeNavivoxJSON(w, http.StatusOK, payload)
	case r.Method == http.MethodGet || r.Method == http.MethodPost:
		writeNavivoxError(w, http.StatusNotFound, "", "not_found", "Config admin route not found")
	default:
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
	}
}

func (c *Channel) validateConfigAdminRequest(w http.ResponseWriter, r *http.Request) (configAdminValidation, bool) {
	var req configAdminRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeNavivoxError(w, http.StatusBadRequest, "", "bad_request", "Invalid config admin request")
		return configAdminValidation{}, false
	}
	validation, err := c.configAdmin.validate(req.Changes)
	if err != nil {
		writeNavivoxError(w, http.StatusServiceUnavailable, "", "config_unavailable", "Config admin state is unavailable")
		return configAdminValidation{}, false
	}
	return validation, true
}

func (b configAdminBackend) loadConfig() (config.Config, error) {
	if b.load == nil {
		return config.Load(nil)
	}
	return b.load()
}

func (b configAdminBackend) validate(changes []configAdminChange) (configAdminValidation, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return configAdminValidation{}, err
	}
	next := cfg
	validation := configAdminValidation{cfg: cfg, next: next, changes: normalizeConfigAdminChanges(changes)}
	if len(validation.changes) == 0 {
		validation.errors = append(validation.errors, configAdminFieldError{Key: "changes", Code: "empty", Message: "At least one config change is required."})
		return validation, nil
	}
	for _, change := range validation.changes {
		field, ok := configAdminFieldByKey(change.Key)
		if !ok {
			validation.errors = append(validation.errors, configAdminFieldError{Key: change.Key, Code: "unsupported", Message: "This config field is not editable through Navivox config admin."})
			continue
		}
		if err := applyConfigAdminChangeToConfig(&next, field, change); err != nil {
			validation.errors = append(validation.errors, configAdminFieldError{Key: change.Key, Code: "invalid", Message: err.Error()})
		}
	}
	if len(validation.errors) == 0 {
		if err := config.ValidateNavivoxForRuntime(&next.Navivox); err != nil {
			key := firstConfigAdminChangedKey(validation.changes, "navivox.bind_host")
			validation.errors = append(validation.errors, configAdminFieldError{Key: key, Code: "invalid_runtime", Message: sanitizeConfigAdminMessage(err)})
		}
	}
	validation.next = next
	validation.diffs = configAdminDiffs(cfg, next, validation.changes)
	return validation, nil
}

func (b configAdminBackend) apply(ctx context.Context, validation configAdminValidation) error {
	for _, change := range validation.changes {
		field, ok := configAdminFieldByKey(change.Key)
		if !ok {
			continue
		}
		if field.Secret {
			envName := config.SecretEnvName(change.Key)
			value := change.Value
			if change.Delete {
				value = ""
			}
			if err := config.WriteEnvValue(b.envPathOrDefault(), envName, value); err != nil {
				return err
			}
			if change.Delete {
				_ = os.Unsetenv(envName)
			} else {
				_ = os.Setenv(envName, value)
			}
			continue
		}
		if err := config.WriteTOMLValue(b.configPathOrDefault(), change.Key, change.Value); err != nil {
			return err
		}
	}
	_ = ctx
	return nil
}

func (b configAdminBackend) reloadResult(ctx context.Context) (reloadApplied bool, pendingRestart bool, sanitizedError string) {
	if b.reload == nil {
		return false, true, ""
	}
	if err := b.reload(ctx); err != nil {
		return false, true, sanitizeConfigAdminMessage(err)
	}
	return true, false, ""
}

func (b configAdminBackend) configPathOrDefault() string {
	if strings.TrimSpace(b.configPath) != "" {
		return b.configPath
	}
	return config.ConfigPath()
}

func (b configAdminBackend) envPathOrDefault() string {
	if strings.TrimSpace(b.envPath) != "" {
		return b.envPath
	}
	return config.EnvPath()
}

func configAdminSchema() []configAdminField {
	return []configAdminField{
		{Key: "navivox.enabled", Type: "bool", Title: "Enable Navivox", Reload: "restart_or_reload"},
		{Key: "navivox.bind_host", Type: "string", Title: "Bind host", Reload: "restart_or_reload"},
		{Key: "navivox.port", Type: "int", Title: "Port", Reload: "restart_or_reload"},
		{Key: "navivox.exposure_mode", Type: "enum", Title: "Exposure mode", Allowed: []string{config.NavivoxExposureLocal, config.NavivoxExposureTailscale, config.NavivoxExposureWireGuard, config.NavivoxExposureVPN, config.NavivoxExposurePublic}, Reload: "restart_or_reload"},
		{Key: "navivox.auth_mode", Type: "enum", Title: "Auth mode", Allowed: []string{config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken, config.NavivoxAuthTailscaleIdentity, config.NavivoxAuthTokenAndTailscaleIdentity}, Reload: "restart_or_reload"},
		{Key: "navivox.allow_origins", Type: "string_list", Title: "Allowed browser origins", Reload: "restart_or_reload"},
		{Key: "navivox.allowed_tailnet_identities", Type: "string_list", Title: "Allowed tailnet identities", Reload: "restart_or_reload"},
		{Key: "navivox.public_confirmed", Type: "bool", Title: "Public exposure confirmed", Reload: "restart_or_reload"},
		{Key: "navivox.token", Type: "secret", Title: "Pairing/static token", Secret: true, Actions: []string{"set", "rotate", "delete", "test"}, Reload: "restart_or_reload"},
	}
}

func configAdminFieldByKey(key string) (configAdminField, bool) {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, field := range configAdminSchema() {
		if field.Key == key {
			return field, true
		}
	}
	return configAdminField{}, false
}

func configAdminValues(cfg config.Config) []configAdminValueState {
	fields := configAdminSchema()
	values := make([]configAdminValueState, 0, len(fields))
	for _, field := range fields {
		state := configAdminValueState{Key: field.Key, Type: field.Type, Secret: field.Secret}
		if field.Secret {
			state.SecretStatus = "unset"
			if strings.TrimSpace(secretValueForConfigAdmin(cfg, field.Key)) != "" {
				state.SecretStatus = "set"
				state.Source = secretSourceForConfigAdmin(field.Key)
			}
		} else {
			state.Value = configAdminFieldValue(cfg, field.Key)
		}
		values = append(values, state)
	}
	return values
}

func configAdminFieldValue(cfg config.Config, key string) any {
	switch key {
	case "navivox.enabled":
		return cfg.Navivox.Enabled
	case "navivox.bind_host":
		return cfg.Navivox.BindHost
	case "navivox.port":
		return cfg.Navivox.Port
	case "navivox.exposure_mode":
		return cfg.Navivox.ExposureMode
	case "navivox.auth_mode":
		return cfg.Navivox.AuthMode
	case "navivox.allow_origins":
		return cloneConfigAdminStrings(cfg.Navivox.AllowOrigins)
	case "navivox.allowed_tailnet_identities":
		return cloneConfigAdminStrings(cfg.Navivox.AllowedTailnetIdentities)
	case "navivox.public_confirmed":
		return cfg.Navivox.PublicConfirmed
	default:
		return nil
	}
}

func secretValueForConfigAdmin(cfg config.Config, key string) string {
	switch key {
	case "navivox.token":
		return cfg.Navivox.Token
	default:
		return ""
	}
}

func secretSourceForConfigAdmin(key string) string {
	envName := config.SecretEnvName(key)
	if strings.TrimSpace(os.Getenv(envName)) != "" {
		return "env:" + envName
	}
	return "config:redacted"
}

func applyConfigAdminChangeToConfig(cfg *config.Config, field configAdminField, change configAdminChange) error {
	if field.Secret {
		return nil
	}
	value := strings.TrimSpace(change.Value)
	switch field.Key {
	case "navivox.enabled":
		parsed, err := parseConfigAdminBool(value)
		if err != nil {
			return err
		}
		cfg.Navivox.Enabled = parsed
	case "navivox.bind_host":
		if value == "" {
			return fmt.Errorf("bind_host cannot be empty")
		}
		cfg.Navivox.BindHost = value
	case "navivox.port":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 65535 {
			return fmt.Errorf("port must be an integer from 1 to 65535")
		}
		cfg.Navivox.Port = parsed
	case "navivox.exposure_mode":
		if !containsConfigAdminAllowed(field.Allowed, value) {
			return fmt.Errorf("exposure_mode must be one of %s", strings.Join(field.Allowed, ", "))
		}
		cfg.Navivox.ExposureMode = value
	case "navivox.auth_mode":
		if !containsConfigAdminAllowed(field.Allowed, value) {
			return fmt.Errorf("auth_mode must be one of %s", strings.Join(field.Allowed, ", "))
		}
		cfg.Navivox.AuthMode = value
	case "navivox.allow_origins":
		cfg.Navivox.AllowOrigins = parseConfigAdminCSV(value)
	case "navivox.allowed_tailnet_identities":
		cfg.Navivox.AllowedTailnetIdentities = parseConfigAdminCSV(value)
	case "navivox.public_confirmed":
		parsed, err := parseConfigAdminBool(value)
		if err != nil {
			return err
		}
		cfg.Navivox.PublicConfirmed = parsed
	}
	return nil
}

func configAdminDiffs(before, after config.Config, changes []configAdminChange) []configAdminDiffEntry {
	diffs := make([]configAdminDiffEntry, 0, len(changes))
	for _, change := range changes {
		field, ok := configAdminFieldByKey(change.Key)
		if !ok {
			continue
		}
		diff := configAdminDiffEntry{Key: field.Key, Type: field.Type, Secret: field.Secret}
		if field.Secret {
			diff.BeforeRedacted = true
			diff.AfterRedacted = true
			if change.Delete {
				diff.SecretStatus = "unset"
			} else if strings.TrimSpace(change.Value) != "" {
				diff.SecretStatus = "set"
			} else {
				diff.SecretStatus = secretStatusForConfigAdmin(before, field.Key)
			}
		} else {
			diff.Before = configAdminFieldValue(before, field.Key)
			diff.After = configAdminFieldValue(after, field.Key)
		}
		diffs = append(diffs, diff)
	}
	return diffs
}

func secretStatusForConfigAdmin(cfg config.Config, key string) string {
	if strings.TrimSpace(secretValueForConfigAdmin(cfg, key)) != "" {
		return "set"
	}
	return "unset"
}

func normalizeConfigAdminChanges(changes []configAdminChange) []configAdminChange {
	out := make([]configAdminChange, 0, len(changes))
	for _, change := range changes {
		change.Key = strings.ToLower(strings.TrimSpace(change.Key))
		change.Value = strings.TrimSpace(change.Value)
		if change.Key == "" {
			change.Key = "<empty>"
		}
		out = append(out, change)
	}
	return out
}

func cloneConfigAdminStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func parseConfigAdminBool(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "yes", "on", "1":
		return true, nil
	case "false", "no", "off", "0":
		return false, nil
	default:
		return false, fmt.Errorf("value must be true or false")
	}
}

func parseConfigAdminCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == "[]" {
		return nil
	}
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, `"'`))
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out
}

func containsConfigAdminAllowed(allowed []string, value string) bool {
	return channelutil.ContainsString(allowed, value)
}

func firstConfigAdminChangedKey(changes []configAdminChange, preferred string) string {
	for _, change := range changes {
		if change.Key == preferred {
			return preferred
		}
	}
	if len(changes) > 0 {
		return changes[0].Key
	}
	return preferred
}

func sanitizeConfigAdminMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return ""
	}
	lower := strings.ToLower(msg)
	for _, hint := range []string{"api_key", "token", "authorization", "bearer ", "secret", "password"} {
		if strings.Contains(lower, hint) {
			return "[redacted]"
		}
	}
	if len(msg) > 240 {
		return msg[:240]
	}
	return msg
}
