// Package nextwork owns read-only builder-candidate selection output for the
// progress control CLI. It may inspect write scopes for filtering, but it must
// not mutate the backlog or generated artifacts.
package nextwork

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// Options controls the read-only next-work selector.
type Options struct {
	// RepoOnly filters candidates whose write_scope resolves outside root.
	RepoOnly bool
}

// NextWork emits the single next action over the canonical backlog without
// mutating it. It uses the progress active-handoff projection so command users
// and generated docs share one row classification and ordering seam.
func NextWork(stdout io.Writer, p *progress.Progress) error {
	return NextWorkWithOptions(stdout, "", p, Options{})
}

// NextWorkWithOptions emits the single next action over the canonical backlog
// without mutating it. Command-level filters apply after canonical ordering.
func NextWorkWithOptions(stdout io.Writer, root string, p *progress.Progress, opts Options) error {
	candidates := progress.ProjectActiveHandoffs(p, 0)
	var err error
	if opts.RepoOnly {
		candidates, err = filterRepoScopedCandidates(root, candidates)
		if err != nil {
			return err
		}
	}
	if len(candidates) == 0 {
		return printNoNextWork(stdout, opts)
	}

	noun := "candidates"
	if len(candidates) == 1 {
		noun = "candidate"
	}
	top := candidates[0]
	if _, err := fmt.Fprintf(stdout, "progress: next-work builder-ready (%d %s)\n", len(candidates), noun); err != nil {
		return err
	}
	fields := []struct {
		key   string
		value string
	}{
		{key: "decision", value: "build"},
	}
	if opts.RepoOnly {
		fields = append(fields, struct {
			key   string
			value string
		}{key: "scope", value: "repo"})
	}
	fields = append(fields, []struct {
		key   string
		value string
	}{
		{key: "phase", value: top.Identity.PhaseID},
		{key: "subphase", value: top.Identity.SubphaseID},
		{key: "name", value: top.Identity.ItemName},
		{key: "reason", value: selectionReason(top)},
		{key: "priority", value: top.Priority},
		{key: "status", value: top.Status},
		{key: "contract_status", value: top.ContractStatus},
		{key: "slice_size", value: top.SliceSize},
		{key: "owner", value: top.ExecutionOwner},
	}...)
	for _, field := range fields {
		if _, err := fmt.Fprintf(stdout, "%s=%s\n", field.key, field.value); err != nil {
			return err
		}
	}
	return nil
}

func printNoNextWork(stdout io.Writer, opts Options) error {
	if opts.RepoOnly {
		if _, err := fmt.Fprintln(stdout, "progress: next-work no in-repo builder-ready rows"); err != nil {
			return err
		}
		for _, line := range []string{
			"decision=plan",
			"scope=repo",
			"reason=no unblocked builder-ready rows within repo write scope",
			"planner_action=split or repair one row whose write_scope stays under the repo root",
		} {
			if _, err := fmt.Fprintln(stdout, line); err != nil {
				return err
			}
		}
		return nil
	}

	if _, err := fmt.Fprintln(stdout, "progress: next-work no builder-ready rows"); err != nil {
		return err
	}
	for _, line := range []string{
		"decision=plan",
		"reason=no unblocked builder-ready rows",
		"planner_action=repair one planned/draft row until it satisfies the handoff contract",
	} {
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}

func selectionReason(candidate progress.ActiveHandoffProjection) string {
	switch {
	case strings.EqualFold(strings.TrimSpace(candidate.Priority), "P0"):
		return "P0 handoff"
	case candidate.Status == string(progress.StatusInProgress):
		return "already active"
	case candidate.ContractStatus == string(progress.ContractStatusFixtureReady):
		return "fixture ready"
	case len(candidate.Unblocks) > 0:
		return "unblocks downstream work"
	case candidate.ContractStatus == string(progress.ContractStatusDraft):
		return "draft contract"
	case candidate.Status == string(progress.StatusPlanned):
		return "planned row"
	default:
		return "planned row"
	}
}

func filterRepoScopedCandidates(root string, candidates []progress.ActiveHandoffProjection) ([]progress.ActiveHandoffProjection, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rootAbs = filepath.Clean(rootAbs)
	out := make([]progress.ActiveHandoffProjection, 0, len(candidates))
	for _, candidate := range candidates {
		if candidateWriteScopeWithinRoot(rootAbs, candidate.WriteScope) {
			out = append(out, candidate)
		}
	}
	return out, nil
}

func candidateWriteScopeWithinRoot(rootAbs string, scopes []string) bool {
	if len(scopes) == 0 {
		return false
	}
	for _, scope := range scopes {
		if !writeScopePathWithinRoot(rootAbs, scope) {
			return false
		}
	}
	return true
}

func writeScopePathWithinRoot(rootAbs, scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	lower := strings.ToLower(scope)
	if strings.Contains(lower, "separate repo") || strings.Contains(lower, "external repo") || strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return false
	}
	candidate := scope
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(rootAbs, candidate)
	}
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(rootAbs, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
