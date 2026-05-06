package slack

import "context"

type Event struct {
	RequestID      string
	ChannelID      string
	TeamID         string
	UserID         string
	Text           string
	ChatType       string
	Timestamp      string
	ThreadTS       string
	SubType        string
	BotID          string
	Blocks         []SlackBlock
	Attachments    []SlackAttachmentPreview
	ThreadReplies  []ThreadMessage
	ApprovalAction *ApprovalAction
}

type ApprovalAction struct {
	ActionID   string
	SessionKey string
	MessageTS  string
	ChannelID  string
	UserID     string
	UserName   string
}

type Client interface {
	AuthTest(context.Context) (string, error)
	Run(context.Context, func(Event)) error
	Ack(requestID string) error
	PostMessage(ctx context.Context, channelID, threadTS, text string) (string, error)
	UpdateMessage(ctx context.Context, channelID, ts, text string) error
	UploadFile(ctx context.Context, channelID, threadTS, filePath string) (string, error)
}

func SessionKey(channelID string) string {
	return "slack:" + channelID
}
