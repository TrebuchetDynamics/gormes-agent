package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	platformredaction "github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp"
	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
	mcpprobe "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/probe"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/remote"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

const (
	defaultDiscoveryTimeout       = 1500 * time.Millisecond
	defaultMaxServers             = 16
	defaultMaxToolsPerServer      = 100
	defaultMaxSchemaBytes         = 64 << 10
	defaultMaxAggregateSchemaByte = 512 << 10
	maxDescriptionRunes           = 1024
	maxSchemaDepth                = 32
	maxSchemaStringRunes          = 8192
	maxProviderNameRunes          = 64
	maxWireNameRunes              = 128
	maxInvocationTimeout          = 2 * time.Minute
)

var serverNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

type Options struct {
	ArtifactRoot            string
	DiscoveryTimeout        time.Duration
	MaxServers              int
	MaxToolsPerServer       int
	MaxSchemaBytes          int
	MaxAggregateSchemaBytes int
}

type Evidence string

const (
	EvidenceRegistered         Evidence = "registered"
	EvidenceConfigRejected     Evidence = "config_rejected"
	EvidenceServerLimit        Evidence = "server_limit"
	EvidenceUnsupported        Evidence = "unsupported_transport"
	EvidenceAuthRequired       Evidence = "auth_required"
	EvidenceDiscoveryFailed    Evidence = "discovery_failed"
	EvidenceFilterRejected     Evidence = "filter_rejected"
	EvidenceMetadataRejected   Evidence = "metadata_rejected"
	EvidenceRegistryCollision  Evidence = "registry_collision"
	EvidenceAggregateLimit     Evidence = "aggregate_schema_limit"
	EvidenceRuntimeUnavailable Evidence = "runtime_unavailable"
)

type Status struct {
	Server   string
	Evidence Evidence
	Count    int
}

type Report struct {
	Registered []string
	Statuses   []Status
}

