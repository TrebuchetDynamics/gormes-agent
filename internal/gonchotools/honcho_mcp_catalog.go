package gonchotools

// HonchoMCPToolStatus records whether an upstream Honcho MCP tool is covered
// by an existing Goncho tool descriptor or remains an explicit compatibility
// gap.
type HonchoMCPToolStatus string

const (
	HonchoMCPToolMapped      HonchoMCPToolStatus = "mapped"
	HonchoMCPToolPartial     HonchoMCPToolStatus = "partial"
	HonchoMCPToolUnsupported HonchoMCPToolStatus = "unsupported"
)

// HonchoMCPToolCatalogEntry is a source-backed compatibility row for one tool
// registered by ../honcho/mcp/src/tools/*.ts.
type HonchoMCPToolCatalogEntry struct {
	Name              string
	Module            string
	SourcePath        string
	RequiredInputs    []string
	OptionalInputs    []string
	Status            HonchoMCPToolStatus
	GonchoTool        string
	Rationale         string
	UnsupportedInputs map[string]string
	UnsupportedReason string
}

// HonchoMCPToolCatalog returns the upstream Honcho MCP tool matrix in the same
// module order used by mcp/src/server.ts: workspace, peers, sessions,
// conclusions, then system.
func HonchoMCPToolCatalog() []HonchoMCPToolCatalogEntry {
	return cloneHonchoMCPToolCatalog(honchoMCPToolCatalog)
}

