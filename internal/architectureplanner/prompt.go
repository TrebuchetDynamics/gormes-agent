package architectureplanner

import (
	"fmt"
	"strings"
)

func BuildPrompt(bundle ContextBundle) string {
	var roots []string
	for _, root := range bundle.SourceRoots {
		status := "missing"
		if root.Exists {
			status = fmt.Sprintf("present, files=%d", root.FileCount)
		}
		roots = append(roots, fmt.Sprintf("- %s: %s (%s)", root.Name, root.Path, status))
	}

	return fmt.Sprintf(`You are the Gormes Architecture Planner Loop.

Mission:
Improve the architecture plan for building full Gormes, the Go port of Hermes, while preserving the internal goncho package direction for Honcho-compatible memory.

Planning scope:
%s

Control plane:
- Canonical progress file: %s
- Target repo: %s
- Current progress items: %d

Required behavior:
1. Study hermes-agent, gbrain, docs/content/upstream-hermes, docs/content/upstream-gbrain, docs/content/building-gormes, and Honcho/Goncho memory references.
2. Improve docs/content/building-gormes/architecture_plan/progress.json conservatively so autoloop workers receive smaller, dependency-aware, TDD-ready slices.
3. Keep GONCHO as the internal implementation name while preserving honcho_* external compatibility where the public tool contract requires it.
4. Include Goncho/Honcho tasks when they affect the full Gormes architecture.
5. Do not implement runtime feature code; planning/docs/progress work only.
6. Do not mark implementation complete without concrete repository evidence.

After edits, run:
- go run ./cmd/progress-gen -write
- go run ./cmd/progress-gen -validate
- go test ./internal/progress -count=1
- go test ./docs -count=1

Required final report sections:
1. Scope scanned
2. Architecture deltas found
3. Progress plan changes
4. Goncho/Honcho implications
5. Validation evidence
6. Recommended next autoloop tasks
7. Risks and ambiguities
`, strings.Join(roots, "\n"), bundle.ProgressJSON, bundle.RepoRoot, bundle.ProgressStats.Items)
}
