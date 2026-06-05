package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/sendmessage"

type SendMessageTool = sendmessage.SendMessageTool

func NewSendMessageTool(fn func(target, message string) error) *SendMessageTool {
	return sendmessage.NewSendMessageTool(fn)
}