var honchoMCPToolCatalog = []HonchoMCPToolCatalogEntry{
	unsupportedMCPTool("inspect_workspace", "workspace", "../honcho/mcp/src/tools/workspace.ts", nil, nil, "workspace metadata, configuration, peer/session pagination inspection is not exposed through a Goncho tool descriptor yet"),
	unsupportedMCPTool("list_workspaces", "workspace", "../honcho/mcp/src/tools/workspace.ts", nil, nil, "hosted workspace pagination is outside the current single-local-workspace Goncho runtime"),
	unsupportedMCPTool("search", "workspace", "../honcho/mcp/src/tools/workspace.ts", []string{"query"}, []string{"peer_id", "session_id"}, "workspace-level message search without a required peer is not exposed through honcho_search"),
	unsupportedMCPTool("get_metadata", "workspace", "../honcho/mcp/src/tools/workspace.ts", nil, []string{"peer_id", "session_id"}, "workspace, peer, and session metadata reads do not have a local Goncho tool descriptor yet"),
	unsupportedMCPTool("set_metadata", "workspace", "../honcho/mcp/src/tools/workspace.ts", []string{"metadata"}, []string{"peer_id", "session_id"}, "workspace, peer, and session metadata writes do not have a local Goncho tool descriptor yet"),

	unsupportedMCPTool("create_peer", "peers", "../honcho/mcp/src/tools/peers.ts", []string{"peer_id"}, []string{"configuration"}, "peer creation and observeMe configuration are not exposed as a local Goncho tool descriptor yet"),
	unsupportedMCPTool("list_peers", "peers", "../honcho/mcp/src/tools/peers.ts", nil, nil, "peer pagination is not exposed as a local Goncho tool descriptor yet"),
	partialMCPTool("chat", "peers", "../honcho/mcp/src/tools/peers.ts", []string{"peer_id", "query"}, []string{"target_peer_id", "session_id", "reasoning_level"}, "honcho_chat", "peer.chat-compatible non-streaming content is exposed by honcho_chat", map[string]string{
		"target_peer_id": "current Goncho chat reports target-specific dialectic reasoning unavailable and falls back to default observer recall",
	}),
	mappedMCPTool("get_peer_card", "peers", "../honcho/mcp/src/tools/peers.ts", []string{"peer_id"}, []string{"target_peer_id"}, "honcho_profile", "peer-card reads map to honcho_profile without a card payload"),
	mappedMCPTool("set_peer_card", "peers", "../honcho/mcp/src/tools/peers.ts", []string{"peer_id", "peer_card"}, []string{"target_peer_id"}, "honcho_profile", "peer-card writes map to honcho_profile with a card payload"),
	partialMCPTool("get_peer_context", "peers", "../honcho/mcp/src/tools/peers.ts", []string{"peer_id"}, []string{"target_peer_id", "search_query", "max_conclusions"}, "honcho_context", "peer context maps to honcho_context representation and peer-card output for the default observer view", map[string]string{
		"target_peer_id":  "current Goncho context reports directional representation unavailable and uses the default observer view",
		"max_conclusions": "current Goncho context records max_conclusions as unavailable evidence instead of enforcing the upstream cap",
	}),
	unsupportedMCPTool("get_representation", "peers", "../honcho/mcp/src/tools/peers.ts", []string{"peer_id"}, []string{"target_peer_id", "session_id", "search_query", "max_conclusions"}, "upstream returns a representation string with target/session/search/max filters; current honcho_context returns a broader context object and does not cover those semantics"),

	unsupportedMCPTool("create_session", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id"}, nil, "session creation is implicit through Gormes session/message writes and has no Goncho tool descriptor yet"),
	unsupportedMCPTool("list_sessions", "sessions", "../honcho/mcp/src/tools/sessions.ts", nil, nil, "session pagination is not exposed as a local Goncho tool descriptor yet"),
	unsupportedMCPTool("delete_session", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id"}, nil, "destructive session deletion is not exposed as a Goncho tool descriptor yet"),
	unsupportedMCPTool("clone_session", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id"}, []string{"message_id"}, "session cloning exists as a transcript helper but is not exposed through a Goncho tool descriptor yet"),
	unsupportedMCPTool("add_peers_to_session", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id", "peers"}, nil, "per-session peer membership and observe flags are not exposed as Goncho tool descriptors yet"),
	unsupportedMCPTool("remove_peers_from_session", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id", "peer_ids"}, nil, "per-session peer removal is not exposed as a Goncho tool descriptor yet"),
	unsupportedMCPTool("get_session_peers", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id"}, nil, "session peer listing is not exposed as a Goncho tool descriptor yet"),
	unsupportedMCPTool("inspect_session", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id"}, nil, "session inspection with peers, message counts, and summaries is not exposed as a Goncho tool descriptor yet"),
	unsupportedMCPTool("add_messages_to_session", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id", "messages"}, nil, "session message insertion is owned by runtime ingestion and not exposed as an agent-callable Goncho tool yet"),
	unsupportedMCPTool("get_session_messages", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id"}, []string{"filters"}, "paginated session-message reads are not exposed as a Goncho tool descriptor yet"),
	unsupportedMCPTool("get_session_message", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id", "message_id"}, nil, "single-message reads are not exposed as a Goncho tool descriptor yet"),
	unsupportedMCPTool("get_session_context", "sessions", "../honcho/mcp/src/tools/sessions.ts", []string{"session_id"}, []string{"summary", "tokens"}, "session-only context without an explicit peer is not exposed through honcho_context yet"),

	unsupportedMCPTool("list_conclusions", "conclusions", "../honcho/mcp/src/tools/conclusions.ts", []string{"peer_id"}, []string{"target_peer_id"}, "paginated conclusion listing is not exposed as a Goncho tool descriptor yet"),
	unsupportedMCPTool("query_conclusions", "conclusions", "../honcho/mcp/src/tools/conclusions.ts", []string{"peer_id", "query"}, []string{"target_peer_id", "top_k"}, "upstream searches conclusion objects with target/top_k semantics; current honcho_search returns local memory search results without target_peer_id or top_k schema"),
	unsupportedMCPTool("create_conclusions", "conclusions", "../honcho/mcp/src/tools/conclusions.ts", []string{"peer_id", "target_peer_id", "conclusions"}, []string{"session_id"}, "bulk target-scoped conclusion creation is not exposed through honcho_conclude yet"),
	unsupportedMCPTool("delete_conclusion", "conclusions", "../honcho/mcp/src/tools/conclusions.ts", []string{"peer_id", "target_peer_id", "conclusion_id"}, nil, "target-scoped conclusion deletion by upstream string id is not exposed through honcho_conclude yet"),

	unsupportedMCPTool("schedule_dream", "system", "../honcho/mcp/src/tools/system.ts", []string{"peer_id"}, []string{"target_peer_id", "session_id"}, "dream scheduling exists as a service/runtime surface but has no Goncho tool descriptor yet"),
	unsupportedMCPTool("get_queue_status", "system", "../honcho/mcp/src/tools/system.ts", nil, nil, "queue status exists as a read model but has no agent-callable Goncho tool descriptor yet"),
}

func mappedMCPTool(name, module, sourcePath string, requiredInputs, optionalInputs []string, gonchoTool, rationale string) HonchoMCPToolCatalogEntry {
	return HonchoMCPToolCatalogEntry{
		Name:           name,
		Module:         module,
		SourcePath:     sourcePath,
		RequiredInputs: cloneStringSlice(requiredInputs),
		OptionalInputs: cloneStringSlice(optionalInputs),
		Status:         HonchoMCPToolMapped,
		GonchoTool:     gonchoTool,
		Rationale:      rationale,
	}
}

func partialMCPTool(name, module, sourcePath string, requiredInputs, optionalInputs []string, gonchoTool, rationale string, unsupportedInputs map[string]string) HonchoMCPToolCatalogEntry {
	entry := mappedMCPTool(name, module, sourcePath, requiredInputs, optionalInputs, gonchoTool, rationale)
	entry.Status = HonchoMCPToolPartial
	entry.UnsupportedInputs = cloneStringMap(unsupportedInputs)
	return entry
}

func unsupportedMCPTool(name, module, sourcePath string, requiredInputs, optionalInputs []string, reason string) HonchoMCPToolCatalogEntry {
	return HonchoMCPToolCatalogEntry{
		Name:              name,
		Module:            module,
		SourcePath:        sourcePath,
		RequiredInputs:    cloneStringSlice(requiredInputs),
		OptionalInputs:    cloneStringSlice(optionalInputs),
		Status:            HonchoMCPToolUnsupported,
		UnsupportedReason: reason,
	}
}

func cloneHonchoMCPToolCatalog(in []HonchoMCPToolCatalogEntry) []HonchoMCPToolCatalogEntry {
	out := make([]HonchoMCPToolCatalogEntry, len(in))
	for i, entry := range in {
		out[i] = entry
		out[i].RequiredInputs = cloneStringSlice(entry.RequiredInputs)
		out[i].OptionalInputs = cloneStringSlice(entry.OptionalInputs)
		out[i].UnsupportedInputs = cloneStringMap(entry.UnsupportedInputs)
	}
	return out
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
