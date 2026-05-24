package gonchotools

import (
	"reflect"
	"strings"
	"testing"
)

func TestHonchoMCPCatalog_CoversEveryUpstreamToolName(t *testing.T) {
	got := honchoMCPCatalogNames(HonchoMCPToolCatalog())
	want := []string{
		"inspect_workspace",
		"list_workspaces",
		"search",
		"get_metadata",
		"set_metadata",
		"create_peer",
		"list_peers",
		"chat",
		"get_peer_card",
		"set_peer_card",
		"get_peer_context",
		"get_representation",
		"create_session",
		"list_sessions",
		"delete_session",
		"clone_session",
		"add_peers_to_session",
		"remove_peers_from_session",
		"get_session_peers",
		"inspect_session",
		"add_messages_to_session",
		"get_session_messages",
		"get_session_message",
		"get_session_context",
		"list_conclusions",
		"query_conclusions",
		"create_conclusions",
		"delete_conclusion",
		"schedule_dream",
		"get_queue_status",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("HonchoMCPToolCatalog names = %#v, want %#v", got, want)
	}
}

func TestHonchoMCPCatalog_MappedRowsPointAtRegisteredGonchoDescriptors(t *testing.T) {
	reg, _, cleanup := newTestHonchoRegistry(t)
	defer cleanup()

	for _, entry := range HonchoMCPToolCatalog() {
		if entry.Status != HonchoMCPToolMapped {
			continue
		}
		if entry.GonchoTool == "" {
			t.Fatalf("%s mapped without GonchoTool", entry.Name)
		}
		if _, ok := reg.Get(entry.GonchoTool); !ok {
			t.Fatalf("%s maps to unregistered Goncho tool %q", entry.Name, entry.GonchoTool)
		}
	}
}

func TestHonchoMCPCatalog_UnsupportedRowsCarryRationaleAndSource(t *testing.T) {
	statusCounts := map[HonchoMCPToolStatus]int{}
	for _, entry := range HonchoMCPToolCatalog() {
		statusCounts[entry.Status]++
		if entry.Name == "" || entry.Module == "" || entry.SourcePath == "" {
			t.Fatalf("catalog entry missing source identity: %+v", entry)
		}
		switch entry.Status {
		case HonchoMCPToolMapped:
			if entry.GonchoTool == "" || entry.Rationale == "" || len(entry.UnsupportedInputs) != 0 {
				t.Fatalf("mapped entry missing descriptor/rationale: %+v", entry)
			}
		case HonchoMCPToolPartial:
			if entry.GonchoTool == "" || entry.Rationale == "" || len(entry.UnsupportedInputs) == 0 {
				t.Fatalf("partial entry missing descriptor/rationale/unsupported inputs: %+v", entry)
			}
		case HonchoMCPToolUnsupported:
			if entry.UnsupportedReason == "" {
				t.Fatalf("unsupported entry missing reason: %+v", entry)
			}
		default:
			t.Fatalf("unknown status for %s: %q", entry.Name, entry.Status)
		}
	}
	if statusCounts[HonchoMCPToolMapped] == 0 {
		t.Fatal("catalog has no mapped rows")
	}
	if statusCounts[HonchoMCPToolPartial] == 0 {
		t.Fatal("catalog has no partial rows")
	}
	if statusCounts[HonchoMCPToolUnsupported] == 0 {
		t.Fatal("catalog has no unsupported rows")
	}
}

func TestHonchoMCPCatalog_RecordsRepresentativeInputContracts(t *testing.T) {
	entries := map[string]HonchoMCPToolCatalogEntry{}
	for _, entry := range HonchoMCPToolCatalog() {
		entries[entry.Name] = entry
	}

	assertCatalogInputs(t, entries["chat"], []string{"peer_id", "query"}, []string{"target_peer_id", "session_id", "reasoning_level"})
	assertCatalogInputs(t, entries["create_conclusions"], []string{"peer_id", "target_peer_id", "conclusions"}, []string{"session_id"})
	assertCatalogInputs(t, entries["add_peers_to_session"], []string{"session_id", "peers"}, nil)
	assertCatalogInputs(t, entries["list_peers"], nil, nil)
}

func TestHonchoMCPCatalog_PartialRowsRecordFieldLevelDegradation(t *testing.T) {
	entries := map[string]HonchoMCPToolCatalogEntry{}
	for _, entry := range HonchoMCPToolCatalog() {
		entries[entry.Name] = entry
	}

	assertPartialInput(t, entries["chat"], "honcho_chat", "target_peer_id", "target-specific dialectic reasoning")
	assertPartialInput(t, entries["get_peer_context"], "honcho_context", "target_peer_id", "directional representation")
	assertPartialInput(t, entries["get_peer_context"], "honcho_context", "max_conclusions", "unavailable evidence")

	if got := entries["get_representation"]; got.Status != HonchoMCPToolUnsupported || got.GonchoTool != "" {
		t.Fatalf("get_representation = %+v, want unsupported because honcho_context returns a broader context object", got)
	}
	if got := entries["query_conclusions"]; got.Status != HonchoMCPToolUnsupported || got.GonchoTool != "" {
		t.Fatalf("query_conclusions = %+v, want unsupported because honcho_search lacks target/top_k conclusion-object semantics", got)
	}
}

func TestHonchoMCPCatalog_ReturnsDefensiveCopy(t *testing.T) {
	first := HonchoMCPToolCatalog()
	first[0].Name = "mutated"
	first[2].RequiredInputs[0] = "mutated"
	first[7].UnsupportedInputs["target_peer_id"] = "mutated"

	second := HonchoMCPToolCatalog()
	if second[0].Name == "mutated" {
		t.Fatal("HonchoMCPToolCatalog leaked mutable entry slice")
	}
	if second[2].RequiredInputs[0] == "mutated" {
		t.Fatal("HonchoMCPToolCatalog leaked mutable RequiredInputs slice")
	}
	if second[7].UnsupportedInputs["target_peer_id"] == "mutated" {
		t.Fatal("HonchoMCPToolCatalog leaked mutable UnsupportedInputs map")
	}
}

func honchoMCPCatalogNames(entries []HonchoMCPToolCatalogEntry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name)
	}
	return out
}

func assertCatalogInputs(t *testing.T, entry HonchoMCPToolCatalogEntry, required, optional []string) {
	t.Helper()
	if !reflect.DeepEqual(entry.RequiredInputs, required) {
		t.Fatalf("%s RequiredInputs = %#v, want %#v", entry.Name, entry.RequiredInputs, required)
	}
	if !reflect.DeepEqual(entry.OptionalInputs, optional) {
		t.Fatalf("%s OptionalInputs = %#v, want %#v", entry.Name, entry.OptionalInputs, optional)
	}
}

func assertPartialInput(t *testing.T, entry HonchoMCPToolCatalogEntry, gonchoTool, input, reasonFragment string) {
	t.Helper()
	if entry.Status != HonchoMCPToolPartial {
		t.Fatalf("%s status = %q, want partial", entry.Name, entry.Status)
	}
	if entry.GonchoTool != gonchoTool {
		t.Fatalf("%s GonchoTool = %q, want %q", entry.Name, entry.GonchoTool, gonchoTool)
	}
	reason := entry.UnsupportedInputs[input]
	if reason == "" {
		t.Fatalf("%s missing unsupported input %q: %+v", entry.Name, input, entry.UnsupportedInputs)
	}
	if !strings.Contains(reason, reasonFragment) {
		t.Fatalf("%s unsupported input %q reason = %q, want fragment %q", entry.Name, input, reason, reasonFragment)
	}
}