// RegisterConfiguredHTTP resolves and discovers configured MCP servers, then
// installs bounded fresh-session tools into registry. Built-ins survive every
// failure and collision.
func RegisterConfiguredHTTP(ctx context.Context, registry *toolkit.Registry, rawServers map[string]any, connect remote.Connector, opts Options) Report {
	report := Report{Registered: []string{}, Statuses: []Status{}}
	if registry == nil || len(rawServers) == 0 {
		return report
	}
	if connect == nil {
		report.Statuses = append(report.Statuses, Status{Evidence: EvidenceRuntimeUnavailable})
		return report
	}
	opts = optionsWithDefaults(opts)
	resolution, err := mcpconfig.ResolveMCPConfig(map[string]any{"mcp_servers": rawServers}, mcpconfig.MCPConfigOptions{})
	if err != nil {
		report.Statuses = append(report.Statuses, Status{Evidence: EvidenceConfigRejected})
		return report
	}

	definitions := append([]mcpconfig.MCPServerDefinition(nil), resolution.Servers...)
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Name < definitions[j].Name })
	if len(definitions) > opts.MaxServers {
		for _, def := range definitions[opts.MaxServers:] {
			report.Statuses = append(report.Statuses, Status{Server: publicServerName(def.Name), Evidence: EvidenceServerLimit})
		}
		definitions = definitions[:opts.MaxServers]
	}

	type candidate struct {
		definition mcpconfig.MCPServerDefinition
		filter     toolFilter
		tools      []descriptor.RawTool
		evidence   Evidence
	}
	candidates := make([]candidate, 0, len(definitions))
	eligible := make([]candidate, 0, len(definitions))
	for _, def := range definitions {
		if publicServerName(def.Name) == "" {
			candidates = append(candidates, candidate{definition: def, evidence: EvidenceMetadataRejected})
			continue
		}
		raw, ok := rawServers[def.Name].(map[string]any)
		if !ok {
			candidates = append(candidates, candidate{definition: def, evidence: EvidenceConfigRejected})
			continue
		}
		if authKind(raw) == "oauth" {
			candidates = append(candidates, candidate{definition: def, evidence: EvidenceAuthRequired})
			continue
		}
		if def.Transport != mcpconfig.MCPTransportHTTP {
			candidates = append(candidates, candidate{definition: def, evidence: EvidenceUnsupported})
			continue
		}
		filter, err := parseToolFilter(raw["tools"])
		if err != nil {
			candidates = append(candidates, candidate{definition: def, evidence: EvidenceFilterRejected})
			continue
		}
		eligible = append(eligible, candidate{definition: def, filter: filter})
	}

	discoveryCtx, cancel := context.WithTimeout(ctx, opts.DiscoveryTimeout)
	defer cancel()
	results := make(chan candidate, len(eligible))
	var wg sync.WaitGroup
	for _, item := range eligible {
		item := item
		wg.Add(1)
		go func() {
			defer wg.Done()
			tools, err := mcpprobe.One(discoveryCtx, item.definition, remote.ProbeConnector(connect))
			if err != nil {
				item.evidence = EvidenceDiscoveryFailed
			} else {
				item.tools = tools
			}
			results <- item
		}()
	}
	wg.Wait()
	close(results)
	for result := range results {
		candidates = append(candidates, result)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].definition.Name < candidates[j].definition.Name })

	aggregateSchemaBytes := 0
	for _, item := range candidates {
		server := publicServerName(item.definition.Name)
		if item.evidence != "" {
			report.Statuses = append(report.Statuses, Status{Server: server, Evidence: item.evidence})
			continue
		}
		rawTools := filterAndSortTools(item.tools, item.filter)
		if len(rawTools) > opts.MaxToolsPerServer {
			rawTools = rawTools[:opts.MaxToolsPerServer]
		}
		normalized := descriptor.NormalizeTools(item.definition.Name, rawTools)
		registered := 0
		for _, normalizedTool := range normalized.Tools {
			providerName := "mcp__" + server + "__" + normalizedTool.Name
			if !safeWireName(normalizedTool.SourceRaw.Name) || len([]rune(providerName)) > maxProviderNameRunes {
				report.Statuses = append(report.Statuses, Status{Server: server, Evidence: EvidenceMetadataRejected})
				continue
			}
			schema, safe := safeSchema(normalizedTool.InputSchema, opts.MaxSchemaBytes)
			if !safe {
				report.Statuses = append(report.Statuses, Status{Server: server, Evidence: EvidenceMetadataRejected})
				continue
			}
			if aggregateSchemaBytes+len(schema) > opts.MaxAggregateSchemaBytes {
				report.Statuses = append(report.Statuses, Status{Server: server, Evidence: EvidenceAggregateLimit})
				break
			}
			tool := &remoteTool{
				name:         providerName,
				description:  safeDescription(server, normalizedTool.SourceRaw.Name, normalizedTool.Description),
				schema:       schema,
				server:       server,
				wireName:     normalizedTool.SourceRaw.Name,
				definition:   cloneDefinition(item.definition),
				connect:      connect,
				artifactRoot: opts.ArtifactRoot,
			}
			if err := registry.Register(tool); err != nil {
				report.Statuses = append(report.Statuses, Status{Server: server, Evidence: EvidenceRegistryCollision})
				continue
			}
			aggregateSchemaBytes += len(schema)
			registered++
			report.Registered = append(report.Registered, providerName)
		}
		if len(normalized.Rejected) > 0 {
			report.Statuses = append(report.Statuses, Status{Server: server, Evidence: EvidenceMetadataRejected, Count: len(normalized.Rejected)})
		}
		if registered > 0 {
			report.Statuses = append(report.Statuses, Status{Server: server, Evidence: EvidenceRegistered, Count: registered})
		}
	}
	sort.Strings(report.Registered)
	sort.SliceStable(report.Statuses, func(i, j int) bool {
		if report.Statuses[i].Server == report.Statuses[j].Server {
			return report.Statuses[i].Evidence < report.Statuses[j].Evidence
		}
		return report.Statuses[i].Server < report.Statuses[j].Server
	})
	return report
}

type remoteTool struct {
	name         string
	description  string
	schema       json.RawMessage
	server       string
	wireName     string
	definition   mcpconfig.MCPServerDefinition
	connect      remote.Connector
	artifactRoot string
}

