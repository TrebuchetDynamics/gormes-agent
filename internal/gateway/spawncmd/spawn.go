package spawncmd

import (
	"errors"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandline"
)

var ErrUsage = errors.New("usage: /spawn <name> [persona text...]")

// Command is the parsed shape of a /spawn invocation.
type Command struct {
	Name    string
	Persona string
}

// Parse extracts the agent name and optional persona text from a raw /spawn
// command line. It performs no authorization, channel, registry, or I/O work.
func Parse(raw string) (Command, error) {
	token, args := commandline.Split(raw)
	if !isSlashCommandToken(token) || commandline.Name(token) != "spawn" {
		return Command{}, ErrUsage
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return Command{}, ErrUsage
	}
	name := sanitizeName(fields[0])
	if name == "" {
		return Command{}, ErrUsage
	}
	return Command{
		Name:    name,
		Persona: strings.TrimSpace(strings.TrimPrefix(args, fields[0])),
	}, nil
}

func isSlashCommandToken(token string) bool {
	return strings.HasPrefix(token, "/") || strings.HasPrefix(token, "／")
}

func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if skipNameRune(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func skipNameRune(r rune) bool {
	if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
		return true
	}
	switch {
	case r >= 0x200b && r <= 0x200f:
		return true
	case r >= 0x2028 && r <= 0x202e:
		return true
	case r >= 0x2060 && r <= 0x2069:
		return true
	case r == 0xfeff || r == 0xfffc:
		return true
	case r >= 0xfff9 && r <= 0xfffb:
		return true
	default:
		return false
	}
}
