package tools

import (
	"strings"
	"testing"
)

func TestHomeAssistantToolRegistrationRequiresToken(t *testing.T) {
	t.Setenv("HASS_TOKEN", "")
	if got := NewHomeAssistantTools(HomeAssistantConfig{}); len(got) != 0 {
		t.Fatalf("tools without token = %d, want 0", len(got))
	}

	t.Setenv("HASS_TOKEN", "test-token")
	reg := NewRegistry()
	RegisterHomeAssistantTools(reg, HomeAssistantConfig{})
	names := map[string]bool{}
	for _, desc := range reg.Descriptors() {
		names[desc.Name] = true
		if contains(string(desc.Schema), "test-token") {
			t.Fatalf("%s schema leaked token: %s", desc.Name, desc.Schema)
		}
	}
	for _, name := range []string{"ha_list_entities", "ha_get_state", "ha_list_services", "ha_call_service"} {
		if !names[name] {
			t.Fatalf("registry missing %s in descriptors %#v", name, names)
		}
	}
}

func contains(haystack string, needle string) bool { return strings.Contains(haystack, needle) }
