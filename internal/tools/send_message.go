package tools

import (
	"context"
	"encoding/json"
	"time"
)

type SendMessageTool struct {
	sendFn func(target, message string) error
}

func NewSendMessageTool(fn func(target, message string) error) *SendMessageTool {
	return &SendMessageTool{sendFn: fn}
}

func (*SendMessageTool) Name() string { return "send_message" }
func (*SendMessageTool) Description() string {
	return "Send a message through configured gateway channels. Use to notify users on Telegram, Discord, Slack, etc."
}
func (*SendMessageTool) Timeout() time.Duration { return 10 * time.Second }

func (*SendMessageTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"target":{"type":"string","description":"Target platform and chat ID, e.g. 'telegram:123456' or 'discord:channel_id'"},
			"message":{"type":"string","description":"The message content to send"}
		},
		"required":["target","message"]
	}`)
}

func (t *SendMessageTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Target  string `json:"target"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, err
	}
	if t.sendFn != nil {
		if err := t.sendFn(in.Target, in.Message); err != nil {
			return json.Marshal(map[string]any{
				"success": false,
				"target":  in.Target,
				"error":   err.Error(),
			})
		}
	}
	return json.Marshal(map[string]any{
		"success": true,
		"target":  in.Target,
		"message": "Message sent successfully.",
	})
}
