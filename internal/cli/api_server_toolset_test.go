package cli

import (
	"reflect"
	"sort"
	"testing"
)

func TestAPIServerToolsetDefaultIncludesHermesAPIServerTools(t *testing.T) {
	cfg, _ := parseToolsetConfigYAML(t, `platform_toolsets: {}`)

	status, err := cfg.PlatformStatus("api_server")
	if err != nil {
		t.Fatalf("PlatformStatus(api_server): %v", err)
	}

	for _, tool := range []string{
		"web_search", "web_extract",
		"browser_navigate", "browser_snapshot", "browser_click",
		"ha_list_entities", "ha_get_state", "ha_list_services", "ha_call_service",
		"read_file", "write_file", "patch", "search_files",
		"execute_code", "delegate_task", "todo", "memory", "session_search", "cronjob",
	} {
		assertContains(t, status.RuntimeToolsets, tool)
	}
	for _, forbidden := range []string{"clarify", "send_message", "text_to_speech"} {
		assertNotContains(t, status.RuntimeToolsets, forbidden)
	}
	if got := append([]string(nil), status.RuntimeToolsets...); !sort.StringsAreSorted(got) {
		t.Fatalf("api_server default tools are not sorted: %v", got)
	}
}

func TestAPIServerToolsetOverride(t *testing.T) {
	cfg, _ := parseToolsetConfigYAML(t, `
platform_toolsets:
  api_server:
    - web
    - terminal
`)

	status, err := cfg.PlatformStatus("api_server")
	if err != nil {
		t.Fatalf("PlatformStatus(api_server): %v", err)
	}
	want := []string{"terminal", "web"}
	if !reflect.DeepEqual(status.RuntimeToolsets, want) {
		t.Fatalf("api_server override runtime toolsets = %v, want %v", status.RuntimeToolsets, want)
	}
}
