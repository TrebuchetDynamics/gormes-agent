package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/redaction"
	"gopkg.in/yaml.v3"
)

const (
	defaultMCPToolTimeout    = 120 * time.Second
	defaultMCPConnectTimeout = 60 * time.Second
	defaultMCPSamplingTime   = 30 * time.Second

	// RedactedMCPConfigValue is the public placeholder used in MCP status
	// surfaces when config contains credentials or token-shaped values.
	RedactedMCPConfigValue = redaction.Value
)

// MCPTransport identifies the transport a resolved MCP server would use.
type MCPTransport string

const (
	MCPTransportStdio MCPTransport = "stdio"
	MCPTransportHTTP  MCPTransport = "http"
)

// MCPConfigStatus is the degraded-mode state for one configured server.
type MCPConfigStatus string

const (
	MCPConfigStatusReady            MCPConfigStatus = "ready"
	MCPConfigStatusDisabled         MCPConfigStatus = "disabled"
	MCPConfigStatusMissingSDK       MCPConfigStatus = "missing_sdk"
	MCPConfigStatusInvalidTransport MCPConfigStatus = "invalid_transport"
	MCPConfigStatusInvalidEnv       MCPConfigStatus = "invalid_env"
	MCPConfigStatusInvalidConfig    MCPConfigStatus = "invalid_config"
)

// MCPConfigOptions controls pure MCP config resolution.
type MCPConfigOptions struct {
	LookupEnv func(string) (string, bool)
	// RuntimeAvailable lets callers report missing MCP runtime/SDK support
	// without importing that runtime in the config resolver.
	RuntimeAvailable   *bool
	RuntimeUnavailable string
}

// MCPConfigResolution is the safe, typed MCP config surface. Servers is empty
// when any config error is present so callers cannot launch a partial set.
type MCPConfigResolution struct {
	Servers  []MCPServerDefinition
	Statuses []MCPServerStatus
}

// MCPServerDefinition is a validated MCP server configuration. It contains
// resolved credentials because runtime transport code needs them; use Statuses
// or RedactedStatusText for operator-visible output.
type MCPServerDefinition struct {
	Name           string
	Enabled        bool
	Transport      MCPTransport
	Command        string
	Args           []string
	Env            map[string]string
	URL            string
	Headers        map[string]string
	Timeout        time.Duration
	ConnectTimeout time.Duration
	Sampling       MCPSamplingConfig
}

// MCPSamplingConfig captures server-initiated sampling limits without
// creating any MCP SDK objects.
type MCPSamplingConfig struct {
	Enabled       bool
	Model         string
	MaxTokensCap  int
	Timeout       time.Duration
	MaxRPM        int
	AllowedModels []string
	MaxToolRounds int
	LogLevel      string
}

// MCPServerStatus is the redacted config/status view used before runtime
// connection attempts.
type MCPServerStatus struct {
	Name           string
	Enabled        bool
	Status         MCPConfigStatus
	Reason         string
	Transport      MCPTransport
	Command        string
	Args           []string
	Env            map[string]string
	URL            string
	Headers        map[string]string
	Timeout        time.Duration
	ConnectTimeout time.Duration
	Sampling       MCPSamplingConfig
}

type mcpConfigIssue struct {
	server  string
	status  MCPConfigStatus
	message string
}

// MCPConfigError reports one or more config issues with credentials redacted.
type MCPConfigError struct {
	Issues []mcpConfigIssue
}

func (e *MCPConfigError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "mcp config: invalid"
	}
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.server, redaction.String(issue.message)))
	}
	return "mcp config: " + strings.Join(parts, "; ")
}

// ParseMCPConfigYAML parses a Hermes-compatible YAML document with a
// top-level mcp_servers section.
func ParseMCPConfigYAML(data []byte, opts MCPConfigOptions) (MCPConfigResolution, error) {
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return MCPConfigResolution{}, fmt.Errorf("mcp config yaml: %w", err)
	}
	return ResolveMCPConfig(raw, opts)
}

// ParseMCPConfigJSON parses a Hermes-compatible JSON document with a
// top-level mcp_servers section.
func ParseMCPConfigJSON(data []byte, opts MCPConfigOptions) (MCPConfigResolution, error) {
	raw, err := decodeMCPConfigJSONDocument(data)
	if err != nil {
		return MCPConfigResolution{}, err
	}
	return ResolveMCPConfig(raw, opts)
}

