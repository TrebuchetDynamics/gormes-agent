package mcpstore

import (
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/paths"
	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
)

var serverNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type ErrorKind string

const (
	ErrorUnavailable   ErrorKind = "config_unavailable"
	ErrorRejected      ErrorKind = "config_rejected"
	ErrorAlreadyExists ErrorKind = "already_exists"
	ErrorInvalidInput  ErrorKind = "invalid_input"
	ErrorNotFound      ErrorKind = "not_found"
)

const (
	MaxToolSelections = 100
	MaxToolNameRunes  = 128
)

type ToolSelectionMode string

const (
	ToolSelectionInclude ToolSelectionMode = "include"
	ToolSelectionNone    ToolSelectionMode = "none"
	ToolSelectionAll     ToolSelectionMode = "all"
)

type ToolSelection struct {
	Mode    ToolSelectionMode
	Include []string
}

type Error struct {
	Kind ErrorKind
}

func (err *Error) Error() string {
	if err == nil || err.Kind == "" {
		return "mcp config unavailable"
	}
	return "mcp " + strings.ReplaceAll(string(err.Kind), "_", " ")
}

type HTTPRecord struct {
	URL          string
	OAuth        bool
	Enabled      bool
	ToolsInclude []string
}

type ServerRecord struct {
	Definition mcpconfig.MCPServerDefinition
	Status     mcpconfig.MCPServerStatus
	Auth       string
}

type ServerSummary struct {
	Name          string
	Transport     string
	Enabled       bool
	Status        string
	Auth          string
	Target        string
	ToolsSelected int
}

type Store struct {
	Path          string
	WriteDocument func(string, map[string]any) error
}

func (store Store) Load(opts mcpconfig.MCPConfigOptions) (mcpconfig.MCPConfigResolution, error) {
	doc, err := configwriter.ReadTOMLDoc(store.path())
	if err != nil {
		return mcpconfig.MCPConfigResolution{}, &Error{Kind: ErrorUnavailable}
	}
	resolved, err := mcpconfig.ResolveMCPConfig(doc, opts)
	if err != nil {
		return resolved, &Error{Kind: ErrorRejected}
	}
	return resolved, nil
}

func (store Store) Server(name string, opts mcpconfig.MCPConfigOptions) (ServerRecord, bool, error) {
	name = strings.TrimSpace(name)
	if !serverNamePattern.MatchString(name) {
		return ServerRecord{}, false, &Error{Kind: ErrorInvalidInput}
	}
	doc, err := configwriter.ReadTOMLDoc(store.path())
	if err != nil {
		return ServerRecord{}, false, &Error{Kind: ErrorUnavailable}
	}
	servers, err := serverMap(doc)
	if err != nil {
		return ServerRecord{}, false, err
	}
	rawValue, exists := servers[name]
	if !exists {
		return ServerRecord{}, false, nil
	}
	raw, ok := rawValue.(map[string]any)
	if !ok {
		return ServerRecord{}, false, &Error{Kind: ErrorRejected}
	}
	resolved, err := mcpconfig.ResolveMCPConfig(doc, opts)
	if err != nil {
		return ServerRecord{}, false, &Error{Kind: ErrorRejected}
	}
	status, ok := resolved.Status(name)
	if !ok {
		return ServerRecord{}, false, &Error{Kind: ErrorRejected}
	}
	record := ServerRecord{Status: status, Auth: authSummary(raw)}
	for _, definition := range resolved.Servers {
		if definition.Name == name {
			record.Definition = definition
			break
		}
	}
	return record, true, nil
}

func (store Store) UpsertHTTP(name string, record HTTPRecord) error {
	return store.writeHTTP(name, record, true)
}

func (store Store) AddHTTP(name string, record HTTPRecord, overwrite bool) error {
	return store.writeHTTP(name, record, overwrite)
}

