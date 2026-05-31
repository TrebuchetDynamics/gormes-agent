package events

import eventinbound "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/inbound"

// Attachment is the channel-neutral media descriptor attached to an inbound
// event. SourceID preserves the platform-side media identifier so failures can
// still be diagnosed even when URL resolution fails.
type Attachment = eventinbound.Attachment

// SubmitText builds the channel-neutral kernel submit body from message text,
// reply context, and normalized attachments.
func SubmitText(text string, replyToText string, attachments []Attachment) string {
	return eventinbound.SubmitText(text, replyToText, attachments)
}