func decodeMCPConfigJSONDocument(data []byte) (any, error) {
	var raw any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("mcp config json: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("mcp config json: trailing content after first document")
		}
		return nil, fmt.Errorf("mcp config json: trailing content after first document: %w", err)
	}
	return raw, nil
}

// ResolveMCPConfig turns an in-memory config document into safe server
// definitions without importing an MCP SDK, spawning processes, or opening
// transports.
func ResolveMCPConfig(raw any, opts MCPConfigOptions) (MCPConfigResolution, error) {
	lookupEnv := opts.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}

	root := mcpMap(raw)
	if root == nil {
		return MCPConfigResolution{}, nil
	}
	block := lookupMCPServersBlock(root)
	if block.Conflict.HasConflict() {
		reason := block.Conflict.Reason()
		status := MCPServerStatus{
			Name:   "mcp_servers",
			Status: MCPConfigStatusInvalidConfig,
			Reason: redaction.String(reason),
		}
		issue := mcpConfigIssue{server: "mcp_servers", status: MCPConfigStatusInvalidConfig, message: reason}
		return MCPConfigResolution{Statuses: []MCPServerStatus{status}}, &MCPConfigError{Issues: []mcpConfigIssue{issue}}
	}
	if !block.Found {
		return MCPConfigResolution{}, nil
	}
	serversMap := mcpMap(block.Value)
	if serversMap == nil {
		status := MCPServerStatus{
			Name:   "mcp_servers",
			Status: MCPConfigStatusInvalidConfig,
			Reason: "mcp_servers must be a map",
		}
		issue := mcpConfigIssue{server: "mcp_servers", status: MCPConfigStatusInvalidConfig, message: status.Reason}
		return MCPConfigResolution{Statuses: []MCPServerStatus{status}}, &MCPConfigError{Issues: []mcpConfigIssue{issue}}
	}

	names := sortedMCPKeys(serversMap)
	definitions := make([]MCPServerDefinition, 0, len(names))
	statuses := make([]MCPServerStatus, 0, len(names))
	var issues []mcpConfigIssue
	for _, name := range names {
		def, status, issue := resolveMCPServer(name, serversMap[name], lookupEnv)
		statuses = append(statuses, status)
		if issue != nil {
			issues = append(issues, *issue)
			continue
		}
		definitions = append(definitions, def)
	}

	if len(issues) > 0 {
		return MCPConfigResolution{Statuses: statuses}, &MCPConfigError{Issues: issues}
	}
	if opts.RuntimeAvailable != nil && !*opts.RuntimeAvailable {
		reason := strings.TrimSpace(opts.RuntimeUnavailable)
		if reason == "" {
			reason = "MCP runtime unavailable"
		}
		var runtimeIssues []mcpConfigIssue
		for i := range statuses {
			if statuses[i].Status != MCPConfigStatusReady {
				continue
			}
			statuses[i].Status = MCPConfigStatusMissingSDK
			statuses[i].Reason = redaction.String(reason)
			runtimeIssues = append(runtimeIssues, mcpConfigIssue{
				server:  statuses[i].Name,
				status:  MCPConfigStatusMissingSDK,
				message: reason,
			})
		}
		if len(runtimeIssues) > 0 {
			return MCPConfigResolution{Statuses: statuses}, &MCPConfigError{Issues: runtimeIssues}
		}
	}
	return MCPConfigResolution{Servers: definitions, Statuses: statuses}, nil
}

