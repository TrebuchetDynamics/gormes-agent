package homeassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	HomeAssistantEvidenceOK                = "homeassistant_tool_ok"
	HomeAssistantEvidenceUnavailable       = "homeassistant_tool_unavailable"
	HomeAssistantEvidenceValidationFailed  = "homeassistant_validation_failed"
	defaultHomeAssistantURL                = "http://homeassistant.local:8123"
	defaultHomeAssistantTimeout            = 15 * time.Second
	homeAssistantListEntitiesToolName      = "ha_list_entities"
	homeAssistantGetStateToolName          = "ha_get_state"
	homeAssistantListServicesToolName      = "ha_list_services"
	homeAssistantCallServiceToolName       = "ha_call_service"
	homeAssistantBlockedDomainShellCommand = "shell_command"
	homeAssistantBlockedDomainCommandLine  = "command_line"
	homeAssistantBlockedDomainPythonScript = "python_script"
	homeAssistantBlockedDomainPyscript     = "pyscript"
	homeAssistantBlockedDomainHassio       = "hassio"
	homeAssistantBlockedDomainAddon        = "addon"
	homeAssistantBlockedDomainRestCommand  = "rest_command"
)

var (
	homeAssistantEntityIDPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*\.[a-z0-9_]+$`)
	homeAssistantServicePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	homeAssistantBlockedDomains  = map[string]struct{}{
		homeAssistantBlockedDomainShellCommand: {},
		homeAssistantBlockedDomainCommandLine:  {},
		homeAssistantBlockedDomainPythonScript: {},
		homeAssistantBlockedDomainPyscript:     {},
		homeAssistantBlockedDomainHassio:       {},
		homeAssistantBlockedDomainAddon:        {},
		homeAssistantBlockedDomainRestCommand:  {},
	}
)

type HomeAssistantConfig struct {
	BaseURL string
	Token   string
	Client  HomeAssistantClient
	Timeout time.Duration
}

type HomeAssistantClient interface {
	ListStates(context.Context) ([]HomeAssistantState, error)
	GetState(context.Context, string) (HomeAssistantState, error)
	ListServices(context.Context) ([]HomeAssistantServiceDomain, error)
	CallService(context.Context, string, string, map[string]any) (any, error)
}

type HomeAssistantState struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes,omitempty"`
	LastChanged string         `json:"last_changed,omitempty"`
	LastUpdated string         `json:"last_updated,omitempty"`
}

type HomeAssistantServiceDomain struct {
	Domain   string                          `json:"domain"`
	Services map[string]HomeAssistantService `json:"services"`
}

type HomeAssistantService struct {
	Description string                               `json:"description,omitempty"`
	Fields      map[string]HomeAssistantServiceField `json:"fields,omitempty"`
}

type HomeAssistantServiceField struct {
	Description string `json:"description,omitempty"`
}

type homeAssistantTool struct {
	name        string
	description string
	schema      json.RawMessage
	cfg         HomeAssistantConfig
}

type homeAssistantEnvelope struct {
	Result   any    `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
	Evidence string `json:"evidence,omitempty"`
}

type homeAssistantEntitySummary struct {
	EntityID     string `json:"entity_id"`
	State        string `json:"state"`
	FriendlyName string `json:"friendly_name"`
}

type homeAssistantEntitiesResult struct {
	Count    int                          `json:"count"`
	Entities []homeAssistantEntitySummary `json:"entities"`
}

type homeAssistantStateResult struct {
	EntityID    string         `json:"entity_id"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
	LastChanged string         `json:"last_changed,omitempty"`
	LastUpdated string         `json:"last_updated,omitempty"`
}

type homeAssistantServicesResult struct {
	Count   int                                 `json:"count"`
	Domains []homeAssistantServiceDomainSummary `json:"domains"`
}

type homeAssistantServiceDomainSummary struct {
	Domain   string                                 `json:"domain"`
	Services map[string]homeAssistantServiceSummary `json:"services"`
}

