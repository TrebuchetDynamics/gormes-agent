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
	if commandline.Name(token) != "spawn" {
		return Command{}, ErrUsage
	}
	fields := strings.Fields(args)
	if len(fields) == 0 {
		return Command{}, ErrUsage
	}
	return Command{
		Name:    fields[0],
		Persona: strings.TrimSpace(strings.TrimPrefix(args, fields[0])),
	}, nil
}