func (store Store) writeHTTP(name string, record HTTPRecord, overwrite bool) error {
	name = strings.TrimSpace(name)
	if !serverNamePattern.MatchString(name) || !validHTTPURL(record.URL) {
		return &Error{Kind: ErrorInvalidInput}
	}
	doc, err := configwriter.ReadTOMLDoc(store.path())
	if err != nil {
		return &Error{Kind: ErrorUnavailable}
	}
	servers, err := serverMap(doc)
	if err != nil {
		return err
	}
	if _, exists := servers[name]; exists && !overwrite {
		return &Error{Kind: ErrorAlreadyExists}
	}

	entry := map[string]any{
		"url":     strings.TrimSpace(record.URL),
		"enabled": record.Enabled,
	}
	if record.OAuth {
		entry["auth"] = "oauth"
	}
	if existing, ok := servers[name].(map[string]any); ok {
		if tools, ok := existing["tools"].(map[string]any); ok {
			if include, exists := tools["include"]; exists {
				entry["tools"] = map[string]any{"include": cloneStringList(include)}
			}
		}
	} else if servers[name] != nil {
		return &Error{Kind: ErrorRejected}
	}
	if _, exists := entry["tools"]; !exists && record.ToolsInclude != nil {
		entry["tools"] = map[string]any{"include": append([]string(nil), record.ToolsInclude...)}
	}
	servers[name] = entry
	doc["mcp_servers"] = servers
	if _, err := mcpconfig.ResolveMCPConfig(doc, mcpconfig.MCPConfigOptions{LookupEnv: noEnvironment}); err != nil {
		return &Error{Kind: ErrorRejected}
	}
	write := store.WriteDocument
	if write == nil {
		write = configwriter.WriteTOMLAtomic
	}
	if err := write(store.path(), doc); err != nil {
		return &Error{Kind: ErrorUnavailable}
	}
	return nil
}

// ConfigureTools atomically updates only one existing server's tool-selection
// fields. It deliberately performs no server probe, environment lookup, or
// runtime mutation.
func (store Store) ConfigureTools(name string, selection ToolSelection) error {
	name = strings.TrimSpace(name)
	include, err := normalizeToolSelection(selection)
	if !validServerName(name) || err != nil {
		return &Error{Kind: ErrorInvalidInput}
	}
	doc, err := configwriter.ReadTOMLDoc(store.path())
	if err != nil {
		return &Error{Kind: ErrorUnavailable}
	}
	servers, err := serverMap(doc)
	if err != nil {
		return err
	}
	rawEntry, exists := servers[name]
	if !exists {
		return &Error{Kind: ErrorNotFound}
	}
	entry, ok := rawEntry.(map[string]any)
	if !ok {
		return &Error{Kind: ErrorRejected}
	}
	var tools map[string]any
	if rawTools, exists := entry["tools"]; exists && rawTools != nil {
		var ok bool
		tools, ok = rawTools.(map[string]any)
		if !ok {
			return &Error{Kind: ErrorRejected}
		}
	}
	if tools == nil {
		tools = map[string]any{}
	}
	switch selection.Mode {
	case ToolSelectionInclude, ToolSelectionNone:
		tools["include"] = include
		delete(tools, "exclude")
		entry["tools"] = tools
	case ToolSelectionAll:
		delete(tools, "include")
		delete(tools, "exclude")
		if len(tools) == 0 {
			delete(entry, "tools")
		} else {
			entry["tools"] = tools
		}
	default:
		return &Error{Kind: ErrorInvalidInput}
	}
	servers[name] = entry
	doc["mcp_servers"] = servers
	if _, err := mcpconfig.ResolveMCPConfig(doc, mcpconfig.MCPConfigOptions{LookupEnv: redactedEnvironment}); err != nil {
		return &Error{Kind: ErrorRejected}
	}
	write := store.WriteDocument
	if write == nil {
		write = configwriter.WriteTOMLAtomic
	}
	if err := write(store.path(), doc); err != nil {
		return &Error{Kind: ErrorUnavailable}
	}
	return nil
}

func normalizeToolSelection(selection ToolSelection) ([]string, error) {
	switch selection.Mode {
	case ToolSelectionInclude:
		if len(selection.Include) == 0 || len(selection.Include) > MaxToolSelections {
			return nil, &Error{Kind: ErrorInvalidInput}
		}
	case ToolSelectionNone:
		if len(selection.Include) != 0 {
			return nil, &Error{Kind: ErrorInvalidInput}
		}
		return []string{}, nil
	case ToolSelectionAll:
		if len(selection.Include) != 0 {
			return nil, &Error{Kind: ErrorInvalidInput}
		}
		return nil, nil
	default:
		return nil, &Error{Kind: ErrorInvalidInput}
	}
	out := append([]string(nil), selection.Include...)
	seen := make(map[string]struct{}, len(out))
	for _, name := range out {
		if name == "" || strings.TrimSpace(name) != name || !utf8.ValidString(name) || utf8.RuneCountInString(name) > MaxToolNameRunes {
			return nil, &Error{Kind: ErrorInvalidInput}
		}
		for _, r := range name {
			if unicode.IsControl(r) {
				return nil, &Error{Kind: ErrorInvalidInput}
			}
		}
		if _, exists := seen[name]; exists {
			return nil, &Error{Kind: ErrorInvalidInput}
		}
		seen[name] = struct{}{}
	}
	sort.Strings(out)
	return out, nil
}

func validServerName(name string) bool {
	return len(name) <= 64 && serverNamePattern.MatchString(name)
}

