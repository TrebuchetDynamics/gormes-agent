package tuigateway

// PlatformEventKind discriminates the gateway↔TUI platform events.
//
// The string values are the canonical kind tags carried over the wire by
// the future remote TUI transport. They intentionally match no JSON-RPC
// method name 1:1 — transport assembly is a separate concern. Today these
// constants exist only to give the round-trip JSON tests in
// media_personality_event_test.go a stable schema, plus a discriminator
// callers can switch on once dispatch is wired in.
type PlatformEventKind string

const (
	// PlatformEventKindSubmit corresponds to a user-initiated prompt
	// submission (upstream: prompt.submit JSON-RPC method).
	PlatformEventKindSubmit PlatformEventKind = "submit"
	// PlatformEventKindCancel corresponds to a session-scoped interrupt
	// (upstream: session.interrupt JSON-RPC method).
	PlatformEventKindCancel PlatformEventKind = "cancel"
	// PlatformEventKindResize carries terminal column updates from the TUI
	// (upstream: terminal.resize JSON-RPC method).
	PlatformEventKindResize PlatformEventKind = "resize"
	// PlatformEventKindProgress is a server→client progress notification
	// (upstream: tool.progress emit payload).
	PlatformEventKindProgress PlatformEventKind = "progress"
	// PlatformEventKindImageMetadata is the image attachment summary the
	// gateway returns alongside image.attach (upstream: _image_meta).
	PlatformEventKindImageMetadata PlatformEventKind = "image_metadata"
)

// validPlatformEventKinds is the closed enumeration accepted by
// ValidPlatformEventKind. Membership is O(1) for dispatch in callers.
var validPlatformEventKinds = map[PlatformEventKind]struct{}{
	PlatformEventKindSubmit:        {},
	PlatformEventKindCancel:        {},
	PlatformEventKindResize:        {},
	PlatformEventKindProgress:      {},
	PlatformEventKindImageMetadata: {},
}

// ValidPlatformEventKind reports whether k is one of the recognised
// PlatformEventKind constants. It is O(1) and side-effect free; callers
// use it to guard JSON-RPC dispatch without a switch ladder.
func ValidPlatformEventKind(k PlatformEventKind) bool {
	_, ok := validPlatformEventKinds[k]
	return ok
}

// platformEvent is the shared interface every concrete event type
// satisfies. Today it only exposes the kind so callers can dispatch on a
// generic PlatformEvent without importing the concrete struct types
// individually. Future transport work can extend the interface (e.g. with
// a Validate() method) without breaking callers since each concrete type
// already round-trips JSON via encoding/json's struct support.
type platformEvent interface {
	EventKind() PlatformEventKind
}

// SubmitEvent is the schema for a prompt submission travelling from the
// TUI to the gateway. SessionID identifies the session under which the
// prompt should run; Text is the operator's input text.
type SubmitEvent struct {
	Kind      PlatformEventKind `json:"kind"`
	SessionID string            `json:"session_id"`
	Text      string            `json:"text"`
}

// EventKind reports SubmitEvent's discriminator. Always returns
// PlatformEventKindSubmit when populated through JSON unmarshal of a
// well-formed payload; the zero-value Kind is kept honest by tests.
func (e SubmitEvent) EventKind() PlatformEventKind { return e.Kind }

// CancelEvent is the schema for a session-scoped cancel signal. It only
// carries SessionID — the upstream session.interrupt method has the same
// minimal shape.
type CancelEvent struct {
	Kind      PlatformEventKind `json:"kind"`
	SessionID string            `json:"session_id"`
}

// EventKind reports CancelEvent's discriminator.
func (e CancelEvent) EventKind() PlatformEventKind { return e.Kind }

// ResizeEvent carries a terminal column update from the TUI. Cols is the
// integer column count Bubble Tea reports during SIGWINCH and friends.
type ResizeEvent struct {
	Kind      PlatformEventKind `json:"kind"`
	SessionID string            `json:"session_id"`
	Cols      int               `json:"cols"`
}

// EventKind reports ResizeEvent's discriminator.
func (e ResizeEvent) EventKind() PlatformEventKind { return e.Kind }

// ProgressEvent is the structured payload behind upstream's tool.progress
// emit notifications. ToolName identifies the running tool; Preview is a
// caller-supplied snippet of progress text.
type ProgressEvent struct {
	Kind      PlatformEventKind `json:"kind"`
	SessionID string            `json:"session_id"`
	ToolName  string            `json:"tool_name"`
	Preview   string            `json:"preview,omitempty"`
}

// EventKind reports ProgressEvent's discriminator.
func (e ProgressEvent) EventKind() PlatformEventKind { return e.Kind }

// ImageMetadataEvent wraps an ImageMetadata struct as a transport-ready
// platform event. Callers compose ReadImageMetadata's output here when
// they want to surface attachment summaries through the same envelope as
// progress/resize/etc.
type ImageMetadataEvent struct {
	Kind      PlatformEventKind `json:"kind"`
	SessionID string            `json:"session_id"`
	Metadata  ImageMetadata     `json:"metadata"`
}

// EventKind reports ImageMetadataEvent's discriminator.
func (e ImageMetadataEvent) EventKind() PlatformEventKind { return e.Kind }

// Compile-time guards: each concrete event satisfies the platformEvent
// interface so future dispatch code can rely on the contract without
// runtime checks at call sites.
var (
	_ platformEvent = SubmitEvent{}
	_ platformEvent = CancelEvent{}
	_ platformEvent = ResizeEvent{}
	_ platformEvent = ProgressEvent{}
	_ platformEvent = ImageMetadataEvent{}
)