func resolveMCPServer(name string, raw any, lookupEnv func(string) (string, bool)) (MCPServerDefinition, MCPServerStatus, *mcpConfigIssue) {
	server := mcpMap(raw)
	baseStatus := MCPServerStatus{
		Name:           name,
		Enabled:        true,
		Timeout:        defaultMCPToolTimeout,
		ConnectTimeout: defaultMCPConnectTimeout,
		Sampling:       defaultMCPSamplingConfig(),
	}
	if server == nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidConfig, "server config must be a map"), &mcpConfigIssue{
			server:  name,
			status:  MCPConfigStatusInvalidConfig,
			message: "server config must be a map",
		}
	}

	enabled, issue := resolveMCPEnabled(name, server, baseStatus)
	if issue != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, issue.status, issue.message), issue
	}
	baseStatus.Enabled = enabled
	if !enabled {
		return disabledMCPServer(name, baseStatus)
	}

	if field, variants, ok := ambiguousActiveMCPServerField(server); ok {
		reason := fmt.Sprintf("ambiguous %s field variants: %s", field, strings.Join(variants, ", "))
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidConfig, reason), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidConfig, message: reason}
	}

	command, err := mcpOptionalString(server, "command", lookupEnv)
	if err != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidEnv, err.Error()), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidEnv, message: err.Error()}
	}
	url, err := mcpOptionalString(server, "url", lookupEnv)
	if err != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidEnv, err.Error()), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidEnv, message: err.Error()}
	}
	command = strings.TrimSpace(command)
	url = strings.TrimSpace(url)
	transport, err := resolveMCPTransport(command, url)
	if err != nil {
		reason := err.Error()
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidTransport, reason), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidTransport, message: reason}
	}

	args, err := mcpStringList(mcpValue(server, "args"), "args", lookupEnv)
	if err != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidEnv, err.Error()), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidEnv, message: err.Error()}
	}
	env, err := mcpStringMap(mcpValue(server, "env"), "env", true, lookupEnv)
	if err != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidEnv, err.Error()), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidEnv, message: err.Error()}
	}
	headers, err := mcpStringMap(mcpValue(server, "headers"), "headers", false, lookupEnv)
	if err != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidEnv, err.Error()), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidEnv, message: err.Error()}
	}
	timeout, err := mcpDuration(mcpValue(server, "timeout"), defaultMCPToolTimeout, "timeout")
	if err != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidConfig, err.Error()), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidConfig, message: err.Error()}
	}
	connectTimeout, err := mcpDuration(mcpValue(server, "connect_timeout"), defaultMCPConnectTimeout, "connect_timeout")
	if err != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidConfig, err.Error()), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidConfig, message: err.Error()}
	}
	sampling, err := mcpSamplingConfig(mcpValue(server, "sampling"), lookupEnv)
	if err != nil {
		return MCPServerDefinition{}, invalidMCPStatus(baseStatus, MCPConfigStatusInvalidConfig, err.Error()), &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidConfig, message: err.Error()}
	}

	def := MCPServerDefinition{
		Name:           name,
		Enabled:        enabled,
		Transport:      transport,
		Command:        command,
		Args:           args,
		Env:            env,
		URL:            url,
		Headers:        headers,
		Timeout:        timeout,
		ConnectTimeout: connectTimeout,
		Sampling:       sampling,
	}
	status := redactedMCPStatus(def, MCPConfigStatusReady, "")
	if !enabled {
		status.Status = MCPConfigStatusDisabled
		status.Reason = "server disabled by config"
	}
	return def, status, nil
}

func resolveMCPEnabled(name string, server map[string]any, baseStatus MCPServerStatus) (bool, *mcpConfigIssue) {
	if field, variants, ok := ambiguousCaseFoldedMCPField(server, "enabled"); ok {
		reason := fmt.Sprintf("ambiguous %s field variants: %s", field, strings.Join(variants, ", "))
		return false, &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidConfig, message: reason}
	}
	enabled, err := parseMCPBoolField(mcpValue(server, "enabled"), baseStatus.Enabled, "enabled")
	if err != nil {
		return false, &mcpConfigIssue{server: name, status: MCPConfigStatusInvalidConfig, message: err.Error()}
	}
	return enabled, nil
}

func ambiguousActiveMCPServerField(server map[string]any) (string, []string, bool) {
	return ambiguousCaseFoldedMCPField(server, "command", "url", "args", "env", "headers", "timeout", "connect_timeout", "sampling")
}

func resolveMCPTransport(command, rawURL string) (MCPTransport, error) {
	hasCommand := command != ""
	hasURL := rawURL != ""
	switch {
	case hasCommand && hasURL:
		return "", fmt.Errorf("server has both command and url; choose exactly one transport")
	case !hasCommand && !hasURL:
		return "", fmt.Errorf("server requires command or url")
	case hasURL:
		if err := validateMCPHTTPURL(rawURL); err != nil {
			return "", err
		}
		return MCPTransportHTTP, nil
	default:
		return MCPTransportStdio, nil
	}
}

func validateMCPHTTPURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("url is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if !hasMCPHTTPHostname(parsed) {
		return fmt.Errorf("url must include a host")
	}
	return nil
}

func hasMCPHTTPHostname(parsed *url.URL) bool {
	return parsed != nil && strings.TrimSpace(parsed.Hostname()) != ""
}

