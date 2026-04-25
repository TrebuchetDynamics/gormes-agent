package plugins

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestFirstPartySpotifyPluginLoadsToolPackageMetadataAndDegradesWithoutAuth(t *testing.T) {
	dir := writePluginFixture(t, "spotify", map[string]string{
		"plugin.yaml": spotifyPluginYAML,
		"__init__.py": spotifyPluginInitPy,
		"tools.py":    spotifyPluginToolsPy,
	})

	status := LoadDir(dir, LoadOptions{
		Source:               SourceBundled,
		CurrentGormesVersion: "1.0.0",
		EnvLookup:            func(string) bool { return false },
		AuthLookup:           func(string) bool { return false },
	})

	if status.RuntimeCodeExecuted {
		t.Fatal("Spotify plugin package metadata load executed runtime code")
	}
	if status.State != StateDisabled {
		t.Fatalf("state = %q, want disabled; evidence=%+v", status.State, status.Evidence)
	}
	if status.Manifest.Name != "spotify" || status.Manifest.Kind != "backend" {
		t.Fatalf("manifest identity = %+v, want spotify backend", status.Manifest)
	}
	if !slices.Equal(status.Manifest.RequiresAuth, []string{"providers.spotify"}) {
		t.Fatalf("requires_auth = %#v, want providers.spotify inferred before handler load", status.Manifest.RequiresAuth)
	}
	assertEvidence(t, status.Evidence, EvidenceMissingCredential, "providers.spotify")
	assertEvidence(t, status.Evidence, EvidenceExecutionDisabled, "runtime")

	wantTools := []string{
		"spotify_albums",
		"spotify_devices",
		"spotify_library",
		"spotify_playback",
		"spotify_playlists",
		"spotify_queue",
		"spotify_search",
	}
	if got := toolMetadataNames(status.Tools); !slices.Equal(got, wantTools) {
		t.Fatalf("tool metadata names = %#v, want %#v", got, wantTools)
	}
	if got := capabilityNames(status.Capabilities, CapabilityTool); !slices.Equal(got, wantTools) {
		t.Fatalf("tool capabilities = %#v, want %#v", got, wantTools)
	}
	for _, name := range wantTools {
		capability := findCapability(status.Capabilities, CapabilityTool, name)
		if capability == nil {
			t.Fatalf("missing capability for %s", name)
		}
		if capability.State != StateDisabled {
			t.Fatalf("%s state = %q, want disabled", name, capability.State)
		}
		assertEvidence(t, capability.Evidence, EvidenceMissingCredential, "providers.spotify")
		assertEvidence(t, capability.Evidence, EvidenceExecutionDisabled, "runtime")

		meta := findToolMetadata(status.Tools, name)
		if meta == nil {
			t.Fatalf("missing tool metadata for %s", name)
		}
		if meta.Toolset != "spotify" {
			t.Fatalf("%s toolset = %q, want spotify", name, meta.Toolset)
		}
		if meta.ResultEnvelope.Encoding != "json-string" {
			t.Fatalf("%s result envelope encoding = %q, want json-string", name, meta.ResultEnvelope.Encoding)
		}
		if !slices.Contains(meta.ResultEnvelope.ErrorFields, "error") {
			t.Fatalf("%s result envelope error fields = %#v, want error", name, meta.ResultEnvelope.ErrorFields)
		}
	}

	search := requireToolMetadata(t, status.Tools, "spotify_search")
	assertSchemaRequired(t, search.Schema, "query")
	assertSchemaProperty(t, search.Schema, "types", "array")
	assertSchemaProperty(t, search.Schema, "include_external", "string")
	if search.Description != "Search the Spotify catalog for tracks, albums, artists, playlists, shows, or episodes." {
		t.Fatalf("spotify_search description = %q", search.Description)
	}

	playback := requireToolMetadata(t, status.Tools, "spotify_playback")
	assertSchemaRequired(t, playback.Schema, "action")
	assertSchemaEnumValue(t, playback.Schema, "action", "get_currently_playing")
	assertSchemaEnumValue(t, playback.Schema, "action", "set_volume")
	assertSchemaProperty(t, playback.Schema, "uris", "array")
	if !slices.Contains(playback.ResultEnvelope.SuccessFields, "success") ||
		!slices.Contains(playback.ResultEnvelope.SuccessFields, "action") ||
		!slices.Contains(playback.ResultEnvelope.SuccessFields, "result") {
		t.Fatalf("spotify_playback success fields = %#v, want success/action/result", playback.ResultEnvelope.SuccessFields)
	}

	library := requireToolMetadata(t, status.Tools, "spotify_library")
	assertSchemaRequired(t, library.Schema, "kind")
	assertSchemaRequired(t, library.Schema, "action")
	assertSchemaEnumValue(t, library.Schema, "kind", "tracks")
	assertSchemaEnumValue(t, library.Schema, "kind", "albums")
}

