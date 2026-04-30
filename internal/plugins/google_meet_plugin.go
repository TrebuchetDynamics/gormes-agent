package plugins

// LoadGoogleMeet loads the first-party Google Meet plugin metadata from dir
// and augments the resulting PluginStatus with the realtime, remote-node, and
// safety evidence the upstream plugin contract requires. No browser, audio
// device, OpenAI Realtime call, node websocket, Python import, or meeting
// join is performed.
func LoadGoogleMeet(dir string, opts LoadOptions) PluginStatus {
	status := LoadDir(dir, opts)
	if status.Manifest.Name != "google_meet" {
		return status
	}

	runtime := []Evidence{
		evidence(EvidenceGoogleMeetRuntimeUnavailable, "playwright", "playwright Python package and Chromium browser are required to run the meet bot"),
		evidence(EvidenceBrowserProfileRequired, "chromium", "headless Chromium with a signed-in Google profile or guest-join is required before a meeting can be entered"),
	}
	realtime := []Evidence{
		evidence(EvidenceGoogleMeetRealtimeUnconfigured, "OPENAI_API_KEY", "OpenAI Realtime API key is required for meet_say duplex audio"),
		evidence(EvidenceGoogleMeetRealtimeUnconfigured, "audio_bridge", "PulseAudio null-sink (linux) or BlackHole (macos) virtual audio device is required for realtime mode"),
	}
	node := []Evidence{
		evidence(EvidenceNodeAuthRequired, "bearer_token", "remote node host requires an approved bearer token via 'hermes meet node approve'"),
		evidence(EvidenceExecutionDisabled, "runtime", "remote node runtime execution is disabled for metadata-only discovery"),
	}
	safety := []Evidence{
		evidence(EvidenceMeetURLGate, "url", "meet_join only accepts explicit https://meet.google.com/ URLs passed in by the operator"),
		evidence(EvidenceNoCalendarAutoDial, "calendar", "the plugin never scans Google Calendar and never auto-dials meetings"),
		evidence(EvidenceOneActiveMeeting, "meet_join", "only one active meeting bot is permitted at a time"),
	}

	status.Evidence = append(status.Evidence, runtime...)
	status.Evidence = append(status.Evidence, realtime...)
	status.Evidence = append(status.Evidence, node...)
	status.Evidence = append(status.Evidence, safety...)

	for i := range status.Capabilities {
		if status.Capabilities[i].Kind == CapabilityTool {
			status.Capabilities[i].Evidence = append(status.Capabilities[i].Evidence, runtime...)
		}
		if status.Capabilities[i].Kind == CapabilityTool && status.Capabilities[i].Name == "meet_say" {
			status.Capabilities[i].Evidence = append(status.Capabilities[i].Evidence,
				evidence(EvidenceMeetSayRealtimeRequired, "mode", "meet_say requires the active meeting to be joined with mode='realtime'"),
			)
			status.Capabilities[i].Evidence = append(status.Capabilities[i].Evidence, realtime...)
		}
	}

	status.Capabilities = append(status.Capabilities, CapabilityStatus{
		Plugin:   "google_meet",
		Kind:     CapabilityRealtime,
		Name:     "google_meet_realtime_audio",
		State:    StateDisabled,
		Evidence: append(append([]Evidence(nil), realtime...), evidence(EvidenceExecutionDisabled, "runtime", "realtime audio runtime execution is disabled for metadata-only discovery")),
	})
	status.Capabilities = append(status.Capabilities, CapabilityStatus{
		Plugin:   "google_meet",
		Kind:     CapabilityRemoteNode,
		Name:     "google_meet_node",
		State:    StateDisabled,
		Evidence: cloneEvidence(node),
	})

	status.Manifest.Capabilities = append(status.Manifest.Capabilities,
		Capability{Kind: CapabilityRealtime, Name: "google_meet_realtime_audio", SourceField: "google_meet:realtime"},
		Capability{Kind: CapabilityRemoteNode, Name: "google_meet_node", SourceField: "google_meet:node"},
	)

	return sortPluginStatus(status)
}

