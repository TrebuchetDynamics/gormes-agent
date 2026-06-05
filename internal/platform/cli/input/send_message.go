package input

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/input/messagebody"

var (
	ErrSendMessageMissingBody = messagebody.ErrSendMessageMissingBody
	ErrSendMessageInvalidText = messagebody.ErrSendMessageInvalidText
)

type SendMessageBodyOptions = messagebody.SendMessageBodyOptions
type SendMessageBody = messagebody.SendMessageBody

func ResolveSendMessageBody(opts SendMessageBodyOptions) (SendMessageBody, error) {
	return messagebody.ResolveSendMessageBody(opts)
}