func disabledMCPServer(name string, baseStatus MCPServerStatus) (MCPServerDefinition, MCPServerStatus, *mcpConfigIssue) {
	def := MCPServerDefinition{
		Name:           name,
		Enabled:        false,
		Timeout:        baseStatus.Timeout,
		ConnectTimeout: baseStatus.ConnectTimeout,
		Sampling:       baseStatus.Sampling,
	}
	status := redactedMCPStatus(def, MCPConfigStatusDisabled, "server disabled by config")
	return def, status, nil
}

func invalidMCPStatus(status MCPServerStatus, kind MCPConfigStatus, reason string) MCPServerStatus {
	status.Status = kind
	status.Reason = redaction.String(reason)
	if status.Env == nil {
		status.Env = map[string]string{}
	}
	if status.Headers == nil {
		status.Headers = map[string]string{}
	}
	return status
}

func redactedMCPStatus(def MCPServerDefinition, status MCPConfigStatus, reason string) MCPServerStatus {
	return MCPServerStatus{
		Name:           def.Name,
		Enabled:        def.Enabled,
		Status:         status,
		Reason:         redaction.String(reason),
		Transport:      def.Transport,
		Command:        def.Command,
		Args:           append([]string(nil), def.Args...),
		Env:            redaction.Map(def.Env),
		URL:            redactedMCPURL(def.URL),
		Headers:        redaction.Map(def.Headers),
		Timeout:        def.Timeout,
		ConnectTimeout: def.ConnectTimeout,
		Sampling:       def.Sampling,
	}
}

func redactedMCPURL(raw string) string {
	return redaction.String(redactMCPURLUserinfo(raw))
}

func redactMCPURLUserinfo(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return raw
	}
	normalized := parsed.String()
	userinfo := parsed.User.String()
	if userinfo == "" {
		return raw
	}
	return strings.Replace(normalized, "//"+userinfo+"@", "//"+RedactedMCPConfigValue+"@", 1)
}

// Server returns the resolved server definition by name.
func (r MCPConfigResolution) Server(name string) (MCPServerDefinition, bool) {
	for _, server := range r.Servers {
		if server.Name == name {
			return server, true
		}
	}
	return MCPServerDefinition{}, false
}

// Status returns the redacted status row by name.
func (r MCPConfigResolution) Status(name string) (MCPServerStatus, bool) {
	for _, status := range r.Statuses {
		if status.Name == name {
			return status, true
		}
	}
	return MCPServerStatus{}, false
}

// RedactedStatusText renders a stable operator-facing status string with all
// token-shaped config values redacted.
func (r MCPConfigResolution) RedactedStatusText() string {
	rows := append([]MCPServerStatus(nil), r.Statuses...)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		fields := []string{
			"name=" + row.Name,
			"status=" + string(row.Status),
		}
		if row.Transport != "" {
			fields = append(fields, "transport="+string(row.Transport))
		}
		if row.Command != "" {
			fields = append(fields, "command="+row.Command)
		}
		if row.URL != "" {
			fields = append(fields, "url="+redactedMCPURL(row.URL))
		}
		if len(row.Headers) > 0 {
			fields = append(fields, "headers="+redaction.FormatStringMap(row.Headers))
		}
		if len(row.Env) > 0 {
			fields = append(fields, "env="+redaction.FormatStringMap(row.Env))
		}
		if row.Reason != "" {
			fields = append(fields, "reason="+redaction.String(row.Reason))
		}
		parts = append(parts, strings.Join(fields, " "))
	}
	return strings.Join(parts, "\n")
}

func defaultMCPSamplingConfig() MCPSamplingConfig {
	return MCPSamplingConfig{
		Enabled:       true,
		MaxTokensCap:  4096,
		Timeout:       defaultMCPSamplingTime,
		MaxRPM:        10,
		MaxToolRounds: 5,
		LogLevel:      "info",
	}
}

