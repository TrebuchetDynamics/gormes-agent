package schemas

import "encoding/json"

// EmptyObjectToolSchema returns the provider-safe fallback schema for tools with
// absent, invalid, or non-object parameter schemas.
func EmptyObjectToolSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{}}`)
}