func (tool *remoteTool) Name() string            { return tool.name }
func (tool *remoteTool) Description() string     { return tool.description }
func (tool *remoteTool) Schema() json.RawMessage { return append(json.RawMessage(nil), tool.schema...) }
func (tool *remoteTool) Timeout() time.Duration {
	timeout := tool.definition.Timeout
	if timeout <= 0 || timeout > maxInvocationTimeout {
		return maxInvocationTimeout
	}
	return timeout
}
func (tool *remoteTool) Spec() toolkit.OperationSpec {
	return toolkit.OperationSpec{
		ToolDescriptor: toolkit.ToolDescriptor{Name: tool.name, Description: tool.description, Schema: tool.Schema()},
		Mutating:       true,
		Idempotent:     false,
		PromptSafe:     true,
		TrustClass:     []string{"operator", "system"},
		AuditKind:      "mcp",
	}
}

func (tool *remoteTool) Execute(ctx context.Context, raw json.RawMessage) (out json.RawMessage, err error) {
	arguments, err := decodeObjectArguments(raw)
	if err != nil {
		return nil, errors.New("mcp tool invalid arguments")
	}
	session, err := tool.connect(ctx, cloneDefinition(tool.definition))
	if err != nil || session == nil {
		return nil, toolUnavailableError(ctx, tool.server, tool.name)
	}
	defer func() { _ = session.Close() }()
	result, err := session.CallTool(ctx, tool.wireName, arguments)
	if err != nil {
		return nil, toolUnavailableError(ctx, tool.server, tool.name)
	}
	artifactDir := ""
	if tool.artifactRoot != "" {
		artifactDir = filepath.Join(tool.artifactRoot, tool.server)
		_ = os.MkdirAll(artifactDir, 0o700)
	}
	rendered := mcp.RenderCallResultWithOptions(result, mcp.RenderOptions{ServerName: tool.server, ArtifactDir: artifactDir})
	sanitized := platformredaction.SanitizeUntrustedContent("mcp_output", rendered)
	return json.Marshal(sanitized.Text)
}

func toolUnavailableError(ctx context.Context, server, tool string) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("mcp_tool_timeout: server=%s tool=%s", server, tool)
	}
	return fmt.Errorf("mcp_tool_unavailable: server=%s tool=%s", server, tool)
}

func decodeObjectArguments(raw json.RawMessage) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil || value == nil {
		return nil, errors.New("invalid object")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errors.New("trailing content")
	}
	return value, nil
}

type toolFilter struct {
	include    map[string]struct{}
	includeSet bool
	exclude    map[string]struct{}
}

func parseToolFilter(raw any) (toolFilter, error) {
	filter := toolFilter{include: map[string]struct{}{}, exclude: map[string]struct{}{}}
	if raw == nil {
		return filter, nil
	}
	object, ok := raw.(map[string]any)
	if !ok {
		return toolFilter{}, errors.New("tools must be object")
	}
	var err error
	if rawInclude, exists := object["include"]; exists && rawInclude != nil {
		filter.includeSet = true
		filter.include, err = nameSet(rawInclude)
		if err != nil {
			return toolFilter{}, err
		}
	}
	filter.exclude, err = nameSet(object["exclude"])
	if err != nil {
		return toolFilter{}, err
	}
	return filter, nil
}

func nameSet(raw any) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if raw == nil {
		return out, nil
	}
	appendName := func(value string) error {
		if value == "" || len(value) > 256 {
			return errors.New("invalid tool name")
		}
		out[value] = struct{}{}
		return nil
	}
	switch values := raw.(type) {
	case string:
		if err := appendName(values); err != nil {
			return nil, err
		}
	case []string:
		for _, value := range values {
			if err := appendName(value); err != nil {
				return nil, err
			}
		}
	case []any:
		for _, rawValue := range values {
			value, ok := rawValue.(string)
			if !ok {
				return nil, errors.New("tool names must be strings")
			}
			if err := appendName(value); err != nil {
				return nil, err
			}
		}
	default:
		return nil, errors.New("tool filter must be string or list")
	}
	return out, nil
}

