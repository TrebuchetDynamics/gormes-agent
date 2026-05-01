package acp

import "time"

// ACPSession represents an active ACP session.
type ACPSession struct {
	ID        string
	CreatedAt time.Time
	Model     string
	Platform  string
}

// ACPToolKind represents the kind of tool action in ACP protocol.
type ACPToolKind string

const (
	ACPToolKindRead     ACPToolKind = "read"
	ACPToolKindEdit     ACPToolKind = "edit"
	ACPToolKindSearch   ACPToolKind = "search"
	ACPToolKindExecute  ACPToolKind = "execute"
	ACPToolKindFetch    ACPToolKind = "fetch"
	ACPToolKindThink    ACPToolKind = "think"
)
