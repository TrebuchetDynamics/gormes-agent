package interactions

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// NormalizeSkillGroupCommands returns the unique, sorted command list exposed to Discord autocomplete.
func NormalizeSkillGroupCommands(commands []gateway.PlatformCommand) ([]gateway.PlatformCommand, int) {
	out := make([]gateway.PlatformCommand, 0, len(commands))
	seen := map[string]struct{}{}
	hidden := 0
	for _, cmd := range commands {
		name := strings.TrimSpace(cmd.Name)
		if name == "" {
			hidden++
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			hidden++
			continue
		}
		seen[key] = struct{}{}
		out = append(out, gateway.PlatformCommand{
			Name:        name,
			Description: strings.TrimSpace(cmd.Description),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, hidden
}