func mcpSamplingConfig(raw any, lookupEnv func(string) (string, bool)) (MCPSamplingConfig, error) {
	cfg := defaultMCPSamplingConfig()
	if raw == nil {
		return cfg, nil
	}
	values := mcpMap(raw)
	if values == nil {
		return cfg, fmt.Errorf("sampling must be a map")
	}
	if field, variants, ok := ambiguousCaseFoldedMCPField(values, "enabled", "model", "max_tokens_cap", "timeout", "max_rpm", "allowed_models", "max_tool_rounds", "log_level"); ok {
		return cfg, fmt.Errorf("ambiguous sampling.%s field variants: %s", field, strings.Join(variants, ", "))
	}
	if rawEnabled, ok := lookupMCPValue(values, "enabled"); ok {
		parsed, err := parseMCPBoolField(rawEnabled, cfg.Enabled, "sampling.enabled")
		if err != nil {
			return cfg, err
		}
		cfg.Enabled = parsed
	}
	if rawModel, ok := lookupMCPValue(values, "model"); ok {
		model, err := mcpStringValue(rawModel, "sampling.model", lookupEnv)
		if err != nil {
			return cfg, err
		}
		cfg.Model = model
	}
	if rawCap, ok := lookupMCPValue(values, "max_tokens_cap"); ok {
		parsed, err := mcpInt(rawCap, cfg.MaxTokensCap, "sampling.max_tokens_cap", 0)
		if err != nil {
			return cfg, err
		}
		cfg.MaxTokensCap = parsed
	}
	if rawTimeout, ok := lookupMCPValue(values, "timeout"); ok {
		parsed, err := mcpDuration(rawTimeout, cfg.Timeout, "sampling.timeout")
		if err != nil {
			return cfg, err
		}
		cfg.Timeout = parsed
	}
	if rawRPM, ok := lookupMCPValue(values, "max_rpm"); ok {
		parsed, err := mcpInt(rawRPM, cfg.MaxRPM, "sampling.max_rpm", 1)
		if err != nil {
			return cfg, err
		}
		cfg.MaxRPM = parsed
	}
	if rawModels, ok := lookupMCPValue(values, "allowed_models"); ok {
		parsed, err := mcpStringList(rawModels, "sampling.allowed_models", lookupEnv)
		if err != nil {
			return cfg, err
		}
		cfg.AllowedModels = parsed
	}
	if rawRounds, ok := lookupMCPValue(values, "max_tool_rounds"); ok {
		parsed, err := mcpInt(rawRounds, cfg.MaxToolRounds, "sampling.max_tool_rounds", 0)
		if err != nil {
			return cfg, err
		}
		cfg.MaxToolRounds = parsed
	}
	if rawLevel, ok := lookupMCPValue(values, "log_level"); ok {
		parsed, err := mcpStringValue(rawLevel, "sampling.log_level", lookupEnv)
		if err != nil {
			return cfg, err
		}
		cfg.LogLevel = strings.ToLower(strings.TrimSpace(parsed))
	}
	return cfg, nil
}

func mcpMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[any]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[fmt.Sprint(key)] = value
		}
		return out
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = value
		}
		return out
	default:
		return nil
	}
}

type mcpServerBlockCandidate struct {
	Value    any
	Found    bool
	Conflict mcpFieldConflict
}

type mcpFieldConflict struct {
	Field    string
	Variants []string
}

func (c mcpFieldConflict) HasConflict() bool {
	return len(c.Variants) > 1
}

func (c mcpFieldConflict) Reason() string {
	return fmt.Sprintf("ambiguous %s: %s", c.Field, strings.Join(c.Variants, ", "))
}

func lookupMCPServersBlock(root map[string]any) mcpServerBlockCandidate {
	variants := mcpServerBlockVariants(root)
	if len(variants) == 0 {
		return mcpServerBlockCandidate{}
	}
	if len(variants) > 1 {
		return mcpServerBlockCandidate{Conflict: mcpFieldConflict{Field: "mcp server block fields", Variants: variants}}
	}
	return mcpServerBlockCandidate{Value: root[variants[0]], Found: true}
}

func mcpServerBlockVariants(root map[string]any) []string {
	variants := make([]string, 0, 2)
	for key := range root {
		if strings.EqualFold(key, "mcp_servers") || strings.EqualFold(key, "mcpServers") {
			variants = append(variants, key)
		}
	}
	sort.Strings(variants)
	return variants
}

func sortedMCPKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func lookupMCPValue(values map[string]any, name string) (any, bool) {
	value, ok := values[name]
	if ok {
		return value, true
	}
	for key, value := range values {
		if strings.EqualFold(key, name) {
			return value, true
		}
	}
	return nil, false
}

func ambiguousCaseFoldedMCPField(values map[string]any, fields ...string) (string, []string, bool) {
	for _, field := range fields {
		variants := caseFoldedMCPFieldVariants(values, field)
		if len(variants) > 1 {
			return field, variants, true
		}
	}
	return "", nil, false
}

