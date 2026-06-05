// Package featuremodule owns the stable feature-module taxonomy used by the
// progress backlog split and validation surfaces.
package featuremodule

import "strings"

// The grill-corrected module-split taxonomy (2026-05-16): module names are
// feature homes for the physical split, while execution_owner remains work
// ownership. Keep this list alphabetized so filenames, validation messages,
// and generated module views stay stable.
const (
	ModuleBrowser      = "browser"
	ModuleBuilder      = "builder"
	ModuleChannels     = "channels"
	ModuleCLI          = "cli"
	ModuleConfig       = "config"
	ModuleCrossCutting = "cross-cutting"
	ModuleDoctor       = "doctor"
	ModuleDocs         = "docs"
	ModuleFleet        = "fleet"
	ModuleGateway      = "gateway"
	ModuleGoncho       = "goncho"
	ModuleInstall      = "install"
	ModuleKanban       = "kanban"
	ModuleLanding      = "landing"
	ModuleLearningLoop = "learning-loop"
	ModuleMemory       = "memory"
	ModuleNavivox      = "navivox"
	ModulePlanner      = "planner"
	ModuleProfiles     = "profiles"
	ModuleProgress     = "progress"
	ModuleProviders    = "providers"
	ModuleRelease      = "release"
	ModuleRuntime      = "runtime"
	ModuleSessions     = "sessions"
	ModuleSkills       = "skills"
	ModuleSTT          = "stt"
	ModuleTools        = "tools"
	ModuleTTS          = "tts"
	ModuleTUI          = "tui"

	// ModuleUnclassified is the explicit deterministic bucket for rows that
	// carry no execution_owner and no explicit module. It is a compatibility
	// fallback only, not a valid final physical module for C5.
	ModuleUnclassified = "unclassified"
)

var allowedModules = []string{
	ModuleBrowser,
	ModuleBuilder,
	ModuleChannels,
	ModuleCLI,
	ModuleConfig,
	ModuleCrossCutting,
	ModuleDoctor,
	ModuleDocs,
	ModuleFleet,
	ModuleGateway,
	ModuleGoncho,
	ModuleInstall,
	ModuleKanban,
	ModuleLanding,
	ModuleLearningLoop,
	ModuleMemory,
	ModuleNavivox,
	ModulePlanner,
	ModuleProfiles,
	ModuleProgress,
	ModuleProviders,
	ModuleRelease,
	ModuleRuntime,
	ModuleSessions,
	ModuleSkills,
	ModuleSTT,
	ModuleTools,
	ModuleTTS,
	ModuleTUI,
}

var allowedModuleSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(allowedModules))
	for _, module := range allowedModules {
		set[module] = struct{}{}
	}
	return set
}()

// Allowed returns the closed, stable feature-module taxonomy. The returned
// slice is a copy.
func Allowed() []string {
	return append([]string(nil), allowedModules...)
}

// Valid reports whether module is one of the approved physical feature
// buckets. Empty strings are intentionally invalid here; progress validation
// treats them separately as "not assigned yet" until C5g makes explicit
// modules universal.
func Valid(module string) bool {
	_, ok := allowedModuleSet[module]
	return ok
}

// ForExecutionOwner maps the coarse execution_owner enum value to a
// compatibility taxonomy. An empty string means "not derivable from owner
// alone".
func ForExecutionOwner(owner string) string {
	switch owner {
	case "docs":
		return ModuleDocs
	case "gateway":
		return ModuleGateway
	case "memory":
		return ModuleMemory
	case "skills":
		return ModuleSkills
	case "goncho":
		return ModuleGoncho
	case "provider":
		return ModuleProviders
	case "tools":
		return ModuleTools
	case "orchestrator":
		return ModuleFleet
	case "tui":
		return ModuleTUI
	default:
		return ""
	}
}

// DisplayName renders a feature-module key as an operator-facing heading.
func DisplayName(module string) string {
	switch module {
	case ModuleCLI, ModuleSTT, ModuleTTS, ModuleTUI:
		return strings.ToUpper(module)
	}
	parts := strings.Split(module, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
