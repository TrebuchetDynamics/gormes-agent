package slack

import slackrichtext "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/slack/richtext"

type SlackBlock = slackrichtext.Block

type SlackAttachmentPreview = slackrichtext.AttachmentPreview

type SlackRichTextEvidence = slackrichtext.Evidence

const slackRichTextUnavailableCode = slackrichtext.UnavailableCode

func augmentInboundText(text string, blocks []SlackBlock, attachments []SlackAttachmentPreview) (string, []SlackRichTextEvidence) {
	return slackrichtext.AugmentInboundText(text, blocks, attachments)
}
