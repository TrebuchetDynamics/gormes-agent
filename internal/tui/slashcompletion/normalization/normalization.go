package normalization

import "strings"

// Prefix is a canonical, case-insensitive completion prefix.
type Prefix string

// Identity is the display name plus canonical lookup key for a completion item.
type Identity struct {
	Name string
	Key  string
}

// NewIdentity normalizes slash-prefixed display text and derives its lookup key.
func NewIdentity(raw string) Identity {
	name := Name(raw)
	return Identity{Name: name, Key: strings.ToLower(name)}
}

func (id Identity) Valid() bool {
	return id.Key != ""
}

func NewCommandPrefix(raw string) Prefix {
	return Prefix(NewIdentity(raw).Key)
}

func NewSubcommandPrefix(raw string) Prefix {
	return Prefix(Key(raw))
}

func (p Prefix) String() string {
	return string(p)
}

func (p Prefix) Matches(name string) bool {
	return strings.HasPrefix(Key(name), p.String())
}

func Name(name string) string {
	return TrimSlashPrefix(name)
}

func TrimSlashPrefix(name string) string {
	trimmed := strings.TrimSpace(name)
	for strings.HasPrefix(trimmed, "/") {
		trimmed = strings.TrimSpace(strings.TrimLeft(trimmed, "/"))
	}
	return trimmed
}

func Key(name string) string {
	return NewIdentity(name).Key
}
