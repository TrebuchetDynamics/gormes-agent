package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/input"

var (
	ErrSendMessageMissingBody = input.ErrSendMessageMissingBody
	ErrSendMessageInvalidText = input.ErrSendMessageInvalidText
)

type SendMessageBodyOptions = input.SendMessageBodyOptions

type SendMessageBody = input.SendMessageBody

func ResolveSendMessageBody(opts SendMessageBodyOptions) (SendMessageBody, error) {
	return input.ResolveSendMessageBody(opts)
}
