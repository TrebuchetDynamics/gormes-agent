package skillsslash

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

// ReloadResult is the runtime payload produced by refreshing skill slash commands.
type ReloadResult struct {
	Commands []skills.SkillSlashCommand
	Output   string
}

// ReloadFunc refreshes the skill slash catalog.
type ReloadFunc func(context.Context) (ReloadResult, error)

// ReloadDecision is the pure decision root package tui applies to its model.
type ReloadDecision struct {
	Handled  bool
	Commands []skills.SkillSlashCommand
	Rebuild  bool
	Status   string
}

// HandleReload runs the injected skill reload seam and returns the status and
// registry-rebuild decision for /reload-skills.
func HandleReload(ctx context.Context, reload ReloadFunc) ReloadDecision {
	if reload == nil {
		return ReloadDecision{Handled: true, Status: "reload-skills: skill runtime unavailable"}
	}
	result, err := reload(ctx)
	if err != nil {
		return ReloadDecision{Handled: true, Status: "reload-skills: " + err.Error()}
	}
	status := strings.TrimSpace(result.Output)
	if status == "" {
		status = fmt.Sprintf("Skills Reloaded\n%d skill(s) available", len(result.Commands))
	}
	return ReloadDecision{
		Handled:  true,
		Commands: append([]skills.SkillSlashCommand(nil), result.Commands...),
		Rebuild:  true,
		Status:   status,
	}
}