func caseFoldedMCPFieldVariants(values map[string]any, field string) []string {
	variants := make([]string, 0, 2)
	for key := range values {
		if strings.EqualFold(key, field) {
			variants = append(variants, key)
		}
	}
	sort.Strings(variants)
	return variants
}

func mcpValue(values map[string]any, name string) any {
	value, _ := lookupMCPValue(values, name)
	return value
}

func mcpOptionalString(values map[string]any, name string, lookupEnv func(string) (string, bool)) (string, error) {
	raw, ok := lookupMCPValue(values, name)
	if !ok || raw == nil {
		return "", nil
	}
	return mcpStringValue(raw, name, lookupEnv)
}

func mcpStringValue(value any, field string, lookupEnv func(string) (string, bool)) (string, error) {
	var raw string
	switch typed := value.(type) {
	case string:
		raw = typed
	case fmt.Stringer:
		raw = typed.String()
	case json.Number:
		raw = typed.String()
	case int, int64, int32, float64, float32, bool:
		raw = fmt.Sprint(typed)
	default:
		return "", fmt.Errorf("%s must be a string", field)
	}
	return interpolateMCPEnv(raw, lookupEnv)
}

func mcpStringList(value any, field string, lookupEnv func(string) (string, bool)) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	var raw []any
	switch typed := value.(type) {
	case []any:
		raw = typed
	case []string:
		raw = make([]any, len(typed))
		for i, item := range typed {
			raw[i] = item
		}
	default:
		return nil, fmt.Errorf("%s must be a list of strings", field)
	}
	out := make([]string, 0, len(raw))
	for i, item := range raw {
		parsed, err := mcpStringValue(item, fmt.Sprintf("%s[%d]", field, i), lookupEnv)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	return out, nil
}

var mcpEnvNameRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func mcpStringMap(value any, field string, validateKeys bool, lookupEnv func(string) (string, bool)) (map[string]string, error) {
	if value == nil {
		return map[string]string{}, nil
	}
	values := mcpMap(value)
	if values == nil {
		return nil, fmt.Errorf("%s must be a map", field)
	}
	out := make(map[string]string, len(values))
	for _, key := range sortedMCPKeys(values) {
		if validateKeys && !mcpEnvNameRE.MatchString(key) {
			return nil, fmt.Errorf("invalid env variable name %q", key)
		}
		parsed, err := mcpStringValue(values[key], field+"."+key, lookupEnv)
		if err != nil {
			return nil, err
		}
		out[key] = parsed
	}
	return out, nil
}

func interpolateMCPEnv(value string, lookupEnv func(string) (string, bool)) (string, error) {
	var resolved strings.Builder
	for i := 0; i < len(value); {
		if !startsMCPEnvReference(value, i) {
			resolved.WriteByte(value[i])
			i++
			continue
		}

		ref, err := parseMCPEnvReferenceAt(value, i)
		if err != nil {
			return "", err
		}
		envValue, ok := lookupEnv(ref.Name)
		if !ok {
			return "", fmt.Errorf("missing environment variable %s", ref.Name)
		}
		resolved.WriteString(envValue)
		i = ref.End
	}
	return resolved.String(), nil
}

type mcpEnvReference struct {
	Name string
	End  int
}

func startsMCPEnvReference(value string, offset int) bool {
	return offset+1 < len(value) && value[offset] == '$' && value[offset+1] == '{'
}

func parseMCPEnvReferenceAt(value string, offset int) (mcpEnvReference, error) {
	end := strings.IndexByte(value[offset+2:], '}')
	if end < 0 {
		return mcpEnvReference{}, fmt.Errorf("unterminated environment variable reference")
	}
	matchEnd := offset + 2 + end + 1
	name, err := mcpEnvReferenceName(value[offset:matchEnd])
	if err != nil {
		return mcpEnvReference{}, err
	}
	return mcpEnvReference{Name: name, End: matchEnd}, nil
}

func mcpEnvReferenceName(match string) (string, error) {
	name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}"))
	if !mcpEnvNameRE.MatchString(name) {
		return "", fmt.Errorf("invalid env variable name %q", name)
	}
	return name, nil
}

func parseMCPBool(value any, fallback bool) bool {
	parsed, err := parseMCPBoolField(value, fallback, "boolean")
	if err != nil {
		return fallback
	}
	return parsed
}

