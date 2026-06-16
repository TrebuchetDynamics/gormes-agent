package homeassistant

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

func TestHomeAssistantListEntitiesFiltersAndSummarizes(t *testing.T) {
	client := &fakeHomeAssistantClient{
		states: []HomeAssistantState{
			{EntityID: "light.bedroom", State: "on", Attributes: map[string]any{"friendly_name": "Bedroom Light"}},
			{EntityID: "light.kitchen", State: "off", Attributes: map[string]any{"friendly_name": "Kitchen Light"}},
			{EntityID: "sensor.temperature", State: "22.5", Attributes: map[string]any{"friendly_name": "Kitchen Temperature", "unit_of_measurement": "C"}},
			{EntityID: "sensor.humidity", State: "55", Attributes: map[string]any{"friendly_name": "Bedroom Humidity", "area": "bedroom"}},
		},
	}
	tool := mustHomeAssistantTool(t, "ha_list_entities", HomeAssistantConfig{
		Token:  "test-token",
		Client: client,
	})

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"domain":"sensor","area":"kitchen","search":"temp"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got struct {
		Result struct {
			Count    int `json:"count"`
			Entities []struct {
				EntityID     string `json:"entity_id"`
				State        string `json:"state"`
				FriendlyName string `json:"friendly_name"`
			} `json:"entities"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("result JSON: %v\n%s", err, raw)
	}
	if got.Result.Count != 1 || len(got.Result.Entities) != 1 {
		t.Fatalf("result = %+v, want one filtered entity", got.Result)
	}
	entity := got.Result.Entities[0]
	if entity.EntityID != "sensor.temperature" || entity.State != "22.5" || entity.FriendlyName != "Kitchen Temperature" {
		t.Fatalf("entity = %+v, want compact Hermes entity summary", entity)
	}
	if client.listStatesCalls != 1 {
		t.Fatalf("ListStates calls = %d, want 1", client.listStatesCalls)
	}
}

func TestHomeAssistantGetStateAndListServices(t *testing.T) {
	client := &fakeHomeAssistantClient{
		state: HomeAssistantState{
			EntityID:    "light.kitchen",
			State:       "on",
			Attributes:  map[string]any{"friendly_name": "Kitchen Light", "brightness": float64(180)},
			LastChanged: "2026-05-06T02:00:00Z",
			LastUpdated: "2026-05-06T02:01:00Z",
		},
		services: []HomeAssistantServiceDomain{
			{
				Domain: "light",
				Services: map[string]HomeAssistantService{
					"turn_on": {
						Description: "Turn on a light",
						Fields: map[string]HomeAssistantServiceField{
							"brightness": {Description: "Brightness from 0 to 255"},
						},
					},
				},
			},
			{Domain: "climate", Services: map[string]HomeAssistantService{"set_temperature": {Description: "Set target temperature"}}},
		},
	}

	getState := mustHomeAssistantTool(t, "ha_get_state", HomeAssistantConfig{Token: "test-token", Client: client})
	raw, err := getState.Execute(context.Background(), json.RawMessage(`{"entity_id":"light.kitchen"}`))
	if err != nil {
		t.Fatalf("get state Execute: %v", err)
	}
	var stateResult struct {
		Result struct {
			EntityID    string         `json:"entity_id"`
			State       string         `json:"state"`
			Attributes  map[string]any `json:"attributes"`
			LastChanged string         `json:"last_changed"`
			LastUpdated string         `json:"last_updated"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &stateResult); err != nil {
		t.Fatalf("state result JSON: %v\n%s", err, raw)
	}
	if stateResult.Result.EntityID != "light.kitchen" || stateResult.Result.Attributes["friendly_name"] != "Kitchen Light" {
		t.Fatalf("state result = %+v, want detailed state", stateResult.Result)
	}

	listServices := mustHomeAssistantTool(t, "ha_list_services", HomeAssistantConfig{Token: "test-token", Client: client})
	raw, err = listServices.Execute(context.Background(), json.RawMessage(`{"domain":"light"}`))
	if err != nil {
		t.Fatalf("list services Execute: %v", err)
	}
	var servicesResult struct {
		Result struct {
			Count   int `json:"count"`
			Domains []struct {
				Domain   string `json:"domain"`
				Services map[string]struct {
					Description string            `json:"description"`
					Fields      map[string]string `json:"fields"`
				} `json:"services"`
			} `json:"domains"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &servicesResult); err != nil {
		t.Fatalf("services result JSON: %v\n%s", err, raw)
	}
	if servicesResult.Result.Count != 1 || servicesResult.Result.Domains[0].Domain != "light" {
		t.Fatalf("services result = %+v, want one light domain", servicesResult.Result)
	}
	if got := servicesResult.Result.Domains[0].Services["turn_on"].Fields["brightness"]; got != "Brightness from 0 to 255" {
		t.Fatalf("brightness field = %q, want compact field description", got)
	}
	if client.getStateEntityID != "light.kitchen" || client.listServicesCalls != 1 {
		t.Fatalf("client calls = get %q services %d", client.getStateEntityID, client.listServicesCalls)
	}
}

func TestHomeAssistantCallServiceBuildsPayload(t *testing.T) {
	client := &fakeHomeAssistantClient{
		callResult: []HomeAssistantState{
			{EntityID: "light.bedroom", State: "on"},
			{EntityID: "light.kitchen", State: "on"},
		},
	}
	tool := mustHomeAssistantTool(t, "ha_call_service", HomeAssistantConfig{Token: "test-token", Client: client})

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"domain":"light","service":"turn_on","entity_id":"light.bedroom","data":"{\"entity_id\":\"light.other\",\"brightness\":200}"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got struct {
		Result struct {
			Success          bool   `json:"success"`
			Service          string `json:"service"`
			AffectedEntities []struct {
				EntityID string `json:"entity_id"`
				State    string `json:"state"`
			} `json:"affected_entities"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("result JSON: %v\n%s", err, raw)
	}
	if !got.Result.Success || got.Result.Service != "light.turn_on" || len(got.Result.AffectedEntities) != 2 {
		t.Fatalf("call result = %+v, want Hermes success envelope", got.Result)
	}
	if client.callDomain != "light" || client.callService != "turn_on" {
		t.Fatalf("called %s.%s, want light.turn_on", client.callDomain, client.callService)
	}
	if client.callPayload["entity_id"] != "light.bedroom" {
		t.Fatalf("entity_id payload = %#v, want explicit arg precedence", client.callPayload["entity_id"])
	}
	if client.callPayload["brightness"] != float64(200) {
		t.Fatalf("brightness payload = %#v, want parsed JSON string data", client.callPayload["brightness"])
	}
}

func TestHomeAssistantSafetyRejectsTraversalAndBlockedDomains(t *testing.T) {
	tool := mustHomeAssistantTool(t, "ha_call_service", HomeAssistantConfig{Token: "test-token", Client: &fakeHomeAssistantClient{}})
	for _, tc := range []struct {
		name string
		args string
		want string
	}{
		{name: "traversal domain", args: `{"domain":"../../api/config","service":"turn_on"}`, want: "Invalid domain"},
		{name: "traversal service", args: `{"domain":"light","service":"../../api/config"}`, want: "Invalid service"},
		{name: "blocked shell command", args: `{"domain":"shell_command","service":"turn_on"}`, want: "blocked for security"},
		{name: "blocked addon", args: `{"domain":"addon","service":"restart"}`, want: "blocked for security"},
		{name: "invalid entity", args: `{"domain":"light","service":"turn_on","entity_id":"../../../etc/passwd"}`, want: "Invalid entity_id"},
		{name: "invalid data", args: `{"domain":"light","service":"turn_on","entity_id":"light.bedroom","data":"{not valid json}"}`, want: "invalid JSON"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := tool.Execute(context.Background(), json.RawMessage(tc.args))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			var got struct {
				Error    string `json:"error"`
				Evidence string `json:"evidence"`
			}
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("error result JSON: %v\n%s", err, raw)
			}
			if got.Evidence != HomeAssistantEvidenceValidationFailed || got.Error == "" || !contains(got.Error, tc.want) {
				t.Fatalf("error = %+v, want validation evidence containing %q", got, tc.want)
			}
			if contains(got.Error, "test-token") {
				t.Fatalf("error leaked token: %+v", got)
			}
		})
	}
}

