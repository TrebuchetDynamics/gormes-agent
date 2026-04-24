package signal

import "context"

// Attachment is the transport-neutral outbound media descriptor.
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// SendOptions captures Signal reply metadata and outbound attachments.
type SendOptions struct {
	ReplyToMessageID string
	Attachments      []Attachment
}

// ReceiveError reports a receive-loop observability event from the Signal
// transport. Fatal=true terminates Bot.Run; otherwise the consumer keeps
// draining Events() while the transport owns internal reconnects.
type ReceiveError struct {
	Err   error
	Fatal bool
}

// Transport is the Signal transport/bootstrap surface the shared channel
// depends on. It owns the signal-cli or bridge client lifecycle, the
// receive loop (including internal reconnects), and the outbound send seam
// — including attachments. The live implementation will wrap a signal-cli
// daemon or bridge; Bot never pulls a real transport dependency into
// compile.
type Transport interface {
	// Start bootstraps the underlying signal-cli or bridge session. It
	// must be called exactly once, before Events() or Errors() are
	// consumed.
	Start(ctx context.Context) error

	// Events streams normalized inbound Signal messages. Implementations
	// MUST maintain this channel across internal reconnects; only final
	// shutdown closes it.
	Events() <-chan InboundMessage

	// Errors reports receive-loop observability. Fatal errors cause Bot.Run
	// to exit; transient errors are informational.
	Errors() <-chan ReceiveError

	// SendDirect posts a direct message to a recipient (phone number or
	// UUID), honoring reply metadata and attachment payloads carried in
	// SendOptions.
	SendDirect(ctx context.Context, recipientID, text string, opts SendOptions) (string, error)

	// SendGroup posts a group message using the native group ID, honoring
	// reply metadata and attachment payloads carried in SendOptions.
	SendGroup(ctx context.Context, groupID, text string, opts SendOptions) (string, error)

	// Close tears down the transport. It is safe to call once Run exits.
	Close() error
}