func requireToolMetadata(t *testing.T, tools []ToolMetadata, name string) ToolMetadata {
	t.Helper()
	meta := findToolMetadata(tools, name)
	if meta == nil {
		t.Fatalf("missing tool metadata for %s in %+v", name, tools)
	}
	return *meta
}

func findToolMetadata(tools []ToolMetadata, name string) *ToolMetadata {
	for i := range tools {
		if tools[i].Name == name {
			return &tools[i]
		}
	}
	return nil
}

func toolMetadataNames(tools []ToolMetadata) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name)
	}
	return out
}

func capabilityNames(capabilities []CapabilityStatus, kind CapabilityKind) []string {
	out := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability.Kind == kind {
			out = append(out, capability.Name)
		}
	}
	return out
}

func assertSchemaRequired(t *testing.T, schema json.RawMessage, name string) {
	t.Helper()
	var payload struct {
		Parameters struct {
			Required []string `json:"required"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(schema, &payload); err != nil {
		t.Fatalf("schema is invalid JSON: %v\n%s", err, schema)
	}
	if !slices.Contains(payload.Parameters.Required, name) {
		t.Fatalf("schema required = %#v, want %s", payload.Parameters.Required, name)
	}
}

func assertSchemaProperty(t *testing.T, schema json.RawMessage, name, wantType string) {
	t.Helper()
	properties := schemaProperties(t, schema)
	property, ok := properties[name]
	if !ok {
		t.Fatalf("schema properties missing %q in %#v", name, properties)
	}
	if got, _ := property["type"].(string); got != wantType {
		t.Fatalf("schema property %s type = %q, want %q", name, got, wantType)
	}
}

func assertSchemaEnumValue(t *testing.T, schema json.RawMessage, propertyName, want string) {
	t.Helper()
	properties := schemaProperties(t, schema)
	property, ok := properties[propertyName]
	if !ok {
		t.Fatalf("schema properties missing %q in %#v", propertyName, properties)
	}
	values, ok := property["enum"].([]any)
	if !ok {
		t.Fatalf("schema property %s enum missing in %#v", propertyName, property)
	}
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("schema property %s enum = %#v, want %s", propertyName, values, want)
}

func schemaProperties(t *testing.T, schema json.RawMessage) map[string]map[string]any {
	t.Helper()
	var payload struct {
		Parameters struct {
			Properties map[string]map[string]any `json:"properties"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(schema, &payload); err != nil {
		t.Fatalf("schema is invalid JSON: %v\n%s", err, schema)
	}
	return payload.Parameters.Properties
}

const spotifyPluginYAML = `name: spotify
version: 1.0.0
description: "Native Spotify integration - 7 tools (playback, devices, queue, search, playlists, albums, library) using Spotify Web API + PKCE OAuth. Auth via hermes auth spotify. Tools gate on providers.spotify in ~/.hermes/auth.json."
author: NousResearch
kind: backend
provides_tools:
  - spotify_playback
  - spotify_devices
  - spotify_queue
  - spotify_search
  - spotify_playlists
  - spotify_albums
  - spotify_library
`

const spotifyPluginInitPy = `from plugins.spotify.tools import (
    SPOTIFY_ALBUMS_SCHEMA,
    SPOTIFY_DEVICES_SCHEMA,
    SPOTIFY_LIBRARY_SCHEMA,
    SPOTIFY_PLAYBACK_SCHEMA,
    SPOTIFY_PLAYLISTS_SCHEMA,
    SPOTIFY_QUEUE_SCHEMA,
    SPOTIFY_SEARCH_SCHEMA,
    _check_spotify_available,
    _handle_spotify_albums,
    _handle_spotify_devices,
    _handle_spotify_library,
    _handle_spotify_playback,
    _handle_spotify_playlists,
    _handle_spotify_queue,
    _handle_spotify_search,
)

_TOOLS = (
    ("spotify_playback",  SPOTIFY_PLAYBACK_SCHEMA,  _handle_spotify_playback,  "music"),
    ("spotify_devices",   SPOTIFY_DEVICES_SCHEMA,   _handle_spotify_devices,   "speaker"),
    ("spotify_queue",     SPOTIFY_QUEUE_SCHEMA,     _handle_spotify_queue,     "radio"),
    ("spotify_search",    SPOTIFY_SEARCH_SCHEMA,    _handle_spotify_search,    "search"),
    ("spotify_playlists", SPOTIFY_PLAYLISTS_SCHEMA, _handle_spotify_playlists, "library"),
    ("spotify_albums",    SPOTIFY_ALBUMS_SCHEMA,    _handle_spotify_albums,    "album"),
    ("spotify_library",   SPOTIFY_LIBRARY_SCHEMA,   _handle_spotify_library,   "heart"),
)

def register(ctx) -> None:
    for name, schema, handler, emoji in _TOOLS:
        ctx.register_tool(
            name=name,
            toolset="spotify",
            schema=schema,
            handler=handler,
            check_fn=_check_spotify_available,
            emoji=emoji,
        )
`

const spotifyPluginToolsPy = `from hermes_cli.auth import get_auth_status
from tools.registry import tool_error, tool_result

def _check_spotify_available() -> bool:
    return bool(get_auth_status("spotify").get("logged_in"))

COMMON_STRING = {"type": "string"}

SPOTIFY_PLAYBACK_SCHEMA = {
    "name": "spotify_playback",
    "description": "Control Spotify playback, inspect the active playback state, or fetch recently played tracks.",
    "parameters": {
        "type": "object",
        "properties": {
            "action": {"type": "string", "enum": ["get_state", "get_currently_playing", "play", "pause", "next", "previous", "seek", "set_repeat", "set_shuffle", "set_volume", "recently_played"]},
            "device_id": COMMON_STRING,
            "market": COMMON_STRING,
            "context_uri": COMMON_STRING,
            "uris": {"type": "array", "items": COMMON_STRING},
            "offset": {"type": "object"},
            "position_ms": {"type": "integer"},
            "state": {"description": "For set_repeat use track/context/off. For set_shuffle use boolean-like true/false.", "oneOf": [{"type": "string"}, {"type": "boolean"}]},
            "volume_percent": {"type": "integer"},
            "limit": {"type": "integer", "description": "For recently_played: number of tracks (max 50)"},
            "after": {"type": "integer", "description": "For recently_played: Unix ms cursor (after this timestamp)"},
            "before": {"type": "integer", "description": "For recently_played: Unix ms cursor (before this timestamp)"},
        },
        "required": ["action"],
    },
}

SPOTIFY_DEVICES_SCHEMA = {
    "name": "spotify_devices",
    "description": "List Spotify Connect devices or transfer playback to a different device.",
    "parameters": {
        "type": "object",
        "properties": {
            "action": {"type": "string", "enum": ["list", "transfer"]},
            "device_id": COMMON_STRING,
            "play": {"type": "boolean"},
        },
        "required": ["action"],
    },
}

SPOTIFY_QUEUE_SCHEMA = {
    "name": "spotify_queue",
    "description": "Inspect the user's Spotify queue or add an item to it.",
    "parameters": {
        "type": "object",
        "properties": {
            "action": {"type": "string", "enum": ["get", "add"]},
            "uri": COMMON_STRING,
            "device_id": COMMON_STRING,
        },
        "required": ["action"],
    },
}

SPOTIFY_SEARCH_SCHEMA = {
    "name": "spotify_search",
    "description": "Search the Spotify catalog for tracks, albums, artists, playlists, shows, or episodes.",
    "parameters": {
        "type": "object",
        "properties": {
            "query": COMMON_STRING,
            "types": {"type": "array", "items": COMMON_STRING},
            "type": COMMON_STRING,
            "limit": {"type": "integer"},
            "offset": {"type": "integer"},
            "market": COMMON_STRING,
            "include_external": COMMON_STRING,
        },
        "required": ["query"],
    },
}

SPOTIFY_PLAYLISTS_SCHEMA = {
    "name": "spotify_playlists",
    "description": "List, inspect, create, update, and modify Spotify playlists.",
    "parameters": {
        "type": "object",
        "properties": {
            "action": {"type": "string", "enum": ["list", "get", "create", "add_items", "remove_items", "update_details"]},
            "playlist_id": COMMON_STRING,
            "market": COMMON_STRING,
            "limit": {"type": "integer"},
            "offset": {"type": "integer"},
            "name": COMMON_STRING,
            "description": COMMON_STRING,
            "public": {"type": "boolean"},
            "collaborative": {"type": "boolean"},
            "uris": {"type": "array", "items": COMMON_STRING},
            "position": {"type": "integer"},
            "snapshot_id": COMMON_STRING,
        },
        "required": ["action"],
    },
}

SPOTIFY_ALBUMS_SCHEMA = {
    "name": "spotify_albums",
    "description": "Fetch Spotify album metadata or album tracks.",
    "parameters": {
        "type": "object",
        "properties": {
            "action": {"type": "string", "enum": ["get", "tracks"]},
            "album_id": COMMON_STRING,
            "id": COMMON_STRING,
            "market": COMMON_STRING,
            "limit": {"type": "integer"},
            "offset": {"type": "integer"},
        },
        "required": ["action"],
    },
}

SPOTIFY_LIBRARY_SCHEMA = {
    "name": "spotify_library",
    "description": "List, save, or remove the user's saved Spotify tracks or albums. Use kind to select which.",
    "parameters": {
        "type": "object",
        "properties": {
            "kind": {"type": "string", "enum": ["tracks", "albums"], "description": "Which library to operate on"},
            "action": {"type": "string", "enum": ["list", "save", "remove"]},
            "limit": {"type": "integer"},
            "offset": {"type": "integer"},
            "market": COMMON_STRING,
            "uris": {"type": "array", "items": COMMON_STRING},
            "ids": {"type": "array", "items": COMMON_STRING},
            "items": {"type": "array", "items": COMMON_STRING},
        },
        "required": ["kind", "action"],
    },
}
`