func mustHomeAssistantTool(t *testing.T, name string, cfg HomeAssistantConfig) toolkit.Tool {
	t.Helper()
	for _, tool := range NewHomeAssistantTools(cfg) {
		if tool.Name() == name {
			return tool
		}
	}
	t.Fatalf("NewHomeAssistantTools missing %s", name)
	return nil
}

type fakeHomeAssistantClient struct {
	states            []HomeAssistantState
	state             HomeAssistantState
	services          []HomeAssistantServiceDomain
	callResult        any
	listStatesCalls   int
	getStateEntityID  string
	listServicesCalls int
	callDomain        string
	callService       string
	callPayload       map[string]any
}

func (f *fakeHomeAssistantClient) ListStates(context.Context) ([]HomeAssistantState, error) {
	f.listStatesCalls++
	return append([]HomeAssistantState(nil), f.states...), nil
}

func (f *fakeHomeAssistantClient) GetState(_ context.Context, entityID string) (HomeAssistantState, error) {
	f.getStateEntityID = entityID
	return f.state, nil
}

func (f *fakeHomeAssistantClient) ListServices(context.Context) ([]HomeAssistantServiceDomain, error) {
	f.listServicesCalls++
	return append([]HomeAssistantServiceDomain(nil), f.services...), nil
}

func (f *fakeHomeAssistantClient) CallService(_ context.Context, domain string, service string, payload map[string]any) (any, error) {
	f.callDomain = domain
	f.callService = service
	f.callPayload = payload
	return f.callResult, nil
}

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