// GoogleMeetPluginYAML mirrors the upstream Hermes plugin manifest at
// hermes-agent/plugins/google_meet/plugin.yaml@df3c9593.
const GoogleMeetPluginYAML = `name: google_meet
version: 0.2.0
description: "Join a Google Meet call, transcribe live captions, speak in realtime, and follow up afterwards. Explicit-by-design: only joins meet.google.com URLs passed in - no calendar scanning, no auto-dial."
author: NousResearch
kind: standalone
platforms:
  - linux
  - macos
provides_tools:
  - meet_join
  - meet_leave
  - meet_status
  - meet_transcript
  - meet_say
hooks:
  - on_session_end
`

// GoogleMeetPluginInitPy mirrors the relevant tool-row table from upstream
// __init__.py without any of the runtime imports or platform side-effects.
const GoogleMeetPluginInitPy = `from plugins.google_meet.tools import (
    MEET_JOIN_SCHEMA,
    MEET_LEAVE_SCHEMA,
    MEET_SAY_SCHEMA,
    MEET_STATUS_SCHEMA,
    MEET_TRANSCRIPT_SCHEMA,
    check_meet_requirements,
    handle_meet_join,
    handle_meet_leave,
    handle_meet_say,
    handle_meet_status,
    handle_meet_transcript,
)

_TOOLS = (
    ("meet_join",       MEET_JOIN_SCHEMA,       handle_meet_join,       "phone"),
    ("meet_status",     MEET_STATUS_SCHEMA,     handle_meet_status,     "status"),
    ("meet_transcript", MEET_TRANSCRIPT_SCHEMA, handle_meet_transcript, "note"),
    ("meet_leave",      MEET_LEAVE_SCHEMA,      handle_meet_leave,      "wave"),
    ("meet_say",        MEET_SAY_SCHEMA,        handle_meet_say,        "speaker"),
)

def register(ctx) -> None:
    for name, schema, handler, emoji in _TOOLS:
        ctx.register_tool(
            name=name,
            toolset="google_meet",
            schema=schema,
            handler=handler,
            check_fn=check_meet_requirements,
            emoji=emoji,
        )
`

// GoogleMeetPluginToolsPy mirrors the schema constants and tool_result/error
// envelopes from upstream tools.py. Handler bodies are intentionally absent;
// the loader never executes them.
const GoogleMeetPluginToolsPy = `from tools.registry import tool_error, tool_result

MEET_JOIN_SCHEMA = {
    "name": "meet_join",
    "description": "Join a Google Meet call and start scraping live captions into a transcript file. Only meet.google.com URLs are accepted; no calendar scanning, no auto-dial.",
    "parameters": {
        "type": "object",
        "properties": {
            "url": {"type": "string", "description": "Full https://meet.google.com/... URL. Required."},
            "mode": {"type": "string", "enum": ["transcribe", "realtime"]},
            "guest_name": {"type": "string"},
            "duration": {"type": "string"},
            "headed": {"type": "boolean"},
            "node": {"type": "string"},
        },
        "required": ["url"],
        "additionalProperties": False,
    },
}

MEET_STATUS_SCHEMA = {
    "name": "meet_status",
    "description": "Report the current Meet session state.",
    "parameters": {
        "type": "object",
        "properties": {"node": {"type": "string"}},
        "additionalProperties": False,
    },
}

MEET_TRANSCRIPT_SCHEMA = {
    "name": "meet_transcript",
    "description": "Read the scraped transcript for the active Meet session.",
    "parameters": {
        "type": "object",
        "properties": {
            "last": {"type": "integer", "minimum": 1},
            "node": {"type": "string"},
        },
        "additionalProperties": False,
    },
}

MEET_LEAVE_SCHEMA = {
    "name": "meet_leave",
    "description": "Leave the active Meet call cleanly.",
    "parameters": {
        "type": "object",
        "properties": {"node": {"type": "string"}},
        "additionalProperties": False,
    },
}

MEET_SAY_SCHEMA = {
    "name": "meet_say",
    "description": "Speak text into the active Meet call. Requires the active meeting to have been joined with mode='realtime'.",
    "parameters": {
        "type": "object",
        "properties": {
            "text": {"type": "string"},
            "node": {"type": "string"},
        },
        "required": ["text"],
        "additionalProperties": False,
    },
}
`
