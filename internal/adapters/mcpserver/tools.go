package mcpserver

// MCPToolDescriptor describes a single tool exposed by the MCP server.
type MCPToolDescriptor struct {
	Name        string
	Description string
	Schema      map[string]interface{}
}

// MCPTools returns the list of all tools exposed by this MCP server.
// These correspond to the Hermes 9-tool MCP surface plus the
// channels_list tool from the Python implementation.
func MCPTools() []MCPToolDescriptor {
	return []MCPToolDescriptor{
		{
			Name:        "conversations_list",
			Description: "List active messaging conversations across connected platforms. Returns conversations with session keys, platform, chat type, display name, and last activity time.",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"platform": map[string]interface{}{
						"type":        "string",
						"description": "Filter by platform name (e.g. telegram, discord, slack)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of conversations to return",
						"default":     50,
					},
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Optional text to filter conversations by name",
					},
				},
			},
		},
		{
			Name:        "conversation_get",
			Description: "Get detailed info about one conversation by its session key.",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_key": map[string]interface{}{
						"type":        "string",
						"description": "The session key from conversations_list",
					},
				},
				"required": []string{"session_key"},
			},
		},
		{
			Name:        "messages_read",
			Description: "Read recent messages from a conversation. Returns message history in chronological order with role, content, and timestamp.",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_key": map[string]interface{}{
						"type":        "string",
						"description": "The session key from conversations_list",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of messages to return",
						"default":     50,
					},
				},
				"required": []string{"session_key"},
			},
		},
		{
			Name:        "attachments_fetch",
			Description: "List non-text attachments for a message. Extracts images, media files, and other non-text content blocks.",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"session_key": map[string]interface{}{
						"type":        "string",
						"description": "The session key from conversations_list",
					},
					"message_id": map[string]interface{}{
						"type":        "string",
						"description": "The message ID from messages_read",
					},
				},
				"required": []string{"session_key", "message_id"},
			},
		},
		{
			Name:        "events_poll",
			Description: "Poll for new conversation events since a cursor position. Returns events (message, approval_requested, approval_resolved).",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"after_cursor": map[string]interface{}{
						"type":        "integer",
						"description": "Return events after this cursor (0 for all)",
						"default":     0,
					},
					"session_key": map[string]interface{}{
						"type":        "string",
						"description": "Optional filter to one conversation",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum events to return",
						"default":     20,
					},
				},
			},
		},
		{
			Name:        "events_wait",
			Description: "Wait for the next conversation event (long-poll). Blocks until a matching event arrives or timeout expires.",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"after_cursor": map[string]interface{}{
						"type":        "integer",
						"description": "Wait for events after this cursor",
						"default":     0,
					},
					"session_key": map[string]interface{}{
						"type":        "string",
						"description": "Optional filter to one conversation",
					},
					"timeout_ms": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum wait time in milliseconds",
						"default":     30000,
					},
				},
			},
		},
		{
			Name:        "messages_send",
			Description: "Send a message to a platform conversation. Target format is 'platform:chat_id' (e.g. telegram:6308981865).",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]interface{}{
						"type":        "string",
						"description": "Platform target in 'platform:identifier' format",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "The message text to send",
					},
				},
				"required": []string{"target", "message"},
			},
		},
		{
			Name:        "permissions_list_open",
			Description: "List pending approval requests observed during this bridge session.",
			Schema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "permissions_respond",
			Description: "Respond to a pending approval request. Decisions: allow-once, allow-always, deny.",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id": map[string]interface{}{
						"type":        "string",
						"description": "The approval ID from permissions_list_open",
					},
					"decision": map[string]interface{}{
						"type":        "string",
						"description": "One of: allow-once, allow-always, deny",
						"enum":        []string{"allow-once", "allow-always", "deny"},
					},
				},
				"required": []string{"id", "decision"},
			},
		},
	}
}