func filterAndSortTools(tools []descriptor.RawTool, filter toolFilter) []descriptor.RawTool {
	out := make([]descriptor.RawTool, 0, len(tools))
	for _, tool := range tools {
		if filter.includeSet {
			if _, ok := filter.include[tool.Name]; !ok {
				continue
			}
		} else if _, excluded := filter.exclude[tool.Name]; excluded {
			continue
		}
		copyTool := tool
		copyTool.InputSchema = append(json.RawMessage(nil), tool.InputSchema...)
		out = append(out, copyTool)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Description < out[j].Description
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func safeSchema(raw json.RawMessage, maxBytes int) (json.RawMessage, bool) {
	if len(raw) == 0 || len(raw) > maxBytes {
		return nil, false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}
	if _, ok := value.(map[string]any); !ok || !safeSchemaValue(value, 0) {
		return nil, false
	}
	return append(json.RawMessage(nil), raw...), true
}

func safeSchemaValue(value any, depth int) bool {
	if depth > maxSchemaDepth {
		return false
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if !safePromptString(key) || !safeSchemaValue(child, depth+1) {
				return false
			}
		}
	case []any:
		for _, child := range typed {
			if !safeSchemaValue(child, depth+1) {
				return false
			}
		}
	case string:
		return safePromptString(typed)
	}
	return true
}

func safePromptString(value string) bool {
	if len([]rune(value)) > maxSchemaStringRunes || platformredaction.ContainsSecret(value) || len(platformredaction.DetectPromptInjection(value)) > 0 {
		return false
	}
	if platformredaction.StripANSI(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func safeDescription(server, rawTool, description string) string {
	fallback := "MCP tool " + safeName(rawTool) + " from " + server
	if len(platformredaction.DetectPromptInjection(description)) > 0 {
		return fallback
	}
	value := platformredaction.RedactSecrets(platformredaction.StripANSI(description))
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return fallback
	}
	runes := []rune(value)
	if len(runes) > maxDescriptionRunes {
		value = string(runes[:maxDescriptionRunes-1]) + "…"
	}
	return value
}

func authKind(raw map[string]any) string {
	auth, _ := raw["auth"].(string)
	return strings.ToLower(strings.TrimSpace(auth))
}

func publicServerName(value string) string {
	if !serverNamePattern.MatchString(value) || platformredaction.ContainsSecret(value) || len(platformredaction.DetectPromptInjection(value)) > 0 {
		return ""
	}
	return descriptor.SanitizeNameComponent(value)
}

func safeWireName(value string) bool {
	if value == "" || len([]rune(value)) > maxWireNameRunes || platformredaction.ContainsSecret(value) || len(platformredaction.DetectPromptInjection(value)) > 0 || platformredaction.StripANSI(value) != value {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func safeName(value string) string {
	return descriptor.SanitizeNameComponent(value)
}

func cloneDefinition(in mcpconfig.MCPServerDefinition) mcpconfig.MCPServerDefinition {
	out := in
	out.Args = append([]string(nil), in.Args...)
	out.Env = cloneStringMap(in.Env)
	out.Headers = cloneStringMap(in.Headers)
	out.Sampling.AllowedModels = append([]string(nil), in.Sampling.AllowedModels...)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func optionsWithDefaults(opts Options) Options {
	if opts.DiscoveryTimeout <= 0 {
		opts.DiscoveryTimeout = defaultDiscoveryTimeout
	}
	if opts.MaxServers <= 0 {
		opts.MaxServers = defaultMaxServers
	}
	if opts.MaxToolsPerServer <= 0 {
		opts.MaxToolsPerServer = defaultMaxToolsPerServer
	}
	if opts.MaxSchemaBytes <= 0 {
		opts.MaxSchemaBytes = defaultMaxSchemaBytes
	}
	if opts.MaxAggregateSchemaBytes <= 0 {
		opts.MaxAggregateSchemaBytes = defaultMaxAggregateSchemaByte
	}
	return opts
}