func parseMCPBoolField(value any, fallback bool, field string) (bool, error) {
	switch typed := value.(type) {
	case nil:
		return fallback, nil
	case bool:
		return typed, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true, nil
		case "false", "0", "no", "off":
			return false, nil
		default:
			return fallback, fmt.Errorf("%s must be a boolean", field)
		}
	case int:
		return typed != 0, nil
	case int64:
		return typed != 0, nil
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return fallback, fmt.Errorf("%s must be a boolean", field)
		}
		return typed != 0, nil
	case json.Number:
		return mcpJSONNumberBool(typed, fallback, field)
	default:
		return fallback, fmt.Errorf("%s must be a boolean", field)
	}
}

func mcpJSONNumberBool(value json.Number, fallback bool, field string) (bool, error) {
	parsed, err := value.Float64()
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return fallback, fmt.Errorf("%s must be a boolean", field)
	}
	return parsed != 0, nil
}

func mcpDuration(value any, fallback time.Duration, field string) (time.Duration, error) {
	if value == nil {
		return fallback, nil
	}
	switch typed := value.(type) {
	case time.Duration:
		if typed <= 0 {
			return 0, fmt.Errorf("%s must be positive", field)
		}
		return typed, nil
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return fallback, nil
		}
		if dur, err := time.ParseDuration(text); err == nil {
			if dur <= 0 {
				return 0, fmt.Errorf("%s must be positive", field)
			}
			return dur, nil
		}
		seconds, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be seconds or a duration", field)
		}
		return secondsDuration(seconds, field)
	case int:
		return secondsDuration(float64(typed), field)
	case int64:
		return secondsDuration(float64(typed), field)
	case int32:
		return secondsDuration(float64(typed), field)
	case float64:
		return secondsDuration(typed, field)
	case float32:
		return secondsDuration(float64(typed), field)
	case json.Number:
		seconds, err := typed.Float64()
		if err != nil {
			return 0, fmt.Errorf("%s must be numeric seconds", field)
		}
		return secondsDuration(seconds, field)
	default:
		return 0, fmt.Errorf("%s must be seconds or a duration", field)
	}
}

func secondsDuration(seconds float64, field string) (time.Duration, error) {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
		return 0, fmt.Errorf("%s must be positive", field)
	}
	if seconds > maxMCPDurationSeconds() {
		return 0, fmt.Errorf("%s is too large", field)
	}
	duration := time.Duration(seconds * float64(time.Second))
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be at least 1ns", field)
	}
	return duration, nil
}

func maxMCPDurationSeconds() float64 {
	return float64(time.Duration(1<<63-1)) / float64(time.Second)
}

func mcpInt(value any, fallback int, field string, minimum int) (int, error) {
	if value == nil {
		return fallback, nil
	}
	var parsed int64
	switch typed := value.(type) {
	case int:
		parsed = int64(typed)
	case int64:
		parsed = typed
	case int32:
		parsed = int64(typed)
	case float64:
		var err error
		parsed, err = mcpFloatInt(typed, field)
		if err != nil {
			return 0, err
		}
	case json.Number:
		var err error
		parsed, err = typed.Int64()
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return fallback, nil
		}
		var err error
		parsed, err = strconv.ParseInt(text, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
	default:
		return 0, fmt.Errorf("%s must be an integer", field)
	}
	if parsed < int64(minimum) {
		return 0, fmt.Errorf("%s must be at least %d", field, minimum)
	}
	if parsed > maxMCPIntValue() {
		return 0, fmt.Errorf("%s is too large", field)
	}
	return int(parsed), nil
}

func mcpFloatInt(value float64, field string) (int64, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%s must be an integer", field)
	}
	if math.Trunc(value) != value {
		return 0, fmt.Errorf("%s must be an integer", field)
	}
	if value < float64(minMCPIntValue()) {
		return 0, fmt.Errorf("%s is too small", field)
	}
	if value >= float64(maxMCPIntValue()) {
		return 0, fmt.Errorf("%s is too large", field)
	}
	return int64(value), nil
}

func maxMCPIntValue() int64 {
	return int64(int(^uint(0) >> 1))
}

func minMCPIntValue() int64 {
	return -maxMCPIntValue() - 1
}

// RedactString redacts token-shaped values for compatibility facades.
func RedactString(value string) string { return redaction.String(value) }