type homeAssistantServiceSummary struct {
	Description string            `json:"description"`
	Fields      map[string]string `json:"fields,omitempty"`
}

type homeAssistantCallServiceResult struct {
	Success          bool                         `json:"success"`
	Service          string                       `json:"service"`
	AffectedEntities []homeAssistantAffectedState `json:"affected_entities"`
}

type homeAssistantAffectedState struct {
	EntityID string `json:"entity_id"`
	State    string `json:"state"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func NewHomeAssistantTools(cfg HomeAssistantConfig) []toolkit.Tool {
	cfg = cfg.withDefaults()
	if strings.TrimSpace(cfg.Token) == "" {
		return nil
	}
	return []toolkit.Tool{
		&homeAssistantTool{name: homeAssistantListEntitiesToolName, description: "List Home Assistant entities. Optionally filter by domain (light, switch, climate, sensor, binary_sensor, cover, fan, etc.) or by area name (living room, kitchen, bedroom, etc.).", schema: homeAssistantListEntitiesSchema, cfg: cfg},
		&homeAssistantTool{name: homeAssistantGetStateToolName, description: "Get the detailed state of a single Home Assistant entity, including all attributes (brightness, color, temperature setpoint, sensor readings, etc.).", schema: homeAssistantGetStateSchema, cfg: cfg},
		&homeAssistantTool{name: homeAssistantListServicesToolName, description: "List available Home Assistant services (actions) for device control. Shows what actions can be performed on each device type and what parameters they accept. Use this to discover how to control devices found via ha_list_entities.", schema: homeAssistantListServicesSchema, cfg: cfg},
		&homeAssistantTool{name: homeAssistantCallServiceToolName, description: "Call a Home Assistant service to control a device. Use ha_list_services to discover available services and their parameters for each domain.", schema: homeAssistantCallServiceSchema, cfg: cfg},
	}
}

func (c HomeAssistantConfig) withDefaults() HomeAssistantConfig {
	c.Token = strings.TrimSpace(firstNonEmpty(c.Token, os.Getenv("HASS_TOKEN")))
	c.BaseURL = strings.TrimRight(strings.TrimSpace(firstNonEmpty(c.BaseURL, os.Getenv("HASS_URL"), defaultHomeAssistantURL)), "/")
	if c.Timeout <= 0 {
		c.Timeout = defaultHomeAssistantTimeout
	}
	if c.Client == nil && c.Token != "" {
		c.Client = &homeAssistantHTTPClient{baseURL: c.BaseURL, token: c.Token, timeout: c.Timeout, client: http.DefaultClient}
	}
	return c
}

func (t *homeAssistantTool) Name() string { return t.name }

func (t *homeAssistantTool) Description() string { return t.description }

func (t *homeAssistantTool) Schema() json.RawMessage {
	return append(json.RawMessage(nil), t.schema...)
}

func (*homeAssistantTool) Timeout() time.Duration { return defaultHomeAssistantTimeout }

func (t *homeAssistantTool) Spec() toolkit.OperationSpec {
	return toolkit.OperationSpec{
		ToolDescriptor: toolkit.ToolDescriptor{Name: t.Name(), Description: t.Description(), Schema: t.Schema()},
		Mutating:       t.name == homeAssistantCallServiceToolName,
		Idempotent:     t.name != homeAssistantCallServiceToolName,
		PromptSafe:     true,
		TrustClass:     []string{"operator", "gateway", "child-agent", "system"},
		AuditKind:      "homeassistant",
	}
}

func (t *homeAssistantTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t == nil || t.cfg.Client == nil || strings.TrimSpace(t.cfg.Token) == "" {
		return marshalHomeAssistantError(HomeAssistantEvidenceUnavailable, "Home Assistant token or client is unavailable")
	}
	if len(strings.TrimSpace(string(args))) == 0 {
		args = json.RawMessage(`{}`)
	}
	switch t.name {
	case homeAssistantListEntitiesToolName:
		return t.executeListEntities(ctx, args)
	case homeAssistantGetStateToolName:
		return t.executeGetState(ctx, args)
	case homeAssistantListServicesToolName:
		return t.executeListServices(ctx, args)
	case homeAssistantCallServiceToolName:
		return t.executeCallService(ctx, args)
	default:
		return marshalHomeAssistantError(HomeAssistantEvidenceUnavailable, "unknown Home Assistant tool")
	}
}

func (t *homeAssistantTool) executeListEntities(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Domain string `json:"domain"`
		Area   string `json:"area"`
		Search string `json:"search"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, "invalid list entities arguments")
	}
	states, err := t.cfg.Client.ListStates(ctx)
	if err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceUnavailable, t.redact(err.Error()))
	}
	return marshalHomeAssistantResult(filterHomeAssistantStates(states, in.Domain, in.Area, in.Search))
}