func (store Store) Remove(name string) (bool, error) {
	name = strings.TrimSpace(name)
	if !serverNamePattern.MatchString(name) {
		return false, &Error{Kind: ErrorInvalidInput}
	}
	doc, err := configwriter.ReadTOMLDoc(store.path())
	if err != nil {
		return false, &Error{Kind: ErrorUnavailable}
	}
	servers, err := serverMap(doc)
	if err != nil {
		return false, err
	}
	if _, exists := servers[name]; !exists {
		return false, nil
	}
	delete(servers, name)
	if len(servers) == 0 {
		delete(doc, "mcp_servers")
	} else {
		doc["mcp_servers"] = servers
	}
	if _, err := mcpconfig.ResolveMCPConfig(doc, mcpconfig.MCPConfigOptions{LookupEnv: noEnvironment}); err != nil {
		return false, &Error{Kind: ErrorRejected}
	}
	write := store.WriteDocument
	if write == nil {
		write = configwriter.WriteTOMLAtomic
	}
	if err := write(store.path(), doc); err != nil {
		return false, &Error{Kind: ErrorUnavailable}
	}
	return true, nil
}

func (store Store) List() ([]ServerSummary, error) {
	doc, err := configwriter.ReadTOMLDoc(store.path())
	if err != nil {
		return nil, &Error{Kind: ErrorUnavailable}
	}
	servers, err := serverMap(doc)
	if err != nil {
		return nil, err
	}
	resolved, err := mcpconfig.ResolveMCPConfig(doc, mcpconfig.MCPConfigOptions{LookupEnv: redactedEnvironment})
	if err != nil {
		return nil, &Error{Kind: ErrorRejected}
	}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ServerSummary, 0, len(names))
	for _, name := range names {
		raw, ok := servers[name].(map[string]any)
		if !ok {
			return nil, &Error{Kind: ErrorRejected}
		}
		status, ok := resolved.Status(name)
		if !ok {
			return nil, &Error{Kind: ErrorRejected}
		}
		transport := status.Transport
		if transport == "" {
			if _, ok := raw["url"].(string); ok {
				transport = mcpconfig.MCPTransportHTTP
			} else if _, ok := raw["command"].(string); ok {
				transport = mcpconfig.MCPTransportStdio
			}
		}
		summary := ServerSummary{
			Name:      name,
			Transport: string(transport),
			Enabled:   status.Enabled,
			Status:    string(status.Status),
			Auth:      authSummary(raw),
		}
		switch transport {
		case mcpconfig.MCPTransportHTTP:
			rawURL, _ := raw["url"].(string)
			summary.Target = httpOrigin(rawURL)
		case mcpconfig.MCPTransportStdio:
			rawCommand, _ := raw["command"].(string)
			summary.Target = commandBase(rawCommand)
		}
		if tools, ok := raw["tools"].(map[string]any); ok {
			summary.ToolsSelected = stringListLength(tools["include"])
		}
		out = append(out, summary)
	}
	return out, nil
}

func (store Store) path() string {
	if path := strings.TrimSpace(store.Path); path != "" {
		return path
	}
	return paths.ConfigPath()
}

func serverMap(doc map[string]any) (map[string]any, error) {
	raw, exists := doc["mcp_servers"]
	if !exists || raw == nil {
		return map[string]any{}, nil
	}
	servers, ok := raw.(map[string]any)
	if !ok {
		return nil, &Error{Kind: ErrorRejected}
	}
	return servers, nil
}

func validHTTPURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && parsed.User == nil
}

func RedactedHTTPTarget(raw string) string {
	return httpOrigin(raw)
}

func httpOrigin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return ""
	}
	return strings.ToLower(parsed.Scheme) + "://" + parsed.Host
}

func commandBase(command string) string {
	command = strings.ReplaceAll(strings.TrimSpace(command), `\`, "/")
	if command == "" {
		return ""
	}
	return filepath.Base(command)
}

func authSummary(raw map[string]any) string {
	if auth, _ := raw["auth"].(string); strings.EqualFold(strings.TrimSpace(auth), "oauth") {
		return "oauth"
	}
	if headers, ok := raw["headers"].(map[string]any); ok && len(headers) > 0 {
		return "header"
	}
	return "none"
}

func stringListLength(raw any) int {
	switch values := raw.(type) {
	case []string:
		return len(values)
	case []any:
		return len(values)
	default:
		return 0
	}
}

func cloneStringList(value any) any {
	switch values := value.(type) {
	case []string:
		return append([]string(nil), values...)
	case []any:
		out := make([]any, len(values))
		copy(out, values)
		return out
	default:
		return value
	}
}

func noEnvironment(string) (string, bool)       { return "", false }
func redactedEnvironment(string) (string, bool) { return "redacted", true }