func (t *homeAssistantTool) executeGetState(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		EntityID string `json:"entity_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, "invalid get state arguments")
	}
	entityID := strings.TrimSpace(in.EntityID)
	if entityID == "" {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, "Missing required parameter: entity_id")
	}
	if !homeAssistantEntityIDPattern.MatchString(entityID) {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, "Invalid entity_id format: "+entityID)
	}
	state, err := t.cfg.Client.GetState(ctx, entityID)
	if err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceUnavailable, t.redact(err.Error()))
	}
	return marshalHomeAssistantResult(homeAssistantStateResult{
		EntityID:    state.EntityID,
		State:       state.State,
		Attributes:  cloneHomeAssistantAttributes(state.Attributes),
		LastChanged: state.LastChanged,
		LastUpdated: state.LastUpdated,
	})
}

func (t *homeAssistantTool) executeListServices(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, "invalid list services arguments")
	}
	services, err := t.cfg.Client.ListServices(ctx)
	if err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceUnavailable, t.redact(err.Error()))
	}
	return marshalHomeAssistantResult(summarizeHomeAssistantServices(services, in.Domain))
}

func (t *homeAssistantTool) executeCallService(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Domain   string          `json:"domain"`
		Service  string          `json:"service"`
		EntityID string          `json:"entity_id"`
		Data     json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, "invalid call service arguments")
	}
	domain := strings.TrimSpace(in.Domain)
	service := strings.TrimSpace(in.Service)
	if domain == "" || service == "" {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, "Missing required parameters: domain and service")
	}
	if !homeAssistantServicePattern.MatchString(domain) {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, fmt.Sprintf("Invalid domain format: %q", domain))
	}
	if !homeAssistantServicePattern.MatchString(service) {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, fmt.Sprintf("Invalid service format: %q", service))
	}
	if _, blocked := homeAssistantBlockedDomains[domain]; blocked {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, fmt.Sprintf("Service domain %q is blocked for security. Blocked domains: %s", domain, strings.Join(sortedHomeAssistantBlockedDomains(), ", ")))
	}
	entityID := strings.TrimSpace(in.EntityID)
	if entityID != "" && !homeAssistantEntityIDPattern.MatchString(entityID) {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, "Invalid entity_id format: "+entityID)
	}
	payload, err := parseHomeAssistantServiceData(in.Data)
	if err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceValidationFailed, err.Error())
	}
	if entityID != "" {
		payload["entity_id"] = entityID
	}
	result, err := t.cfg.Client.CallService(ctx, domain, service, payload)
	if err != nil {
		return marshalHomeAssistantError(HomeAssistantEvidenceUnavailable, t.redact(err.Error()))
	}
	return marshalHomeAssistantResult(parseHomeAssistantServiceResponse(domain, service, result))
}

func filterHomeAssistantStates(states []HomeAssistantState, domain string, area string, search string) homeAssistantEntitiesResult {
	domain = strings.TrimSpace(domain)
	area = strings.ToLower(strings.TrimSpace(area))
	search = strings.ToLower(strings.TrimSpace(search))
	entities := make([]homeAssistantEntitySummary, 0, len(states))
	for _, state := range states {
		if domain != "" && !strings.HasPrefix(state.EntityID, domain+".") {
			continue
		}
		friendly := strings.ToLower(homeAssistantStringAttribute(state.Attributes, "friendly_name"))
		if area != "" {
			stateArea := strings.ToLower(homeAssistantStringAttribute(state.Attributes, "area"))
			if !strings.Contains(friendly, area) && !strings.Contains(stateArea, area) {
				continue
			}
		}
		if search != "" && !strings.Contains(strings.ToLower(state.EntityID), search) && !strings.Contains(friendly, search) {
			continue
		}
		entities = append(entities, homeAssistantEntitySummary{
			EntityID:     state.EntityID,
			State:        state.State,
			FriendlyName: homeAssistantStringAttribute(state.Attributes, "friendly_name"),
		})
	}
	return homeAssistantEntitiesResult{Count: len(entities), Entities: entities}
}

func summarizeHomeAssistantServices(services []HomeAssistantServiceDomain, domain string) homeAssistantServicesResult {
	domain = strings.TrimSpace(domain)
	domains := make([]homeAssistantServiceDomainSummary, 0, len(services))
	for _, serviceDomain := range services {
		if domain != "" && serviceDomain.Domain != domain {
			continue
		}
		summary := homeAssistantServiceDomainSummary{
			Domain:   serviceDomain.Domain,
			Services: make(map[string]homeAssistantServiceSummary, len(serviceDomain.Services)),
		}
		for name, service := range serviceDomain.Services {
			fields := make(map[string]string, len(service.Fields))
			for fieldName, field := range service.Fields {
				fields[fieldName] = field.Description
			}
			entry := homeAssistantServiceSummary{Description: service.Description}
			if len(fields) > 0 {
				entry.Fields = fields
			}
			summary.Services[name] = entry
		}
		domains = append(domains, summary)
	}
	return homeAssistantServicesResult{Count: len(domains), Domains: domains}
}

func parseHomeAssistantServiceResponse(domain string, service string, result any) homeAssistantCallServiceResult {
	affected := []homeAssistantAffectedState{}
	if states, ok := result.([]HomeAssistantState); ok {
		for _, state := range states {
			affected = append(affected, homeAssistantAffectedState{EntityID: state.EntityID, State: state.State})
		}
	}
	return homeAssistantCallServiceResult{
		Success:          true,
		Service:          domain + "." + service,
		AffectedEntities: affected,
	}
}

func parseHomeAssistantServiceData(raw json.RawMessage) (map[string]any, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return map[string]any{}, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if strings.TrimSpace(text) == "" {
			return map[string]any{}, nil
		}
		var out map[string]any
		if err := json.Unmarshal([]byte(text), &out); err != nil {
			return nil, fmt.Errorf("invalid JSON string in 'data' parameter: %v", err)
		}
		return out, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid service data parameter: %v", err)
	}
	return out, nil
}

func homeAssistantStringAttribute(attributes map[string]any, key string) string {
	if attributes == nil {
		return ""
	}
	value, ok := attributes[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func cloneHomeAssistantAttributes(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func sortedHomeAssistantBlockedDomains() []string {
	out := make([]string, 0, len(homeAssistantBlockedDomains))
	for domain := range homeAssistantBlockedDomains {
		out = append(out, domain)
	}
	sort.Strings(out)
	return out
}

func (t *homeAssistantTool) redact(message string) string {
	token := strings.TrimSpace(t.cfg.Token)
	if token == "" {
		return message
	}
	return strings.ReplaceAll(message, token, "[redacted]")
}

func marshalHomeAssistantResult(result any) (json.RawMessage, error) {
	raw, err := json.Marshal(homeAssistantEnvelope{Result: result, Evidence: HomeAssistantEvidenceOK})
	return raw, err
}

func marshalHomeAssistantError(evidence string, message string) (json.RawMessage, error) {
	raw, err := json.Marshal(homeAssistantEnvelope{Error: message, Evidence: evidence})
	return raw, err
}

type homeAssistantHTTPClient struct {
	baseURL string
	token   string
	timeout time.Duration
	client  *http.Client
}

func (c *homeAssistantHTTPClient) ListStates(ctx context.Context) ([]HomeAssistantState, error) {
	var states []HomeAssistantState
	if err := c.getJSON(ctx, "/api/states", &states); err != nil {
		return nil, err
	}
	return states, nil
}

func (c *homeAssistantHTTPClient) GetState(ctx context.Context, entityID string) (HomeAssistantState, error) {
	var state HomeAssistantState
	if err := c.getJSON(ctx, "/api/states/"+entityID, &state); err != nil {
		return HomeAssistantState{}, err
	}
	return state, nil
}

func (c *homeAssistantHTTPClient) ListServices(ctx context.Context) ([]HomeAssistantServiceDomain, error) {
	var services []HomeAssistantServiceDomain
	if err := c.getJSON(ctx, "/api/services", &services); err != nil {
		return nil, err
	}
	return services, nil
}

func (c *homeAssistantHTTPClient) CallService(ctx context.Context, domain string, service string, payload map[string]any) (any, error) {
	var result []HomeAssistantState
	if err := c.postJSON(ctx, "/api/services/"+domain+"/"+service, payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *homeAssistantHTTPClient) getJSON(ctx context.Context, apiPath string, out any) error {
	req, err := c.request(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *homeAssistantHTTPClient) postJSON(ctx context.Context, apiPath string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := c.request(ctx, http.MethodPost, apiPath, bytes.NewReader(body))
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *homeAssistantHTTPClient) request(ctx context.Context, method string, apiPath string, body io.Reader) (*http.Request, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+apiPath, body)
	if err != nil {
		cancel()
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), homeAssistantCancelKey{}, cancel))
	return req, nil
}

func (c *homeAssistantHTTPClient) do(req *http.Request, out any) error {
	cancel, _ := req.Context().Value(homeAssistantCancelKey{}).(context.CancelFunc)
	if cancel != nil {
		defer cancel()
	}
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("home assistant returned HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

type homeAssistantCancelKey struct{}

var homeAssistantListEntitiesSchema = json.RawMessage(`{"type":"object","properties":{"domain":{"type":"string","description":"Entity domain to filter by (e.g. 'light', 'switch', 'climate', 'sensor', 'binary_sensor', 'cover', 'fan', 'media_player'). Omit to list all entities."},"area":{"type":"string","description":"Area/room name to filter by (e.g. 'living room', 'kitchen'). Matches against entity friendly names. Omit to list all."},"search":{"type":"string","description":"Case-insensitive text to match against entity IDs and friendly names. Omit to skip text filtering."}},"required":[]}`)

var homeAssistantGetStateSchema = json.RawMessage(`{"type":"object","properties":{"entity_id":{"type":"string","description":"The entity ID to query (e.g. 'light.living_room', 'climate.thermostat', 'sensor.temperature')."}},"required":["entity_id"]}`)

var homeAssistantListServicesSchema = json.RawMessage(`{"type":"object","properties":{"domain":{"type":"string","description":"Filter by domain (e.g. 'light', 'climate', 'switch'). Omit to list services for all domains."}},"required":[]}`)

var homeAssistantCallServiceSchema = json.RawMessage(`{"type":"object","properties":{"domain":{"type":"string","description":"Service domain (e.g. 'light', 'switch', 'climate', 'cover', 'media_player', 'fan', 'scene', 'script')."},"service":{"type":"string","description":"Service name (e.g. 'turn_on', 'turn_off', 'toggle', 'set_temperature', 'set_hvac_mode', 'open_cover', 'close_cover', 'set_volume_level')."},"entity_id":{"type":"string","description":"Target entity ID (e.g. 'light.living_room'). Some services (like scene.turn_on) may not need this."},"data":{"type":"string","description":"Additional service data as a JSON string. Examples: {\"brightness\": 255, \"color_name\": \"blue\"} for lights, {\"temperature\": 22, \"hvac_mode\": \"heat\"} for climate, {\"volume_level\": 0.5} for media players."}},"required":["domain","service"]}`)
